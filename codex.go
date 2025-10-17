package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type Codex struct {
	indexTmpl *template.Template
	pageTmpl  *template.Template
	Pages     []Page
}

type Page struct {
	Title string
	Date  string
	Desc  string
	File  string
	Slug  string
}

func newCodex() (*Codex, error) {
	indexTmpl, err := template.ParseGlob("codex/*.go.html")
	if err != nil {
		return nil, errors.New("error parsing codex index template: " + err.Error())
	}

	pageTmpl, err := template.ParseGlob("codex/pages/*.go.html")
	if err != nil {
		return nil, errors.New("error parsing codex page templates: " + err.Error())
	}

	pages, err := loadCodexPages()
	if err != nil {
		return nil, errors.New("error loading codex pages: " + err.Error())
	}

	return &Codex{
		Pages:     pages,
		indexTmpl: indexTmpl,
		pageTmpl:  pageTmpl,
	}, nil
}

func (c *Codex) codexHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var buf bytes.Buffer
	var err error

	if id == "" {
		err = c.indexTmpl.ExecuteTemplate(&buf, "codex", c)
	} else {
		err = c.pageTmpl.ExecuteTemplate(&buf, id, nil)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error rendering Codex", "err", err, "id", id)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")

	buf.WriteTo(w)
}

func loadCodexPages() ([]Page, error) {
	pageFiles, err := os.ReadDir("codex/pages")
	if err != nil {
		return []Page{}, errors.New("error listing page files: " + err.Error())
	}

	var pages []Page
	for _, f := range pageFiles {
		name := f.Name()

		fm, err := readFrontmatter("codex/pages/" + name)
		if err != nil {
			slog.Error("error reading frontmatter of " + name + ": " + err.Error())
			continue
		}

		err = validateFrontmatter(fm)
		if err != nil {
			slog.Error(name + " has invalid frontmatter: " + err.Error())
			continue
		}

		slug := strings.Split(name, ".go.html")[0]

		page := Page{Title: fm["title"], Date: fm["date"], Desc: fm["desc"], File: name, Slug: slug}
		pages = append(pages, page)
	}

	sort.Slice(pages, func(i, j int) bool {
		t1, err1 := time.Parse("2006-01-02", pages[i].Date)
		t2, err2 := time.Parse("2006-01-02", pages[j].Date)
		if err1 != nil || err2 != nil {
			return false
		}
		return t1.After(t2)
	})

	return pages, nil
}

func readFrontmatter(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	frontmatter := make(map[string]string)

	if !scanner.Scan() {
		return nil, errors.New("file is empty")
	}

	if strings.TrimSpace(scanner.Text()) != "---" {
		return nil, errors.New("bad formatting: frontmatter must be at the top")
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "---" {
			break
		}

		if line == "" {
			continue
		}

		var key, value string
		if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
			key = strings.TrimSpace(parts[0])
			value = strings.TrimSpace(parts[1])
		} else if parts := strings.Fields(line); len(parts) >= 2 {
			key = parts[0]
			value = strings.Join(parts[1:], " ")
		} else {
			return nil, errors.New("invalid frontmatter line: " + line)
		}

		frontmatter[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return frontmatter, nil
}

func validateFrontmatter(fm map[string]string) error {
	if len(fm) != 3 {
		return fmt.Errorf("frontmatter must contain exactly 'title', 'date' and 'desc'")
	}

	title, okTitle := fm["title"]
	date, okDate := fm["date"]
	desc, okDesc := fm["desc"]

	if !okTitle || strings.TrimSpace(title) == "" {
		return fmt.Errorf("missing or empty 'title'")
	}
	if !okDate || strings.TrimSpace(date) == "" {
		return fmt.Errorf("missing or empty 'date'")
	}
	if !okDesc || strings.TrimSpace(desc) == "" {
		return fmt.Errorf("missing or empty 'desc'")
	}

	return nil
}
