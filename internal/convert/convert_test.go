package convert

import (
	"strings"
	"testing"
)

const sample = `[
  {"question": "What is a PV?", "marks": 2, "tags": ["storage", "k8s"]},
  {"question": "Define a PVC", "marks": 3, "tags": ["storage"]}
]`

func TestConvertJSON(t *testing.T) {
	r, err := Convert([]byte(sample), "json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(r.Data), "\"question\": \"What is a PV?\"") {
		t.Fatalf("json not pretty/intact:\n%s", r.Data)
	}
}

func TestConvertYAMLPreservesOrderAndInts(t *testing.T) {
	r, err := Convert([]byte(sample), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	out := string(r.Data)
	// key order preserved (question before marks) and 2 stays an int, not "2".
	q := strings.Index(out, "question:")
	m := strings.Index(out, "marks:")
	if q < 0 || m < 0 || q > m {
		t.Fatalf("yaml order wrong:\n%s", out)
	}
	if strings.Contains(out, "marks: \"2\"") {
		t.Fatalf("yaml int rendered as string:\n%s", out)
	}
}

func TestConvertXML(t *testing.T) {
	r, err := Convert([]byte(sample), "xml")
	if err != nil {
		t.Fatal(err)
	}
	out := string(r.Data)
	if !strings.Contains(out, "<items>") || !strings.Contains(out, "<item>") {
		t.Fatalf("xml wrapping missing:\n%s", out)
	}
	if !strings.Contains(out, "<question>What is a PV?</question>") {
		t.Fatalf("xml content missing:\n%s", out)
	}
}

func TestConvertCSV(t *testing.T) {
	r, err := Convert([]byte(sample), "csv")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(r.Data)), "\n")
	if lines[0] != "question,marks,tags" {
		t.Fatalf("csv header/order wrong: %q", lines[0])
	}
	// nested array compact-JSON encoded in a cell.
	if !strings.Contains(string(r.Data), `"[""storage"",""k8s""]"`) {
		t.Fatalf("csv nested encoding wrong:\n%s", r.Data)
	}
}
