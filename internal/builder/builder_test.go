package builder

import (
	"strings"
	"testing"

	"google.golang.org/genai"
	"lucy/internal/schema"
)

func TestBuildSchemaOrderAndRequired(t *testing.T) {
	got, err := BuildSchema([]Field{
		{Name: "question", Type: "string", Required: true},
		{Name: "marks", Type: "integer"},
		{Name: "tags", Type: "array", ItemType: "string"},
		{Name: "", Type: "string"}, // skipped
	})
	if err != nil {
		t.Fatal(err)
	}
	// Feed straight into schema.Build to confirm it parses and wraps cleanly.
	s, err := schema.Build(got, 4)
	if err != nil {
		t.Fatalf("schema.Build on builder output: %v\n%s", err, got)
	}
	if s.Type != genai.TypeArray || s.Items.Type != genai.TypeObject {
		t.Fatalf("expected array of objects, got %+v", s)
	}
	wantOrder := []string{"question", "marks", "tags"}
	for i, k := range wantOrder {
		if s.Items.PropertyOrdering[i] != k {
			t.Fatalf("order mismatch: %v", s.Items.PropertyOrdering)
		}
	}
	if s.Items.Properties["tags"].Type != genai.TypeArray || s.Items.Properties["tags"].Items.Type != genai.TypeString {
		t.Fatalf("array item type lost: %+v", s.Items.Properties["tags"])
	}
	if len(s.Items.Required) != 1 || s.Items.Required[0] != "question" {
		t.Fatalf("required wrong: %v", s.Items.Required)
	}
}

func TestBuildSchemaNoFields(t *testing.T) {
	if _, err := BuildSchema([]Field{{Name: "  "}}); err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("expected no-fields error, got %v", err)
	}
}
