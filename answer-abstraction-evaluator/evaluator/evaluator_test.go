package evaluator

import "testing"

func TestBuildState(t *testing.T) {
	state, err := BuildState(Input{UserRequest: "question", CandidateAnswer: "answer"})
	if err != nil {
		t.Fatal(err)
	}
	if state["user_request"] != "question" {
		t.Fatalf("unexpected state: %#v", state)
	}
}

func TestDecide(t *testing.T) {
	tests := []struct {
		name   string
		scores Scores
		want   string
	}{
		{"clarify", Scores{ClarificationNeeded: .8}, "ask_clarifying_question"},
		{"restructure", Scores{AbstractionMismatch: .8, PrerequisiteFit: 1, ProgressiveDisclosure: 1}, "restructure"},
		{"revise", Scores{PrematureSpecificity: .8, PrerequisiteFit: 1, ProgressiveDisclosure: 1}, "revise_entry"},
		{"pass", Scores{PrerequisiteFit: 1, ProgressiveDisclosure: 1}, "pass"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Decide(tt.scores, DefaultThresholds); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}
