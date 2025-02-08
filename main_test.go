package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkIndexHandler(b *testing.B) {
	tmpl := template.Must(template.ParseGlob("templates/*.go.html"))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	for i := 0; i < b.N; i++ {
		indexHandler(w, r, tmpl)
	}
}
