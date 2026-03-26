package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"sync/atomic"
)

var profileCache atomic.Value

type Profile struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Pic         string    `json:"profilePicture"`
	Favicon     string    `json:"favicon"`
	Render      bool      `json:"render"`
	Sections    []Section `json:"sections"`
}

type Section struct {
	Title   string  `json:"title"`
	Kind    Kind    `json:"kind"`
	Entries []Entry `json:"entries"`
}

type Entry struct {
	Timeframe   string   `json:"timeframe"`
	Icon        string   `json:"projectIcon"`
	Name        string   `json:"entryName"`
	Description string   `json:"entryDescription"`
	Url         string   `json:"url"`
	Stack       []string `json:"stack"`
	RelMe       bool     `json:"relMe"`
}

type Kind int

const (
	About Kind = iota
	Experience
	Education
	Projects
	Links
)

func (k Kind) Kind() string {
	switch k {
	case About:
		return "About"
	case Experience:
		return "Experience"
	case Education:
		return "Education"
	case Projects:
		return "Projects"
	case Links:
		return "Links"
	}

	return "Unknown"
}

func (p *Profile) FromJson(path string) (*Profile, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(file, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func getProfile() (Profile, error) {
	if p, ok := profileCache.Load().(*Profile); ok {
		return *p, nil
	}

	loadedProfile, err := new(Profile).FromJson("./profile.json")
	if err != nil {
		return Profile{}, err
	}

	profileCache.Store(loadedProfile)

	return *loadedProfile, nil
}

func profileHandler(w http.ResponseWriter, r *http.Request, tmpl *template.Template) {
	if r.URL.Path != "/profile/" {
		profileErrorHandler(w, r, http.StatusNotFound, tmpl)
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

func profileErrorHandler(w http.ResponseWriter, r *http.Request, status int, tmpl *template.Template) {
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
