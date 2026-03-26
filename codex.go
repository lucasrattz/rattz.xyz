package main

import (
	"bytes"
	"errors"
	"html/template"
	"net/http"
)

type Codex struct {
	Scriptum  *Scriptum
	Gallery   *Gallery
	indexTmpl *template.Template
}

func newCodex(s *Scriptum, g *Gallery) (*Codex, error) {
	indexTmpl, err := template.ParseGlob("templates/*.go.html")
	if err != nil {
		return nil, errors.New("error parsing codex template: " + err.Error())
	}

	return &Codex{
		Scriptum:  s,
		Gallery:   g,
		indexTmpl: indexTmpl,
	}, nil
}

func (c *Codex) codexHandler(w http.ResponseWriter, r *http.Request) {
	var latestPost Page
	if c.Scriptum != nil {
		latestPost = c.Scriptum.Pages[0]
	}

	var dailyImage Image
	if c.Gallery != nil {
		dailyImage, _ = c.Gallery.ImageOfTheDay()
	}

	data := struct {
		LatestPost Page
		DailyImage Image
	}{
		LatestPost: latestPost,
		DailyImage: dailyImage,
	}

	var buf bytes.Buffer
	err := c.indexTmpl.ExecuteTemplate(&buf, "codex", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	buf.WriteTo(w)
}
