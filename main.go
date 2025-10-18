package main

import (
	"bytes"
	"compress/gzip"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

const (
	profileFallback = "https://raw.githubusercontent.com/lucasrattz/rattz.xyz/main/profile.json"
)

var gzipExtensions = map[string]bool{
	".css": true,
	".svg": true,
}

func main() {
	host, port := os.Getenv("HOST"), os.Getenv("PORT")
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5675"
	}

	if os.Getenv("REMOTE_PROFILE_URL") == "" {
		slog.Warn("Remote profile URL not set, fallback is " + profileFallback)
	}

	subFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	tmpl := template.Must(template.ParseGlob("templates/*.go.html"))

	codex, err := newCodex()
	if err != nil {
		log.Fatal(err)
	}

	gallery, err := newGallery()
	if err != nil {
		log.Fatal(err)
	}

	if err := updateGallery(gallery); err != nil {
		slog.Error("Failed to populate gallery on startup:", "err", err)
	}

	conn := fmt.Sprint(host, ":", port)

	router := http.NewServeMux()
	router.HandleFunc("/", gzipHandler(func(w http.ResponseWriter, r *http.Request) {
		indexHandler(w, r, tmpl)
	}))
	router.HandleFunc("/update/", func(w http.ResponseWriter, r *http.Request) { updateHandler(w, r, gallery) })
	router.Handle("/cefetdb/", http.RedirectHandler("https://cefetdb.rattz.xyz", http.StatusFound))
	router.Handle("/static/", gzipFileServer("/static/", http.FS(subFS)))

	router.HandleFunc("/codex/", gzipHandler(codex.codexHandler))
	router.HandleFunc("/codex/{id}", gzipHandler(codex.codexHandler))
	router.HandleFunc("/codex/pics", gzipHandler(gallery.galleryHandler))
	router.HandleFunc("/codex/pics/{fileName}", gzipHandler(gallery.galleryHandler))

	slog.Info("Server running on " + "http://" + conn)
	log.Fatal(http.ListenAndServe(conn, router))
}

func gzipHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !acceptsGzip(r) {
			handler(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()

		gzw := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		handler(gzw, r)
	}
}

func gzipFileServer(prefix string, fs http.FileSystem) http.Handler {
	return http.StripPrefix(prefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 month
		if acceptsGzip(r) && shouldGzip(r.URL.Path) {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			defer gz.Close()

			gzw := gzipResponseWriter{Writer: gz, ResponseWriter: w}
			http.FileServer(fs).ServeHTTP(gzw, r)
		} else {
			http.FileServer(fs).ServeHTTP(w, r)
		}
	}))
}

func shouldGzip(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return gzipExtensions[ext]
}

func acceptsGzip(r *http.Request) bool {
	return r.Header.Get("Accept-Encoding") != "" &&
		strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer *gzip.Writer
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func indexHandler(w http.ResponseWriter, r *http.Request, tmpl *template.Template) {
	if r.URL.Path != "/" {
		errorHandler(w, r, http.StatusNotFound, tmpl)
		return
	}

	var buf bytes.Buffer

	p, err := getProfile()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error loading profile: " + err.Error())
		return
	}

	err = tmpl.ExecuteTemplate(&buf, "index", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error executing template: " + err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	_, err = buf.WriteTo(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error writing html to buffer: " + err.Error())
		return
	}
}

func updateHandler(w http.ResponseWriter, r *http.Request, g *Gallery) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	remoteProfile := os.Getenv("REMOTE_PROFILE_URL")
	if remoteProfile == "" {
		remoteProfile = profileFallback
	}

	res, err := http.Get(remoteProfile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error getting remote profile object: " + err.Error())
		return
	}

	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error reading profile object: " + err.Error())
		return
	}

	if !json.Valid([]byte(b)) {
		http.Redirect(w, r, "/", http.StatusFound)
		slog.Warn("Invalid JSON found when updating profile")
		return
	}

	err = os.WriteFile("./profile.json", b, 0o644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error writing profile object to disk: " + err.Error())
		return
	}

	slog.Info("Updated profile, request from " + r.RemoteAddr)

	if err := updateGallery(g); err != nil {
		slog.Error("Failed to update gallery:", "err", err)
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func errorHandler(w http.ResponseWriter, r *http.Request, status int, tmpl *template.Template) {
	w.WriteHeader(status)
	if status == http.StatusNotFound {
		var buf bytes.Buffer

		p, err := getProfile()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			slog.Error("Error loading profile: " + err.Error())
			return
		}

		err = tmpl.ExecuteTemplate(&buf, "404", p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			slog.Error("Error executing template: " + err.Error())
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		_, err = buf.WriteTo(w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			slog.Error("Error writing html to buffer: " + err.Error())
			return
		}
	} else {
		slog.Error("Error " + http.StatusText(status) + " in " + r.URL.Path)
		http.Error(w, http.StatusText(status), status)
	}
}
