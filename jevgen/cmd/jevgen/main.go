package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/KEN3pei/jev-tools/jevgen"
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
		configPath := flags.String("config", jevgen.DefaultConfigPath, "configuration file")
		input := flags.String("input", "", "override question set path")
		output := flags.String("output", "", "override generated Go path")
		pkg := flags.String("package", "", "override generated Go package")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		config, err := commandConfig(*configPath, *input, *output, *pkg)
		if err != nil {
			return err
		}
		set, err := jevgen.LoadQuestionSet(config.InputPath())
		if err != nil {
			return err
		}
		code, err := jevgen.GenerateGo(set, config.Package)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(config.OutputPath()), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(config.OutputPath(), code, 0o644); err != nil {
			return err
		}
		fmt.Println(config.OutputPath())
		return nil
	case "check":
		flags := flag.NewFlagSet("check", flag.ContinueOnError)
		configPath := flags.String("config", jevgen.DefaultConfigPath, "configuration file")
		input := flags.String("input", "", "override question set path")
		output := flags.String("output", "", "override generated Go path")
		pkg := flags.String("package", "", "override generated Go package")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		config, err := commandConfig(*configPath, *input, *output, *pkg)
		if err != nil {
			return err
		}
		set, err := jevgen.LoadQuestionSet(config.InputPath())
		if err != nil {
			return err
		}
		want, err := jevgen.GenerateGo(set, config.Package)
		if err != nil {
			return err
		}
		got, err := os.ReadFile(config.OutputPath())
		if err != nil {
			return err
		}
		if string(got) != string(want) {
			return fmt.Errorf("generated file is stale: %s", config.OutputPath())
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
		oldSet, err := jevgen.LoadQuestionSet(*oldPath)
		if err != nil {
			return fmt.Errorf("old: %w", err)
		}
		newSet, err := jevgen.LoadQuestionSet(*newPath)
		if err != nil {
			return fmt.Errorf("new: %w", err)
		}
		return json.NewEncoder(os.Stdout).Encode(jevgen.Compare(oldSet, newSet))
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func commandConfig(configPath, input, output, packageName string) (jevgen.Config, error) {
	config, err := jevgen.LoadConfigOrDefault(configPath)
	if err != nil {
		return jevgen.Config{}, err
	}
	if input != "" {
		absolute, err := filepath.Abs(input)
		if err != nil {
			return jevgen.Config{}, err
		}
		config.Input = absolute
	}
	if output != "" {
		absolute, err := filepath.Abs(output)
		if err != nil {
			return jevgen.Config{}, err
		}
		config.Output = absolute
	}
	if packageName != "" {
		config.Package = packageName
	}
	if err := config.Validate(); err != nil {
		return jevgen.Config{}, err
	}
	return config, nil
}
