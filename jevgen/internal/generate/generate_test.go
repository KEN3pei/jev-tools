package generate

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/KEN3pei/jev-tools/jevgen/internal/schema"
)

func TestGoGeneratesTypedAnswers(t *testing.T) {
	set := schema.QuestionSet{Name: "test", SchemaVersion: "1", Questions: map[string]schema.Question{
		"requested_level": {Type: "choice", Instructions: "Choose", Criteria: json.RawMessage(`{"conceptual":"Conceptual","implementation":"Implementation"}`)},
		"quality":         {Type: "score", Instructions: "Score", Criteria: json.RawMessage(`["bad","good"]`)},
		"mismatch":        {Type: "noul", Instructions: "Mismatch"},
	}}
	code, err := Go(set, "contract")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "questions_gen.go", code, parser.AllErrors); err != nil {
		t.Fatal(err)
	}
	text := string(code)
	for _, want := range []string{
		"type RequestedLevelChoice string",
		"RequestedLevel ChoiceAnswer[RequestedLevelChoice]",
		"Quality",
		"ScoreAnswer",
		"Mismatch",
		"NoulAnswer",
		"DecodeAnswers",
		"missing answer",
		"const SchemaHash",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("generated code does not contain %q\n%s", want, text)
		}
	}
}
