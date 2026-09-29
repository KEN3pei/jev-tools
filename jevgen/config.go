package jevgen

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultConfigPath = "jevgen.json"

type Config struct {
	Version  string `json:"version,omitempty"`
	Input    string `json:"input"`
	Output   string `json:"output"`
	Package  string `json:"package"`
	basePath string
}

func DefaultConfig() Config {
	return Config{
		Version: "1",
		Input:   "questions.json",
		Output:  "questions_gen.go",
		Package: "jevschema",
	}
}

// LoadConfig reads a config file. Relative input and output paths are resolved
// from the directory containing the config file.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	config := DefaultConfig()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	config.basePath = filepath.Dir(path)
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

// LoadConfigOrDefault uses the built-in defaults when the conventional
// jevgen.json path does not exist. A missing explicitly named path is an error.
func LoadConfigOrDefault(path string) (Config, error) {
	config, err := LoadConfig(path)
	if err == nil {
		return config, nil
	}
	if errors.Is(err, os.ErrNotExist) && path == DefaultConfigPath {
		return DefaultConfig(), nil
	}
	return Config{}, err
}

func (c Config) Validate() error {
	if c.Version != "" && c.Version != "1" {
		return fmt.Errorf("unsupported config version %q", c.Version)
	}
	if c.Input == "" {
		return errors.New("config input is required")
	}
	if c.Output == "" {
		return errors.New("config output is required")
	}
	if c.Package == "" {
		return errors.New("config package is required")
	}
	return nil
}

func (c Config) InputPath() string {
	return resolvePath(c.basePath, c.Input)
}

func (c Config) OutputPath() string {
	return resolvePath(c.basePath, c.Output)
}

func resolvePath(basePath, path string) string {
	if basePath == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(basePath, path)
}
