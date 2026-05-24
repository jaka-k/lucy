// Package builder turns the visual field builder's input into a JSON Schema
// string, which the schema package then parses and wraps into a list.
package builder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Types are the field types offered by the visual builder (also used to
// validate input and to populate the type dropdown).
var Types = []string{"string", "integer", "number", "boolean", "array", "object"}

// Field is one row of the visual builder.
type Field struct {
	Name        string
	Type        string
	ItemType    string // element type when Type == "array"
	Description string
	Required    bool
}

// BuildSchema assembles an object JSON Schema from the builder fields. Property
// declaration order is preserved so downstream output keeps the same order.
func BuildSchema(fields []Field) (string, error) {
	var (
		buf      bytes.Buffer
		required []string
		written  int
	)
	buf.WriteString(`{"type":"object","properties":{`)
	for _, f := range fields {
		name := strings.TrimSpace(f.Name)
		if name == "" {
			continue
		}
		if written > 0 {
			buf.WriteByte(',')
		}
		written++

		nameJSON, err := json.Marshal(name)
		if err != nil {
			return "", err
		}
		propJSON, err := json.Marshal(propSchema(f))
		if err != nil {
			return "", err
		}
		buf.Write(nameJSON)
		buf.WriteByte(':')
		buf.Write(propJSON)

		if f.Required {
			required = append(required, name)
		}
	}
	buf.WriteByte('}')

	if written == 0 {
		return "", fmt.Errorf("add at least one field with a name")
	}
	if len(required) > 0 {
		reqJSON, err := json.Marshal(required)
		if err != nil {
			return "", err
		}
		buf.WriteString(`,"required":`)
		buf.Write(reqJSON)
	}
	buf.WriteByte('}')
	return buf.String(), nil
}

func propSchema(f Field) map[string]any {
	p := map[string]any{"type": normalizeType(f.Type)}
	if d := strings.TrimSpace(f.Description); d != "" {
		p["description"] = d
	}
	if p["type"] == "array" {
		p["items"] = map[string]any{"type": normalizeType(f.ItemType)}
	}
	return p
}

func normalizeType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	for _, valid := range Types {
		if t == valid {
			return t
		}
	}
	return "string"
}
