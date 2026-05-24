package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeEnv(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadDotEnvParsesAndKeepsRealEnv(t *testing.T) {
	t.Setenv("LUCY_REAL", "from-real-env")

	path := writeEnv(t, `
# a comment
export LUCY_PLAIN=hello
LUCY_QUOTED="quoted value"
LUCY_SINGLE='single value'
LUCY_REAL=should-be-ignored
`)

	if err := loadDotEnv(path); err != nil {
		t.Fatalf("loadDotEnv: %v", err)
	}

	cases := map[string]string{
		"LUCY_PLAIN":  "hello",
		"LUCY_QUOTED": "quoted value",
		"LUCY_SINGLE": "single value",
		"LUCY_REAL":   "from-real-env", // real env wins over .env
	}
	for k, want := range cases {
		t.Cleanup(func() { os.Unsetenv(k) })
		if got := os.Getenv(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

func TestLoadDotEnvMissingFileOK(t *testing.T) {
	if err := loadDotEnv(filepath.Join(t.TempDir(), "nope.env")); err != nil {
		t.Errorf("missing file should not error, got %v", err)
	}
}

func TestLoadDotEnvMalformed(t *testing.T) {
	path := writeEnv(t, "NOEQUALS\n")
	if err := loadDotEnv(path); err == nil {
		t.Error("expected error for line without '='")
	}
}
