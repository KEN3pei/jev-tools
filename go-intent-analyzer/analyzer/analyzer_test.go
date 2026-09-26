package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	golangquestions "github.com/KEN3pei/jev-tools/go-intent-analyzer/analyzer/languages/golang"
	"github.com/KEN3pei/jev-tools/go-intent-analyzer/analyzer/questions"
)

func TestBuildGoIntentQuestionsUsesIndependentChoices(t *testing.T) {
	questionSet := golangquestions.Questions
	if len(questionSet) != 11 {
		t.Fatalf("got %d questions, want 11", len(questionSet))
	}
	for id, question := range questionSet {
		if question.Type != "choice" {
			t.Errorf("%s type = %q, want choice", id, question.Type)
		}
		if len(question.Criteria) != 4 {
			t.Errorf("%s has %d choices, want 4", id, len(question.Criteria))
		}
	}
}

func TestAnalyzeUnitMapsChoiceResponse(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var request struct {
			Questions map[string]questions.Question `json:"questions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return nil, err
		}
		if len(request.Questions) != 11 {
			return nil, &testError{"unexpected question count"}
		}
		answers := map[string]any{}
		for _, definition := range golangquestions.Definitions {
			choice := "supported"
			if definition.ID == "performance" {
				choice = "insufficient"
			}
			answers[definition.ID] = map[string]any{"type": "choice", "choice": choice, "confidence": 0.8, "probabilities": map[string]float64{"supported": 0.7, "weakly_supported": 0.1, "contradicted": 0.05, "insufficient": 0.15}}
		}
		var body bytes.Buffer
		_ = json.NewEncoder(&body).Encode(map[string]any{"model": "jev-test", "answers": answers, "usage": map[string]int{"input_tokens": 1, "output_tokens": 1}})
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(&body), Header: make(http.Header)}, nil
	})

	unit := UnitContext{ID: "sample", EvidenceByIntent: map[string][]Evidence{"simplicity": {{ID: "e-1", Kind: "source", Signal: "sample"}}}}
	goAnalyzer := Analyzer{Client: Client{APIKey: "test", BaseURL: "https://example.invalid", HTTPClient: &http.Client{Transport: transport}}, Language: golangquestions.Language, Task: golangquestions.Task, Definitions: golangquestions.Definitions, Questions: golangquestions.Questions}
	results, err := goAnalyzer.AnalyzeUnit(context.Background(), unit)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 11 {
		t.Fatalf("got %d results", len(results))
	}
	if results[0].EvidenceFit == nil || *results[0].EvidenceFit != 0.75 {
		t.Fatalf("unexpected evidence fit: %v", results[0].EvidenceFit)
	}
	for _, result := range results {
		if result.Intent == "performance" {
			if result.EvidenceFit != nil {
				t.Errorf("insufficient evidenceFit = %v, want nil", *result.EvidenceFit)
			}
			if len(result.Evidence) != 0 {
				t.Errorf("insufficient result has evidence")
			}
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

type testError struct{ message string }

func (e *testError) Error() string { return e.message }

func TestBuildIntentResultsRejectsMissingAnswers(t *testing.T) {
	goAnalyzer := Analyzer{Definitions: golangquestions.Definitions}
	if _, err := goAnalyzer.buildIntentResults(map[string]Answer{}, UnitContext{}); err == nil {
		t.Fatal("expected error")
	}
}
