package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func loadAPIKey() (string, error) {
	if value := strings.TrimSpace(os.Getenv("TYPESAFE_API_KEY")); value != "" { return value, nil }
	path, err := apiKeyFile()
	if err != nil { return "", err }
	file, err := os.Open(path)
	if os.IsNotExist(err) { return "", nil }
	if err != nil { return "", err }
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") { continue }
		key, value, found := strings.Cut(line, "=")
		if found && strings.TrimSpace(key) == "TYPESAFE_API_KEY" { return strings.TrimSpace(value), nil }
	}
	if err := scanner.Err(); err != nil { return "", fmt.Errorf("read %s: %w", path, err) }
	return "", nil
}

func apiKeyFile() (string, error) {
	if explicit := os.Getenv("GO_INTENT_ANALYZER_ENV_FILE"); explicit != "" { return explicit, nil }
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir(); if err != nil { return "", err }
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "go-intent-analyzer", "env"), nil
}
