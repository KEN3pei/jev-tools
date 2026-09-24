package hook

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/KEN3pei/jev-tools/answer-abstraction-evaluator/evaluator"
)

type Event struct {
	SessionID            string  `json:"session_id"`
	TurnID               string  `json:"turn_id"`
	HookEventName        string  `json:"hook_event_name"`
	Prompt               string  `json:"prompt"`
	StopHookActive       bool    `json:"stop_hook_active"`
	LastAssistantMessage *string `json:"last_assistant_message"`
}

type Output struct {
	Continue      bool   `json:"continue,omitempty"`
	Decision      string `json:"decision,omitempty"`
	Reason        string `json:"reason,omitempty"`
	SystemMessage string `json:"systemMessage,omitempty"`
}

type EvaluationClient interface {
	Evaluate(context.Context, evaluator.Input) (evaluator.Result, error)
}

type Handler struct {
	Client  EvaluationClient
	DataDir string
}

type promptState struct {
	Prompt          string `json:"prompt"`
	RevisionPending bool   `json:"revisionPending"`
}

func (h Handler) CapturePrompt(event Event) error {
	if strings.TrimSpace(event.Prompt) == "" {
		return nil
	}
	if state, err := h.readPromptState(event.SessionID); err == nil && state.RevisionPending {
		return nil
	}
	if err := os.MkdirAll(h.stateDir(), 0o700); err != nil {
		return err
	}
	return h.writePromptState(event.SessionID, promptState{Prompt: event.Prompt})
}

func (h Handler) EvaluateStop(ctx context.Context, event Event) Output {
	answer := ""
	if event.LastAssistantMessage != nil {
		answer = strings.TrimSpace(*event.LastAssistantMessage)
	}
	state, err := h.readPromptState(event.SessionID)
	if err != nil || answer == "" {
		return Output{Continue: true, SystemMessage: "Jev evaluation skipped: no matching question or answer."}
	}
	question := strings.TrimSpace(state.Prompt)
	result, err := h.Client.Evaluate(ctx, evaluator.Input{UserRequest: question, CandidateAnswer: answer})
	if err != nil {
		return Output{Continue: true, SystemMessage: "Jev evaluation failed; the answer was not blocked."}
	}

	bad := result.Decision != "pass"
	if !bad {
		_ = os.Remove(h.promptPath(event.SessionID))
		return Output{Continue: true, SystemMessage: evaluationMessage(result, event.StopHookActive)}
	}
	if event.StopHookActive {
		_ = os.Remove(h.promptPath(event.SessionID))
		return Output{Continue: true, SystemMessage: "Jev evaluation still recommends revision; retry limit reached."}
	}
	_ = h.writePromptState(event.SessionID, promptState{Prompt: question, RevisionPending: true})
	return Output{Decision: "block", Reason: revisionReason(result), SystemMessage: "Jev evaluation requested one answer revision."}
}

func (h Handler) stateDir() string { return filepath.Join(h.DataDir, "state") }
func (h Handler) promptPath(sessionID string) string {
	return filepath.Join(h.stateDir(), safeID(sessionID)+".json")
}

func (h Handler) readPromptState(sessionID string) (promptState, error) {
	data, err := os.ReadFile(h.promptPath(sessionID))
	if err != nil {
		return promptState{}, err
	}
	var state promptState
	if err := json.Unmarshal(data, &state); err != nil {
		return promptState{}, err
	}
	return state, nil
}

func (h Handler) writePromptState(sessionID string, state promptState) error {
	if err := os.MkdirAll(h.stateDir(), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(h.promptPath(sessionID), data, 0o600)
}

func safeID(value string) string {
	if value == "" {
		value = "unknown"
	}
	return digest(value)[:24]
}
func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func evaluationMessage(result evaluator.Result, active bool) string {
	if active {
		return fmt.Sprintf("Jev evaluation: pass after one revision (requested=%s, answer=%s).", result.RequestedLevel, result.AnswerEntryLevel)
	}
	return fmt.Sprintf("Jev evaluation: pass (requested=%s, answer=%s).", result.RequestedLevel, result.AnswerEntryLevel)
}

func revisionReason(result evaluator.Result) string {
	switch result.Decision {
	case "ask_clarifying_question":
		return "The requested abstraction level is ambiguous. Ask one concise clarifying question instead of assuming it."
	case "restructure":
		return fmt.Sprintf("Rewrite the answer from the requested level (%s). It currently begins at %s. Establish the requested conceptual or architectural frame before implementation details.", result.RequestedLevel, result.AnswerEntryLevel)
	default:
		return "Revise the opening and explanation order. Establish the requested mental model before introducing concrete products, APIs, configuration, or code."
	}
}
