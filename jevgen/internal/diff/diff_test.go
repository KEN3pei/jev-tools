package diff

import (
	"encoding/json"
	"testing"

	"github.com/KEN3pei/jev-tools/jevgen/internal/schema"
)

func TestCompareClassifiesContractChanges(t *testing.T) {
	oldSet := schema.QuestionSet{Questions: map[string]schema.Question{
		"choice": {Type: "choice", Instructions: "Old", Criteria: json.RawMessage(`{"a":"A","b":"B"}`)},
		"gone":   {Type: "noul", Instructions: "Gone"},
	}}
	newSet := schema.QuestionSet{Questions: map[string]schema.Question{
		"choice": {Type: "choice", Instructions: "New", Criteria: json.RawMessage(`{"a":"A","c":"C"}`)},
		"added":  {Type: "score", Instructions: "Added", Criteria: json.RawMessage(`["bad","good"]`)},
	}}
	changes := Compare(oldSet, newSet)
	want := map[string]string{"added": "additive", "choice": "breaking", "gone": "breaking"}
	for _, change := range changes {
		if severity, ok := want[change.Question]; ok && change.Severity != severity && change.Message != "instructions changed" {
			t.Errorf("%s: got %s, want %s", change.Question, change.Severity, severity)
		}
	}
}
