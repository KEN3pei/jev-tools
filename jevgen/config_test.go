package jevgen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config.InputPath() != "questions.json" || config.OutputPath() != "questions_gen.go" || config.Package != "jevschema" {
		t.Fatalf("unexpected defaults: %#v", config)
	}
}

func TestLoadConfigResolvesPathsFromConfigDirectory(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "jevgen.json")
	data := []byte(`{"version":"1","input":"contract/questions.json","output":"generated/questions_gen.go","package":"contract"}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(directory, "contract/questions.json"); config.InputPath() != want {
		t.Fatalf("InputPath() = %q, want %q", config.InputPath(), want)
	}
	if want := filepath.Join(directory, "generated/questions_gen.go"); config.OutputPath() != want {
		t.Fatalf("OutputPath() = %q, want %q", config.OutputPath(), want)
	}
}

func TestLoadConfigOrDefaultOnlyIgnoresConventionalMissingFile(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	temporaryDirectory := t.TempDir()
	if err := os.Chdir(temporaryDirectory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDirectory) })

	if _, err := LoadConfigOrDefault(DefaultConfigPath); err != nil {
		t.Fatalf("default config should be optional: %v", err)
	}
	if _, err := LoadConfigOrDefault("custom.json"); err == nil {
		t.Fatal("explicit missing config should fail")
	}
}
