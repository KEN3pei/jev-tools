package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/KEN3pei/jev-tools/answer-abstraction-evaluator/evaluator"
	"github.com/KEN3pei/jev-tools/answer-abstraction-evaluator/hook"
)

func main() {
	eventName := flag.String("event", "", "hook event: user-prompt-submit or stop")
	dataDir := flag.String("data-dir", defaultDataDir(), "state and log directory")
	flag.Parse()
	var event hook.Event
	if err := json.NewDecoder(os.Stdin).Decode(&event); err != nil {
		fail(err)
	}
	handler := hook.Handler{DataDir: *dataDir, Client: evaluator.Client{APIKey: os.Getenv("TYPESAFE_API_KEY")}}
	var output hook.Output
	switch *eventName {
	case "user-prompt-submit":
		if err := handler.CapturePrompt(event); err != nil {
			fail(err)
		}
		output = hook.Output{Continue: true}
	case "stop":
		output = handler.EvaluateStop(context.Background(), event)
	default:
		fail(fmt.Errorf("unsupported --event %q", *eventName))
	}
	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		fail(err)
	}
}

func defaultDataDir() string {
	if value := os.Getenv("JEV_HOOK_DATA_DIR"); value != "" {
		return value
	}
	if value := os.Getenv("PLUGIN_DATA"); value != "" {
		return value
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return ".jev-hook-data"
	}
	return filepath.Join(dir, "jev-tools", "answer-abstraction-evaluator")
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
