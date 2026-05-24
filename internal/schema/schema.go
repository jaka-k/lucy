// Package schema parses a user-supplied JSON Schema and turns it into the
// *genai.Schema that Gemini uses for structured output.
//
// The input may describe either the whole output (top-level type "array") or a
// single list item (any other type). In the latter case the item schema is
// wrapped in an array so the model returns a list. An optional count constrains
// the array length.
package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// Build parses a JSON Schema string, auto-detects whether it describes a list
// or a single item, and returns the response schema to hand to Gemini.
//
// If count > 0 it is applied as the array's exact length (min == max). Build
// always returns an ARRAY schema so generation produces a list.
func Build(raw string, count int) (*genai.Schema, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("schema is empty")
	}

	var rs rawSchema
	dec := json.NewDecoder(strings.NewReader(raw))
	if err := dec.Decode(&rs); err != nil {
		return nil, fmt.Errorf("invalid JSON Schema: %w", err)
	}

	root := rs.toGenai()
	if root.Type == "" && len(root.Properties) == 0 && root.Items == nil {
		return nil, fmt.Errorf("schema has no type, properties, or items")
	}

	// Auto-detect: an array schema is used as-is; anything else is treated as a
	// single item and wrapped so the model returns a list of them.
	list := root
	if root.Type != genai.TypeArray {
		list = &genai.Schema{Type: genai.TypeArray, Items: root}
	}

	if count > 0 {
		n := int64(count)
		list.MinItems = &n
		list.MaxItems = &n
	}
	return list, nil
}

// rawSchema mirrors the subset of JSON Schema we support. Properties uses a
// custom decoder so declaration order is preserved (see orderedProps).
type rawSchema struct {
	Type        any          `json:"type"`
	Description string       `json:"description"`
	Format      string       `json:"format"`
	Enum        []any        `json:"enum"`
	Items       *rawSchema   `json:"items"`
	Properties  orderedProps `json:"properties"`
	Required    []string     `json:"required"`
	Nullable    *bool        `json:"nullable"`
	MinItems    *int64       `json:"minItems"`
	MaxItems    *int64       `json:"maxItems"`
	MinLength   *int64       `json:"minLength"`
	MaxLength   *int64       `json:"maxLength"`
	Minimum     *float64     `json:"minimum"`
	Maximum     *float64     `json:"maximum"`
}

// orderedProps preserves the declaration order of a JSON Schema "properties"
// object, which a plain map[string]... would discard.
type orderedProps struct {
	keys   []string
	values map[string]*rawSchema
}

func (o *orderedProps) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return fmt.Errorf("properties must be an object")
	}
	o.values = map[string]*rawSchema{}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return err
		}
		key := keyTok.(string)
		var sub rawSchema
		if err := dec.Decode(&sub); err != nil {
			return err
		}
		o.keys = append(o.keys, key)
		o.values[key] = &sub
	}
	return nil
}
