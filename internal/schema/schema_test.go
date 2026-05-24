package schema

import (
	"testing"

	"google.golang.org/genai"
)

func TestBuildWrapsItemSchema(t *testing.T) {
	const item = `{
		"type": "object",
		"properties": {
			"question": {"type": "string"},
			"answer": {"type": "string"},
			"difficulty": {"type": "string", "enum": ["easy", "hard"]}
		},
		"required": ["question", "answer"]
	}`

	got, err := Build(item, 5)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got.Type != genai.TypeArray {
		t.Fatalf("want ARRAY root, got %q", got.Type)
	}
	if got.MinItems == nil || *got.MinItems != 5 || got.MaxItems == nil || *got.MaxItems != 5 {
		t.Fatalf("count not applied: min=%v max=%v", got.MinItems, got.MaxItems)
	}
	if got.Items == nil || got.Items.Type != genai.TypeObject {
		t.Fatalf("item should be object, got %+v", got.Items)
	}
	wantOrder := []string{"question", "answer", "difficulty"}
	if len(got.Items.PropertyOrdering) != len(wantOrder) {
		t.Fatalf("ordering len: %v", got.Items.PropertyOrdering)
	}
	for i, k := range wantOrder {
		if got.Items.PropertyOrdering[i] != k {
			t.Fatalf("property order mismatch at %d: got %v want %v", i, got.Items.PropertyOrdering, wantOrder)
		}
	}
	if got.Items.Properties["difficulty"].Format != "enum" {
		t.Errorf("enum string should set format=enum")
	}
}

func TestBuildNestedObjectAndArrayOfObject(t *testing.T) {
	// Mirrors what the visual builder serializes for nested fields.
	const item = `{
		"type": "object",
		"properties": {
			"title": {"type": "string"},
			"meta": {
				"type": "object",
				"properties": {
					"author": {"type": "string"},
					"year": {"type": "integer"}
				},
				"required": ["author"]
			},
			"options": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"text": {"type": "string"},
						"correct": {"type": "boolean"}
					}
				}
			}
		}
	}`

	got, err := Build(item, 0)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got.Type != genai.TypeArray {
		t.Fatalf("want ARRAY root, got %q", got.Type)
	}
	props := got.Items.Properties

	meta := props["meta"]
	if meta == nil || meta.Type != genai.TypeObject {
		t.Fatalf("meta should be a nested object: %+v", meta)
	}
	if meta.Properties["year"].Type != genai.TypeInteger {
		t.Errorf("meta.year type lost: %+v", meta.Properties["year"])
	}
	if len(meta.Required) != 1 || meta.Required[0] != "author" {
		t.Errorf("meta.required wrong: %v", meta.Required)
	}

	opts := props["options"]
	if opts == nil || opts.Type != genai.TypeArray || opts.Items.Type != genai.TypeObject {
		t.Fatalf("options should be an array of objects: %+v", opts)
	}
	if opts.Items.Properties["correct"].Type != genai.TypeBoolean {
		t.Errorf("array-item object property lost: %+v", opts.Items.Properties["correct"])
	}
}

func TestBuildArrayRootUsedAsIs(t *testing.T) {
	const arr = `{"type": "array", "items": {"type": "string"}}`
	got, err := Build(arr, 0)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got.Type != genai.TypeArray || got.Items.Type != genai.TypeString {
		t.Fatalf("array root not preserved: %+v", got)
	}
	if got.MinItems != nil {
		t.Errorf("count 0 should not set MinItems")
	}
}

func TestBuildRejectsEmpty(t *testing.T) {
	if _, err := Build("   ", 0); err == nil {
		t.Fatal("expected error for empty schema")
	}
}

func TestNullableUnionType(t *testing.T) {
	got, err := Build(`{"type": ["string", "null"]}`, 0)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	// Wrapped into an array of nullable strings.
	if got.Items.Type != genai.TypeString || got.Items.Nullable == nil || !*got.Items.Nullable {
		t.Fatalf("nullable union not handled: %+v", got.Items)
	}
}
