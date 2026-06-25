package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lucy/internal/convert"
	"lucy/internal/gemini"
	"lucy/internal/schema"
	"lucy/internal/store"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const generateTimeout = 90 * time.Second

// fieldTypes are the types offered in the visual builder's type dropdowns.
var fieldTypes = []string{"string", "integer", "number", "boolean", "array", "object"}

type indexData struct {
	Models       []gemini.ModelInfo
	DefaultModel string
	Formats      []string
	Types        []string
	Collections  []store.Collection
}

type resultData struct {
	Error          string
	Format         string
	Count          int
	Filename       string
	ContentType    string
	Output         string
	CollectionID   string
	CollectionName string
	Tag            string
	Inserted       int
	AutoCommit     bool
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	collections, _ := s.store.ListCollections(r.Context())
	s.render(w, "index.html", indexData{
		Models:       s.models,
		DefaultModel: s.defaultModel,
		Formats:      convert.Formats(),
		Types:        fieldTypes,
		Collections:  collections,
	})
}

func (s *Server) handleListCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := s.store.ListCollections(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collections)
}

func (s *Server) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := bsonIDFromHex(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	raw, err := s.store.GetSchema(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if raw == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(raw)
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

	rctx := r.Context()
	collectionID, collectionName := s.resolveCollection(rctx, r)
	tag := strings.TrimSpace(r.FormValue("tag"))
	autoCommit := r.FormValue("auto_commit") == "1"

	inserted := 0
	if collectionID != (bson.ObjectID{}) {
		if err := s.store.UpsertSchema(rctx, collectionID, json.RawMessage(schemaStr)); err != nil {
			log.Printf("upsert schema: %v", err)
		}
		if autoCommit {
			n, err := s.store.InsertItems(rctx, collectionName, jsonBytes, tag)
			if err != nil {
				log.Printf("insert items: %v", err)
			} else {
				inserted = n
			}
		}
	}

	s.renderResult(w, resultData{
		Format:         format,
		Count:          countItems(jsonBytes),
		Filename:       "lucy-output." + res.Ext,
		ContentType:    res.ContentType,
		Output:         string(res.Data),
		CollectionID:   collectionID.Hex(),
		CollectionName: collectionName,
		Tag:            tag,
		Inserted:       inserted,
		AutoCommit:     autoCommit,
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

func (s *Server) handleCommit(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CollectionID   string            `json:"collection_id"`
		CollectionName string            `json:"collection_name"`
		Tag            string            `json:"tag"`
		Items          []json.RawMessage `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.CollectionName == "" {
		http.Error(w, "collection_name required", http.StatusBadRequest)
		return
	}
	if len(body.Items) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"inserted": 0})
		return
	}

	// Re-marshal the selected items as an array for InsertItems.
	itemsJSON, err := json.Marshal(body.Items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	n, err := s.store.InsertItems(r.Context(), body.CollectionName, itemsJSON, body.Tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"inserted": n})
}

func bsonIDFromHex(s string) (bson.ObjectID, error) {
	return bson.ObjectIDFromHex(s)
}

// resolveCollectionName resolves the path's {id} to a registered collection
// name and writes the appropriate HTTP error if it can't.
func (s *Server) resolveCollectionName(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, err := bsonIDFromHex(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid collection id", http.StatusBadRequest)
		return "", false
	}
	c, err := s.store.GetCollection(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return "", false
	}
	if c == nil {
		http.Error(w, "collection not found", http.StatusNotFound)
		return "", false
	}
	return c.Name, true
}

func (s *Server) handleListItems(w http.ResponseWriter, r *http.Request) {
	name, ok := s.resolveCollectionName(w, r)
	if !ok {
		return
	}

	limit := int64(0)
	if v, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("limit")), 10, 64); err == nil {
		limit = v
	}

	items, err := s.store.ListItems(r.Context(), name, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert _id (bson.ObjectID) to hex so the client can address rows.
	out := make([]map[string]any, 0, len(items))
	for _, doc := range items {
		if oid, ok := doc["_id"].(bson.ObjectID); ok {
			doc["_id"] = oid.Hex()
		}
		out = append(out, doc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	name, ok := s.resolveCollectionName(w, r)
	if !ok {
		return
	}
	itemID, err := bsonIDFromHex(r.PathValue("itemId"))
	if err != nil {
		http.Error(w, "invalid item id", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteItem(r.Context(), name, itemID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "item not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	name, ok := s.resolveCollectionName(w, r)
	if !ok {
		return
	}
	itemID, err := bsonIDFromHex(r.PathValue("itemId"))
	if err != nil {
		http.Error(w, "invalid item id", http.StatusBadRequest)
		return
	}

	var fields map[string]any
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateItem(r.Context(), name, itemID, fields); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "item not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// resolveCollection returns the ObjectID and name of the target collection
// (existing or newly created). Returns zero-value ID if no collection was
// selected or if creation fails.
func (s *Server) resolveCollection(ctx context.Context, r *http.Request) (bson.ObjectID, string) {
	if idStr := strings.TrimSpace(r.FormValue("collection_id")); idStr != "" {
		id, err := bson.ObjectIDFromHex(idStr)
		if err != nil {
			return bson.ObjectID{}, ""
		}
		collections, err := s.store.ListCollections(ctx)
		if err != nil {
			return bson.ObjectID{}, ""
		}
		for _, c := range collections {
			if c.ID == id {
				return id, c.Name
			}
		}
		return bson.ObjectID{}, ""
	}

	if name := strings.TrimSpace(r.FormValue("new_collection")); name != "" {
		id, err := s.store.EnsureCollection(ctx, name)
		if err != nil {
			log.Printf("ensure collection %q: %v", name, err)
			return bson.ObjectID{}, ""
		}
		return id, name
	}

	return bson.ObjectID{}, ""
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
