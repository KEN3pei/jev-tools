package evaluator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	DefaultModel   = "jev-latest"
	DefaultBaseURL = "https://api.typesafe.ai/v1/systemone"
)

type Input struct {
	PrecedingContext []string `json:"precedingContext,omitempty"`
	UserRequest      string   `json:"userRequest"`
	CandidateAnswer  string   `json:"candidateAnswer"`
}

type Question struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}

var Questions = map[string]Question{
	"requested_level": {
		Type: "choice", Instructions: "What is the primary abstraction level requested by `user_request`, interpreted with `preceding_context`?",
		Criteria: levelCriteria(),
	},
	"answer_entry_level": {
		Type: "choice", Instructions: "At what abstraction level does `candidate_answer` begin its substantive explanation?",
		Criteria: levelCriteria(),
	},
	"abstraction_mismatch": {
		Type: "noul", Instructions: "Does `candidate_answer` begin at a materially different abstraction level from the one primarily requested, making orientation harder even if later content is relevant?",
		Criteria: map[string]string{"true": "The answer starts too concretely or too abstractly relative to the request.", "false": "The answer starts at an appropriate level and then moves through detail coherently."},
	},
	"premature_specificity": {
		Type: "noul", Instructions: "Does `candidate_answer` introduce code, products, APIs, configuration, or implementation mechanisms before establishing the mental model requested by the user?",
		Criteria: map[string]string{"true": "Specific detail arrives before the reader has an adequate conceptual map.", "false": "The conceptual map is established first, or implementation detail is clearly the primary request."},
	},
	"progressive_disclosure": {
		Type: "noul", Instructions: "Does `candidate_answer` progress in an appropriate order from the requested level toward patterns, tradeoffs, examples, and implementation details?",
		Criteria: map[string]string{"true": "The answer introduces detail in an order that supports understanding.", "false": "The answer jumps between levels or introduces lower-level detail before the necessary foundation."},
	},
	"prerequisite_fit": {
		Type: "noul", Instructions: "Does `candidate_answer` avoid assuming that the user already understands concepts they are currently asking to understand?",
		Criteria: map[string]string{"true": "The answer supplies the necessary conceptual prerequisites before relying on them.", "false": "The answer relies on unexplained concepts that are part of the user's current learning goal."},
	},
	"clarification_needed": {
		Type: "noul", Instructions: "Was the intended abstraction level too ambiguous to choose a reasonable answer sequence without asking a clarifying question?",
		Criteria: map[string]string{"true": "Multiple materially different levels were equally plausible from the available context.", "false": "The request and preceding context provided enough evidence to choose a reasonable level."},
	},
}

func levelCriteria() map[string]string {
	return map[string]string{
		"conceptual_orientation":        "A basic definition, orientation, or mental model is primary.",
		"architecture_and_design_space": "System patterns, components, boundaries, alternatives, and tradeoffs are primary.",
		"implementation_mechanics":      "Concrete code, APIs, configuration, commands, or integration steps are primary.",
		"operations_and_governance":     "Deployment, monitoring, reliability, security, governance, or organizational operation is primary.",
		"unclear":                       "The intended abstraction level cannot be inferred reliably.",
	}
}

type Thresholds struct {
	ClarificationNeeded          float64
	AbstractionMismatch          float64
	PrematureSpecificity         float64
	ProgressiveDisclosureMinimum float64
	PrerequisiteFitMinimum       float64
}

var DefaultThresholds = Thresholds{0.7, 0.7, 0.65, 0.5, 0.3}

type Scores struct {
	AbstractionMismatch   float64 `json:"abstractionMismatch"`
	PrematureSpecificity  float64 `json:"prematureSpecificity"`
	ProgressiveDisclosure float64 `json:"progressiveDisclosure"`
	PrerequisiteFit       float64 `json:"prerequisiteFit"`
	ClarificationNeeded   float64 `json:"clarificationNeeded"`
}

type Result struct {
	RequestedLevel             string            `json:"requestedLevel"`
	RequestedLevelConfidence   float64           `json:"requestedLevelConfidence"`
	AnswerEntryLevel           string            `json:"answerEntryLevel"`
	AnswerEntryLevelConfidence float64           `json:"answerEntryLevelConfidence"`
	Scores                     Scores            `json:"scores"`
	Decision                   string            `json:"decision"`
	Model                      string            `json:"model,omitempty"`
	Usage                      map[string]any    `json:"usage,omitempty"`
	RawAnswers                 map[string]Answer `json:"rawAnswers"`
}

type Answer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice,omitempty"`
	Confidence    float64            `json:"confidence,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Noul          *float64           `json:"noul,omitempty"`
}

type Client struct {
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
	Thresholds Thresholds
}

func BuildState(input Input) (map[string]any, error) {
	if strings.TrimSpace(input.UserRequest) == "" {
		return nil, fmt.Errorf("userRequest must be non-empty")
	}
	if strings.TrimSpace(input.CandidateAnswer) == "" {
		return nil, fmt.Errorf("candidateAnswer must be non-empty")
	}
	if input.PrecedingContext == nil {
		input.PrecedingContext = []string{}
	}
	return map[string]any{
		"evaluation_task":   "Infer the requested abstraction level and evaluate whether the answer begins and progresses at an appropriate level. Do not answer the user request.",
		"preceding_context": input.PrecedingContext,
		"user_request":      input.UserRequest,
		"candidate_answer":  input.CandidateAnswer,
	}, nil
}

func Decide(s Scores, t Thresholds) string {
	if s.ClarificationNeeded >= t.ClarificationNeeded {
		return "ask_clarifying_question"
	}
	if s.AbstractionMismatch >= t.AbstractionMismatch || s.PrerequisiteFit < t.PrerequisiteFitMinimum {
		return "restructure"
	}
	if s.PrematureSpecificity >= t.PrematureSpecificity || s.ProgressiveDisclosure < t.ProgressiveDisclosureMinimum {
		return "revise_entry"
	}
	return "pass"
}

func (c Client) Evaluate(ctx context.Context, input Input) (Result, error) {
	if c.APIKey == "" {
		return Result{}, fmt.Errorf("TYPESAFE_API_KEY is not configured")
	}
	state, err := BuildState(input)
	if err != nil {
		return Result{}, err
	}
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	model := c.Model
	if model == "" {
		model = DefaultModel
	}
	thresholds := c.Thresholds
	if thresholds == (Thresholds{}) {
		thresholds = DefaultThresholds
	}
	payload, err := json.Marshal(map[string]any{"model": model, "state": state, "questions": Questions})
	if err != nil {
		return Result{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewReader(payload))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("Jev request failed (%d): %.300s", resp.StatusCode, body)
	}
	var apiResult struct {
		Answers map[string]Answer `json:"answers"`
		Model   string            `json:"model"`
		Usage   map[string]any    `json:"usage"`
	}
	if err := json.Unmarshal(body, &apiResult); err != nil {
		return Result{}, fmt.Errorf("decode Jev response: %w", err)
	}
	getNoul := func(name string) (float64, error) {
		a, ok := apiResult.Answers[name]
		if !ok || a.Noul == nil {
			return 0, fmt.Errorf("Jev response is missing a valid Noul answer: %s", name)
		}
		return *a.Noul, nil
	}
	requested, ok := apiResult.Answers["requested_level"]
	if !ok || requested.Choice == "" {
		return Result{}, fmt.Errorf("Jev response is missing requested_level")
	}
	entry, ok := apiResult.Answers["answer_entry_level"]
	if !ok || entry.Choice == "" {
		return Result{}, fmt.Errorf("Jev response is missing answer_entry_level")
	}
	names := []string{"abstraction_mismatch", "premature_specificity", "progressive_disclosure", "prerequisite_fit", "clarification_needed"}
	values := make([]float64, len(names))
	for i, name := range names {
		values[i], err = getNoul(name)
		if err != nil {
			return Result{}, err
		}
	}
	scores := Scores{values[0], values[1], values[2], values[3], values[4]}
	return Result{requested.Choice, requested.Confidence, entry.Choice, entry.Confidence, scores, Decide(scores, thresholds), apiResult.Model, apiResult.Usage, apiResult.Answers}, nil
}
