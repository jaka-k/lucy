package schema

import (
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// toGenai converts the parsed JSON Schema into a *genai.Schema, preserving
// property declaration order via PropertyOrdering.
func (r *rawSchema) toGenai() *genai.Schema {
	if r == nil {
		return nil
	}

	typ, nullable := normalizeType(r.Type)

	s := &genai.Schema{
		Type:        typ,
		Description: r.Description,
		Format:      r.Format,
		Required:    r.Required,
		Nullable:    r.Nullable,
		MinItems:    r.MinItems,
		MaxItems:    r.MaxItems,
		MinLength:   r.MinLength,
		MaxLength:   r.MaxLength,
		Minimum:     r.Minimum,
		Maximum:     r.Maximum,
	}
	if nullable && s.Nullable == nil {
		t := true
		s.Nullable = &t
	}

	if r.Items != nil {
		s.Items = r.Items.toGenai()
	}

	if len(r.Enum) > 0 {
		s.Enum = make([]string, 0, len(r.Enum))
		for _, e := range r.Enum {
			s.Enum = append(s.Enum, fmt.Sprint(e))
		}
		if s.Type == genai.TypeString && s.Format == "" {
			s.Format = "enum"
		}
	}

	if len(r.Properties.keys) > 0 {
		s.Properties = make(map[string]*genai.Schema, len(r.Properties.keys))
		s.PropertyOrdering = append([]string(nil), r.Properties.keys...)
		for name, sub := range r.Properties.values {
			s.Properties[name] = sub.toGenai()
		}
	}

	// Infer a type when none was given, so Gemini always receives a valid type.
	if s.Type == "" {
		switch {
		case len(s.Properties) > 0:
			s.Type = genai.TypeObject
		case s.Items != nil:
			s.Type = genai.TypeArray
		default:
			s.Type = genai.TypeString
		}
	}
	return s
}

// normalizeType maps a JSON Schema "type" (a string, or an array such as
// ["string","null"]) to a genai.Type, reporting whether "null" was present.
func normalizeType(v any) (genai.Type, bool) {
	switch t := v.(type) {
	case string:
		return mapType(t), false
	case []any:
		var out genai.Type
		nullable := false
		for _, e := range t {
			s, ok := e.(string)
			if !ok {
				continue
			}
			if strings.EqualFold(s, "null") {
				nullable = true
				continue
			}
			if out == "" {
				out = mapType(s)
			}
		}
		return out, nullable
	default:
		return "", false
	}
}

func mapType(s string) genai.Type {
	switch strings.ToLower(s) {
	case "string":
		return genai.TypeString
	case "number":
		return genai.TypeNumber
	case "integer":
		return genai.TypeInteger
	case "boolean", "bool":
		return genai.TypeBoolean
	case "array":
		return genai.TypeArray
	case "object":
		return genai.TypeObject
	default:
		return ""
	}
}
