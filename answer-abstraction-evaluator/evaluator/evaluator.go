package evaluator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/KEN3pei/jev-tools/answer-abstraction-evaluator/contract"
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

type Thresholds struct {
	EvaluationApplicableMinimum  float64
	ClarificationNeeded          float64
	AbstractionMismatch          float64
	PrematureSpecificity         float64
	ProgressiveDisclosureMinimum float64
	PrerequisiteFitMinimum       float64
}

var DefaultThresholds = Thresholds{
	EvaluationApplicableMinimum:  0.5,
	ClarificationNeeded:          0.7,
	AbstractionMismatch:          0.7,
	PrematureSpecificity:         0.65,
	ProgressiveDisclosureMinimum: 0.5,
	PrerequisiteFitMinimum:       0.3,
}

type Scores struct {
	EvaluationApplicable  float64 `json:"evaluationApplicable"`
	AbstractionMismatch   float64 `json:"abstractionMismatch"`
	PrematureSpecificity  float64 `json:"prematureSpecificity"`
	ProgressiveDisclosure float64 `json:"progressiveDisclosure"`
	PrerequisiteFit       float64 `json:"prerequisiteFit"`
	ClarificationNeeded   float64 `json:"clarificationNeeded"`
}

type Result struct {
	RequestedLevel             string           `json:"requestedLevel"`
	RequestedLevelConfidence   float64          `json:"requestedLevelConfidence"`
	AnswerEntryLevel           string           `json:"answerEntryLevel"`
	AnswerEntryLevelConfidence float64          `json:"answerEntryLevelConfidence"`
	Scores                     Scores           `json:"scores"`
	Decision                   string           `json:"decision"`
	Model                      string           `json:"model,omitempty"`
	Usage                      map[string]any   `json:"usage,omitempty"`
	RawAnswers                 contract.Answers `json:"rawAnswers"`
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
	if s.EvaluationApplicable < t.EvaluationApplicableMinimum {
		return "not_applicable"
	}
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
	payload, err := json.Marshal(map[string]any{"model": model, "state": state, "questions": contract.Questions()})
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
		Answers json.RawMessage `json:"answers"`
		Model   string          `json:"model"`
		Usage   map[string]any  `json:"usage"`
	}
	if err := json.Unmarshal(body, &apiResult); err != nil {
		return Result{}, fmt.Errorf("decode Jev response: %w", err)
	}
	answers, err := contract.DecodeAnswers(apiResult.Answers)
	if err != nil {
		return Result{}, fmt.Errorf("decode Jev answers: %w", err)
	}
	scores := Scores{
		EvaluationApplicable:  answers.EvaluationApplicable.Noul,
		AbstractionMismatch:   answers.AbstractionMismatch.Noul,
		PrematureSpecificity:  answers.PrematureSpecificity.Noul,
		ProgressiveDisclosure: answers.ProgressiveDisclosure.Noul,
		PrerequisiteFit:       answers.PrerequisiteFit.Noul,
		ClarificationNeeded:   answers.ClarificationNeeded.Noul,
	}
	return Result{string(answers.RequestedLevel.Choice), answers.RequestedLevel.Confidence, string(answers.AnswerEntryLevel.Choice), answers.AnswerEntryLevel.Confidence, scores, Decide(scores, thresholds), apiResult.Model, apiResult.Usage, answers}, nil
}
