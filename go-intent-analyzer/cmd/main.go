package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/KEN3pei/jev-tools/go-intent-analyzer/analyzer"
	golangquestions "github.com/KEN3pei/jev-tools/go-intent-analyzer/analyzer/languages/golang"
)

const analyzerVersion = "0.1.0"

func main() {
	var (
		dir        = flag.String("dir", ".", "解析するGoパッケージのディレクトリパス")
		model      = flag.String("model", "", "使用するJevモデル（デフォルト: jev-latest）")
		minEvidFit = flag.Float64("min-evidencefit", 0.0, "この値未満のevidenceFitを持つ意図を出力から除外（0.0=除外なし）")
	)
	flag.Parse()

	apiKey, keyErr := loadAPIKey()
	if keyErr != nil {
		fmt.Fprintf(os.Stderr, "error loading TypeSafe API key: %v\n", keyErr)
		os.Exit(1)
	}
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "error: TypeSafe API key is not configured; set TYPESAFE_API_KEY or run install_setup.sh")
		os.Exit(1)
	}

	absDir, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	units, err := analyzer.ExtractUnits(absDir, golangquestions.Definitions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error extracting units: %v\n", err)
		os.Exit(1)
	}

	intentAnalyzer := analyzer.Analyzer{
		Client:      analyzer.Client{APIKey: apiKey, Model: *model},
		Language:    golangquestions.Language,
		Task:        golangquestions.Task,
		Definitions: golangquestions.Definitions,
		Questions:   golangquestions.Questions,
	}

	report := analyzer.Report{
		SchemaVersion:   analyzer.SchemaVersion,
		AnalyzerVersion: analyzerVersion,
		GeneratedAt:     time.Now().UTC(),
		Repository:      analyzer.Repository{Root: absDir, Revision: analyzer.RepositoryRevision(absDir)},
		Units:           make([]analyzer.UnitReport, 0, len(units)),
	}
	ctx := context.Background()

	for _, u := range units {
		intents, err := intentAnalyzer.AnalyzeUnit(ctx, u.Ctx())
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: analyze %s %s: %v\n", u.Kind(), u.Name(), err)
			report.Errors = append(report.Errors, analyzer.AnalysisError{UnitID: u.ID(), Message: err.Error()})
			continue
		}

		filtered := filterIntents(intents, *minEvidFit)

		report.Units = append(report.Units, analyzer.UnitReport{
			ID:      u.ID(),
			URI:     u.URI(),
			Range:   u.SrcRange(),
			Kind:    u.Kind(),
			Name:    u.Name(),
			Intents: filtered,
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding output: %v\n", err)
		os.Exit(1)
	}
	if len(report.Errors) > 0 {
		os.Exit(1)
	}
}

func filterIntents(intents []analyzer.IntentResult, minFit float64) []analyzer.IntentResult {
	if minFit <= 0 {
		return intents
	}
	out := intents[:0]
	for _, i := range intents {
		if i.EvidenceFit == nil {
			continue
		}
		if *i.EvidenceFit >= minFit {
			out = append(out, i)
		}
	}
	return out
}
