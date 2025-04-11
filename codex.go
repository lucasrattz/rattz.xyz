package main

import (
	"bytes"
	"net/http"
	"text/template"
)

func codexHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id != "" {
		w.Write([]byte("Olá haha"))
		return
	}

	var buf bytes.Buffer

	tmpl := template.Must(template.ParseGlob("codex/*.go.html"))
	_ = tmpl.ExecuteTemplate(&buf, "codex", &Profile{})

	buf.WriteTo(w)
}
