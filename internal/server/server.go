// Package server exposes the HTTP UI and the generation endpoint.
package server

import (
	"context"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"

	"lucy/internal/gemini"
	"lucy/internal/store"
	"lucy/web"
)

const listModelsTimeout = 15 * time.Second

// Server holds the dependencies shared across HTTP handlers.
type Server struct {
	gem          *gemini.Client
	store        *store.Store
	tmpl         *template.Template
	mux          *http.ServeMux
	models       []gemini.ModelInfo
	defaultModel string
}

// New parses the embedded templates, fetches the available models, wires
// routes, and returns a ready Server.
func New(ctx context.Context, gem *gemini.Client, st *store.Store) (*Server, error) {
	tmpl, err := template.ParseFS(web.Files, "templates/*.html")
	if err != nil {
		return nil, err
	}

	models := loadModels(ctx, gem)

	s := &Server{
		gem:          gem,
		store:        st,
		tmpl:         tmpl,
		mux:          http.NewServeMux(),
		models:       models,
		defaultModel: pickDefault(models),
	}

	staticFS, err := fs.Sub(web.Files, "static")
	if err != nil {
		return nil, err
	}

	s.mux.HandleFunc("GET /{$}", s.handleIndex)
	s.mux.HandleFunc("POST /generate", s.handleGenerate)
	s.mux.HandleFunc("GET /collections", s.handleListCollections)
	s.mux.HandleFunc("GET /collections/{id}/schema", s.handleGetSchema)
	s.mux.HandleFunc("POST /commit", s.handleCommit)
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	return s, nil
}

// Handler returns the router for use by http.Server.
func (s *Server) Handler() http.Handler { return s.mux }

// loadModels fetches generate-capable models from Gemini, falling back to a
// small static list if the call fails so the UI still works offline.
func loadModels(ctx context.Context, gem *gemini.Client) []gemini.ModelInfo {
	lctx, cancel := context.WithTimeout(ctx, listModelsTimeout)
	defer cancel()

	models, err := gem.ListModels(lctx)
	if err != nil || len(models) == 0 {
		log.Printf("could not list Gemini models (%v); using fallback list", err)
		return fallbackModels()
	}
	log.Printf("loaded %d Gemini models", len(models))
	return models
}

func fallbackModels() []gemini.ModelInfo {
	return []gemini.ModelInfo{
		{ID: "gemini-2.5-flash", DisplayName: "Gemini 2.5 Flash"},
		{ID: "gemini-2.5-pro", DisplayName: "Gemini 2.5 Pro"},
		{ID: "gemini-2.0-flash", DisplayName: "Gemini 2.0 Flash"},
	}
}

func pickDefault(models []gemini.ModelInfo) string {
	const preferred = "gemini-2.5-flash"
	for _, m := range models {
		if m.ID == preferred {
			return preferred
		}
	}
	if len(models) > 0 {
		return models[0].ID
	}
	return preferred
}
