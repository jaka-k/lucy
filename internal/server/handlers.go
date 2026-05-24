package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lucy/internal/convert"
	"lucy/internal/gemini"
	"lucy/internal/schema"
)

const generateTimeout = 90 * time.Second

// fieldTypes are the types offered in the visual builder's type dropdowns.
var fieldTypes = []string{"string", "integer", "number", "boolean", "array", "object"}

type indexData struct {
	Models       []gemini.ModelInfo
	DefaultModel string
	Formats      []string
	Types        []string
}

type resultData struct {
	Error       string
	Format      string
	Count       int
	Filename    string
	ContentType string
	Output      string
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.render(w, "index.html", indexData{
		Models:       s.models,
		DefaultModel: s.defaultModel,
		Formats:      convert.Formats(),
		Types:        fieldTypes,
	})
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderResult(w, resultData{Error: "could not parse form data"})
		return
	}

	prompt := strings.TrimSpace(r.FormValue("prompt"))
	if prompt == "" {
		s.renderResult(w, resultData{Error: "prompt is required"})
		return
	}

	format := r.FormValue("format")
	if !validFormat(format) {
		format = "json"
	}

	count := 0
	if c, err := strconv.Atoi(strings.TrimSpace(r.FormValue("count"))); err == nil && c > 0 {
		count = c
	}

	schemaStr, err := s.schemaFromForm(r)
	if err != nil {
		s.renderResult(w, resultData{Error: err.Error()})
		return
	}

	respSchema, err := schema.Build(schemaStr, count)
	if err != nil {
		s.renderResult(w, resultData{Error: err.Error()})
		return
	}

	model := strings.TrimSpace(r.FormValue("model"))
	if !s.knownModel(model) {
		model = s.defaultModel
	}

	ctx, cancel := context.WithTimeout(r.Context(), generateTimeout)
	defer cancel()

	jsonBytes, err := s.gem.GenerateList(ctx, model, prompt, respSchema)
	if err != nil {
		s.renderResult(w, resultData{Error: err.Error()})
		return
	}

	res, err := convert.Convert(jsonBytes, format)
	if err != nil {
		s.renderResult(w, resultData{Error: err.Error()})
		return
	}

	s.renderResult(w, resultData{
		Format:      format,
		Count:       countItems(jsonBytes),
		Filename:    "lucy-output." + res.Ext,
		ContentType: res.ContentType,
		Output:      string(res.Data),
	})
}

// schemaFromForm returns the JSON Schema string: pasted raw, or the schema the
// visual builder serialized client-side into builder_schema.
func (s *Server) schemaFromForm(r *http.Request) (string, error) {
	if r.FormValue("schema_mode") == "raw" {
		return r.FormValue("raw_schema"), nil
	}
	return r.FormValue("builder_schema"), nil
}

func countItems(b []byte) int {
	var arr []json.RawMessage
	if err := json.Unmarshal(b, &arr); err == nil {
		return len(arr)
	}
	return 1
}

func validFormat(f string) bool {
	for _, v := range convert.Formats() {
		if v == f {
			return true
		}
	}
	return false
}

func (s *Server) knownModel(id string) bool {
	for _, m := range s.models {
		if m.ID == id {
			return true
		}
	}
	return false
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) renderResult(w http.ResponseWriter, data resultData) {
	s.render(w, "result", data)
}
