// Package server exposes the HTTP UI and the generation endpoint.
package server

import (
	"html/template"
	"io/fs"
	"net/http"

	"lucy/internal/config"
	"lucy/internal/gemini"
	"lucy/web"
)

// Server holds the dependencies shared across HTTP handlers.
type Server struct {
	cfg  config.Config
	gem  *gemini.Client
	tmpl *template.Template
	mux  *http.ServeMux
}

// New parses the embedded templates, wires routes, and returns a ready Server.
func New(cfg config.Config, gem *gemini.Client) (*Server, error) {
	tmpl, err := template.ParseFS(web.Files, "templates/*.html")
	if err != nil {
		return nil, err
	}

	s := &Server{cfg: cfg, gem: gem, tmpl: tmpl, mux: http.NewServeMux()}

	staticFS, err := fs.Sub(web.Files, "static")
	if err != nil {
		return nil, err
	}

	s.mux.HandleFunc("GET /{$}", s.handleIndex)
	s.mux.HandleFunc("POST /generate", s.handleGenerate)
	s.mux.HandleFunc("GET /builder/row", s.handleRow)
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	return s, nil
}

// Handler returns the router for use by http.Server.
func (s *Server) Handler() http.Handler { return s.mux }
