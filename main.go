package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
)

type fileSystem struct {
	fs http.FileSystem
}

const (
	profileFallback = "https://raw.githubusercontent.com/lucasrattz/rattz.xyz/main/profile.json"
)

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

	tmpl := template.Must(template.ParseGlob("templates/*.go.html"))

	conn := fmt.Sprint(host, ":", port)

	router := http.NewServeMux()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { indexHandler(w, r, tmpl) })
	router.HandleFunc("/update/", updateHandler)
	router.Handle("/cefetdb/", http.RedirectHandler("https://cefetdb.rattz.xyz", http.StatusFound))
	router.Handle("/static/", http.StripPrefix("/static/", http.FileServer(fileSystem{http.Dir("./static")})))

	slog.Info("Server running on " + conn)
	log.Fatal(http.ListenAndServe(conn, router))
}

func indexHandler(w http.ResponseWriter, r *http.Request, tmpl *template.Template) {
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

	slog.Info("Index served to " + r.RemoteAddr)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
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

	http.Redirect(w, r, "/", http.StatusFound)
	slog.Info("Updated profile, request from " + r.RemoteAddr)
}

func (fs fileSystem) Open(path string) (http.File, error) {
	f, err := fs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if err != nil {
		closeErr := f.Close()
		if closeErr != nil {
			return nil, closeErr
		}
	}

	if s.IsDir() {
		return fs.fs.Open("404")
	}

	return f, nil
}
