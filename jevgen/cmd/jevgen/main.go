package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	differ "github.com/KEN3pei/jev-tools/jevgen/internal/diff"
	"github.com/KEN3pei/jev-tools/jevgen/internal/generate"
	"github.com/KEN3pei/jev-tools/jevgen/internal/schema"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "jevgen:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: jevgen <generate|check|diff>")
	}
	switch args[0] {
	case "generate":
		flags := flag.NewFlagSet("generate", flag.ContinueOnError)
		input := flags.String("input", "", "question set JSON")
		output := flags.String("output", "questions_gen.go", "generated Go file")
		pkg := flags.String("package", "jevschema", "generated Go package")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		set, err := schema.Load(*input)
		if err != nil {
			return err
		}
		code, err := generate.Go(set, *pkg)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(*output, code, 0o644); err != nil {
			return err
		}
		fmt.Println(*output)
		return nil
	case "check":
		flags := flag.NewFlagSet("check", flag.ContinueOnError)
		input := flags.String("input", "", "question set JSON")
		output := flags.String("output", "", "generated Go file to verify")
		pkg := flags.String("package", "jevschema", "generated Go package")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		set, err := schema.Load(*input)
		if err != nil {
			return err
		}
		if *output != "" {
			want, err := generate.Go(set, *pkg)
			if err != nil {
				return err
			}
			got, err := os.ReadFile(*output)
			if err != nil {
				return err
			}
			if string(got) != string(want) {
				return fmt.Errorf("generated file is stale: %s", *output)
			}
		}
		hash, err := set.Hash()
		if err != nil {
			return err
		}
		fmt.Printf("valid %s %s %s\n", set.Name, set.SchemaVersion, hash)
		return nil
	case "diff":
		flags := flag.NewFlagSet("diff", flag.ContinueOnError)
		oldPath := flags.String("old", "", "old question set JSON")
		newPath := flags.String("new", "", "new question set JSON")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		oldSet, err := schema.Load(*oldPath)
		if err != nil {
			return fmt.Errorf("old: %w", err)
		}
		newSet, err := schema.Load(*newPath)
		if err != nil {
			return fmt.Errorf("new: %w", err)
		}
		return json.NewEncoder(os.Stdout).Encode(differ.Compare(oldSet, newSet))
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
