package schema

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateAcceptsSupportedQuestions(t *testing.T) {
	set := QuestionSet{Name: "test", SchemaVersion: "1", Questions: map[string]Question{
		"choice_question": {Type: "choice", Instructions: "Choose", Criteria: json.RawMessage(`{"a":"A"}`)},
		"score_question":  {Type: "score", Instructions: "Score", Criteria: json.RawMessage(`["bad","good"]`)},
		"noul_question":   {Type: "noul", Instructions: "Probability"},
	}}
	if err := set.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReportsAllUsefulProblems(t *testing.T) {
	set := QuestionSet{Questions: map[string]Question{
		"Bad-ID": {Type: "unknown"},
	}}
	err := set.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	for _, want := range []string{"name is required", "schemaVersion is required", "ID must match", "instructions are required", "unsupported type"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
}

func TestHashIsIndependentOfMapInsertionOrder(t *testing.T) {
	question := Question{Type: "noul", Instructions: "Test"}
	left := QuestionSet{Name: "test", SchemaVersion: "1", Questions: map[string]Question{"a": question, "b": question}}
	right := QuestionSet{Name: "test", SchemaVersion: "1", Questions: map[string]Question{"b": question, "a": question}}
	leftHash, _ := left.Hash()
	rightHash, _ := right.Hash()
	if leftHash != rightHash {
		t.Fatalf("hashes differ: %s != %s", leftHash, rightHash)
	}
}
