package contract

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeAnswersRejectsMissingAnswer(t *testing.T) {
	_, err := DecodeAnswers([]byte(`{}`))
	if err == nil || !strings.Contains(err.Error(), "missing answer") {
		t.Fatalf("expected missing answer error, got %v", err)
	}
}

func TestDecodeAnswersRejectsUnknownChoice(t *testing.T) {
	answers := validAnswers()
	answers["requested_level"] = map[string]any{"choice": "new_unrecognized_level"}
	data, _ := json.Marshal(answers)
	_, err := DecodeAnswers(data)
	if err == nil || !strings.Contains(err.Error(), "invalid choice for requested_level") {
		t.Fatalf("expected invalid choice error, got %v", err)
	}
}

func TestDecodeAnswersAcceptsCompleteResponse(t *testing.T) {
	data, _ := json.Marshal(validAnswers())
	answers, err := DecodeAnswers(data)
	if err != nil {
		t.Fatal(err)
	}
	if answers.RequestedLevel.Choice != RequestedLevelChoiceConceptualOrientation {
		t.Fatalf("unexpected requested level: %s", answers.RequestedLevel.Choice)
	}
}

func validAnswers() map[string]any {
	return map[string]any{
		"evaluation_applicable":  map[string]any{"noul": 0.9},
		"requested_level":        map[string]any{"choice": "conceptual_orientation", "confidence": 0.8},
		"answer_entry_level":     map[string]any{"choice": "implementation_mechanics", "confidence": 0.7},
		"abstraction_mismatch":   map[string]any{"noul": 0.6},
		"premature_specificity":  map[string]any{"noul": 0.5},
		"progressive_disclosure": map[string]any{"noul": 0.4},
		"prerequisite_fit":       map[string]any{"noul": 0.3},
		"clarification_needed":   map[string]any{"noul": 0.2},
	}
}
