package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAPIKeyPrefersEnvironment(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "from-env")
	t.Setenv("GO_INTENT_ANALYZER_ENV_FILE", filepath.Join(t.TempDir(), "missing"))
	value, err := loadAPIKey()
	if err != nil || value != "from-env" { t.Fatalf("value=%q err=%v", value, err) }
}

func TestLoadAPIKeyFromFile(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "")
	path := filepath.Join(t.TempDir(), "env")
	if err := os.WriteFile(path, []byte("# secret\nTYPESAFE_API_KEY=from-file\n"), 0o600); err != nil { t.Fatal(err) }
	t.Setenv("GO_INTENT_ANALYZER_ENV_FILE", path)
	value, err := loadAPIKey()
	if err != nil || value != "from-file" { t.Fatalf("value=%q err=%v", value, err) }
}
