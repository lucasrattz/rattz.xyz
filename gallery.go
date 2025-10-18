package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	galleryURL  = "https://api.github.com/repos/lucasrattz/rattz.xyz/contents/gallery/content"
	galleryPath = "gallery"
	cacheDir    = galleryPath + "/cache"
	cacheFile   = cacheDir + "/cache.json"
)

type galleryCache map[string]string

type githubFile struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
	SHA         string `json:"sha"`
}

type Image struct {
	Filename    string `json:"filename"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Date        string `json:"date"`
}

type Gallery struct {
	mu        sync.RWMutex
	Images    []Image
	indexTmpl *template.Template
}

func newGallery() (*Gallery, error) {
	tmpl, err := template.ParseGlob(galleryPath + "/*.go.html")
	if err != nil {
		return nil, fmt.Errorf("error parsing gallery templates: %w", err)
	}

	g := &Gallery{
		indexTmpl: tmpl,
	}

	if err := g.loadFromDisk(); err != nil {
		slog.Warn("failed to load gallery from disk", "err", err)
	}

	return g, nil
}

func (g *Gallery) loadFromDisk() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	entries := []Image{}

	err := filepath.WalkDir(cacheDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".rio") {
			return nil
		}

		meta, _, err := decodeBinFile(path)
		if err != nil {
			slog.Warn("failed to decode gallery file", "path", path, "err", err)
			return nil
		}

		entries = append(entries, meta)
		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking gallery cache: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		t1, err1 := time.Parse("2006-01-02", entries[i].Date)
		t2, err2 := time.Parse("2006-01-02", entries[j].Date)
		if err1 != nil || err2 != nil {
			return false
		}
		return t1.After(t2)
	})

	g.Images = entries
	return nil
}

func decodeBinFile(path string) (Image, []byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return Image{}, nil, err
	}
	defer f.Close()

	var metaLen uint32
	if err := binary.Read(f, binary.LittleEndian, &metaLen); err != nil {
		return Image{}, nil, err
	}

	metaBytes := make([]byte, metaLen)
	if _, err := io.ReadFull(f, metaBytes); err != nil {
		return Image{}, nil, err
	}

	var meta Image
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return Image{}, nil, err
	}

	var imgLen uint32
	if err := binary.Read(f, binary.LittleEndian, &imgLen); err != nil {
		return meta, nil, err
	}

	imgBytes := make([]byte, imgLen)
	if _, err := io.ReadFull(f, imgBytes); err != nil {
		return meta, nil, err
	}

	return meta, imgBytes, nil
}

func (g *Gallery) galleryHandler(w http.ResponseWriter, r *http.Request) {
	fileName := r.PathValue("fileName")

	g.mu.RLock()
	defer g.mu.RUnlock()

	if fileName == "" {
		var buf bytes.Buffer
		err := g.indexTmpl.ExecuteTemplate(&buf, "gallery", g.Images)
		if err != nil {
			slog.Error("failed to render gallery", "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		buf.WriteTo(w)
		return
	}

	path := filepath.Join(cacheDir, fileName)
	if !strings.HasSuffix(fileName, ".rio") || !fileExists(path) {
		http.NotFound(w, r)
		return
	}

	_, imgBytes, err := decodeBinFile(path)
	if err != nil {
		slog.Error("failed to read image", "path", path, "err", err)
		http.Error(w, "Failed to read image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/webp")
	w.Write(imgBytes)
}

func updateGallery(g *Gallery) error {
	files, err := listGithubFiles()
	if err != nil {
		return err
	}

	os.MkdirAll(cacheDir, 0o755)
	cache, _ := loadCache(cacheFile)

	currentFiles := map[string]bool{}

	for _, f := range files {
		if !strings.HasSuffix(f.Name, ".rio") {
			continue
		}
		currentFiles[f.Name] = true

		destPath := filepath.Join(cacheDir, f.Name)
		if cache[f.Name] != f.SHA {
			if err := downloadFile(f.DownloadURL, destPath); err != nil {
				slog.Error("failed to download file", "file", f.Name, "err", err)
				continue
			}
			cache[f.Name] = f.SHA
		}
	}

	diskFiles, _ := os.ReadDir(cacheDir)
	for _, f := range diskFiles {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".rio") {
			continue
		}
		if !currentFiles[f.Name()] {
			os.Remove(filepath.Join(cacheDir, f.Name()))
			delete(cache, f.Name())
			slog.Info("removed deleted file", "file", f.Name())
		}
	}

	if err := saveCache(cacheFile, cache); err != nil {
		slog.Warn("failed to save cache", "err", err)
	}

	return g.loadFromDisk()
}

func loadCache(path string) (galleryCache, error) {
	cache := galleryCache{}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cache, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(b, &cache); err != nil {
		return nil, err
	}
	return cache, nil
}

func saveCache(path string, cache galleryCache) error {
	b, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func listGithubFiles() ([]githubFile, error) {
	resp, err := http.Get(galleryURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var files []githubFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, err
	}

	return files, nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
