package analyzer

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"

	golangquestions "github.com/KEN3pei/jev-tools/go-intent-analyzer/analyzer/languages/golang"
)

func TestExtractUnitsRecursivelyAndTracksContext(t *testing.T) {
	root := t.TempDir()
	packageDir := filepath.Join(root, "service")
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := `// Package service demonstrates an extensible boundary.
package service

type Runner interface { Run() error }

type Worker struct{}

// Run validates input so callers can rely on a stable boundary.
func (Worker) Run() error { return nil }

func Execute(r Runner) error { return r.Run() }
`
	testSource := `package service
import "testing"
func TestWorkerRun(t *testing.T) { if err := (Worker{}).Run(); err != nil { t.Fatal(err) } }
`
	if err := os.WriteFile(filepath.Join(packageDir, "service.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageDir, "service_test.go"), []byte(testSource), 0o644); err != nil {
		t.Fatal(err)
	}

	units, err := ExtractUnits(root, golangquestions.Definitions)
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 5 {
		t.Fatalf("got %d units, want 5", len(units))
	}
	var run *UnitContext
	for _, unit := range units {
		if unit.Name() == "Worker.Run" {
			context := unit.Ctx()
			run = &context
			break
		}
	}
	if run == nil {
		t.Fatal("Worker.Run not extracted")
	}
	if len(run.Interfaces) != 1 || run.Interfaces[0].Name != "Runner" {
		t.Fatalf("interfaces = %#v", run.Interfaces)
	}
	if len(run.RelatedTests) != 1 || run.RelatedTests[0].Snippet == "" {
		t.Fatalf("tests = %#v", run.RelatedTests)
	}
	if len(run.EvidenceByIntent["extensibility"]) < 2 {
		t.Fatalf("extensibility evidence = %#v", run.EvidenceByIntent["extensibility"])
	}
}

func TestLSPPositionUsesUTF16CodeUnits(t *testing.T) {
	position := lspPositionForTest("a😀b", len("a😀"))
	if position != 3 {
		t.Fatalf("character = %d, want 3", position)
	}
}

func lspPositionForTest(line string, byteOffset int) int {
	position := lspPosition(tokenPositionForTest(byteOffset), line)
	return position.Character
}

func tokenPositionForTest(offset int) token.Position {
	return token.Position{Line: 1, Column: offset + 1, Offset: offset}
}
