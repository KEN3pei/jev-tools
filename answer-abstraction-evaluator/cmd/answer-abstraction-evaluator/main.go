package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/KEN3pei/jev-tools/answer-abstraction-evaluator/evaluator"
)

func main() {
	inputPath := flag.String("input", "", "input JSON file; stdin is used when omitted")
	flag.Parse()

	var reader io.Reader = os.Stdin
	if *inputPath != "" {
		file, err := os.Open(*inputPath)
		if err != nil {
			fail(err)
		}
		defer file.Close()
		reader = file
	}

	var input evaluator.Input
	if err := json.NewDecoder(reader).Decode(&input); err != nil {
		fail(fmt.Errorf("read input: %w", err))
	}
	client := evaluator.Client{APIKey: os.Getenv("TYPESAFE_API_KEY")}
	result, err := client.Evaluate(context.Background(), input)
	if err != nil {
		fail(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
