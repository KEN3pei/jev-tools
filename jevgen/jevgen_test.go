package jevgen_test

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"testing"

	"github.com/KEN3pei/jev-tools/jevgen"
)

func TestPublicPackageGeneratesContractWithoutDataFiles(t *testing.T) {
	set := jevgen.QuestionSet{
		Name:          "example",
		SchemaVersion: "1",
		Questions: map[string]jevgen.Question{
			"decision": {
				Type:         "choice",
				Instructions: "Choose a decision.",
				Criteria:     json.RawMessage(`{"accept":"Accept","reject":"Reject"}`),
			},
		},
	}
	if err := set.Validate(); err != nil {
		t.Fatal(err)
	}
	code, err := jevgen.GenerateGo(set, "contract")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "questions_gen.go", code, parser.AllErrors); err != nil {
		t.Fatal(err)
	}
}
