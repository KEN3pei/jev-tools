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
	"time"

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
	Now     func() time.Time
}

type promptState struct {
	Prompt          string `json:"prompt"`
	RevisionPending bool   `json:"revisionPending"`
}

type LogEntry struct {
	Timestamp           time.Time        `json:"timestamp"`
	SessionID           string           `json:"sessionId,omitempty"`
	TurnID              string           `json:"turnId,omitempty"`
	Attempt             int              `json:"attempt"`
	Decision            string           `json:"decision"`
	Scores              evaluator.Scores `json:"scores,omitempty"`
	RequestedLevel      string           `json:"requestedLevel,omitempty"`
	AnswerEntryLevel    string           `json:"answerEntryLevel,omitempty"`
	CorrectionRequested bool             `json:"correctionRequested"`
	Corrected           bool             `json:"corrected"`
	QuestionSHA256      string           `json:"questionSha256,omitempty"`
	AnswerSHA256        string           `json:"answerSha256,omitempty"`
	Model               string           `json:"model,omitempty"`
	Usage               map[string]any   `json:"usage,omitempty"`
	Error               string           `json:"error,omitempty"`
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
		h.log(LogEntry{Timestamp: h.now(), SessionID: event.SessionID, TurnID: event.TurnID, Decision: "skipped", Error: errorText(err, answer)})
		return Output{Continue: true}
	}
	question := strings.TrimSpace(state.Prompt)
	result, err := h.Client.Evaluate(ctx, evaluator.Input{UserRequest: question, CandidateAnswer: answer})
	if err != nil {
		h.log(LogEntry{Timestamp: h.now(), SessionID: event.SessionID, TurnID: event.TurnID, Decision: "error", QuestionSHA256: digest(question), AnswerSHA256: digest(answer), Error: err.Error()})
		return Output{Continue: true, SystemMessage: "Jev evaluation failed; the answer was not blocked."}
	}

	bad := result.Decision != "pass"
	entry := LogEntry{
		Timestamp: h.now(), SessionID: event.SessionID, TurnID: event.TurnID,
		Attempt: attempt(event.StopHookActive), Decision: result.Decision, Scores: result.Scores,
		RequestedLevel: result.RequestedLevel, AnswerEntryLevel: result.AnswerEntryLevel,
		CorrectionRequested: bad && !event.StopHookActive, Corrected: event.StopHookActive && result.Decision == "pass",
		QuestionSHA256: digest(question), AnswerSHA256: digest(answer), Model: result.Model, Usage: result.Usage,
	}
	h.log(entry)
	if !bad {
		_ = h.writePromptState(event.SessionID, promptState{Prompt: question})
		return Output{Continue: true, SystemMessage: correctionMessage(event.StopHookActive)}
	}
	if event.StopHookActive {
		_ = h.writePromptState(event.SessionID, promptState{Prompt: question})
		return Output{Continue: true, SystemMessage: "Jev evaluation still recommends revision; retry limit reached."}
	}
	_ = h.writePromptState(event.SessionID, promptState{Prompt: question, RevisionPending: true})
	return Output{Decision: "block", Reason: revisionReason(result), SystemMessage: "Jev evaluation requested one answer revision."}
}

func (h Handler) stateDir() string { return filepath.Join(h.DataDir, "state") }
func (h Handler) promptPath(sessionID string) string {
	return filepath.Join(h.stateDir(), safeID(sessionID)+".json")
}
func (h Handler) logPath() string { return filepath.Join(h.DataDir, "evaluations.jsonl") }
func (h Handler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

func (h Handler) log(entry LogEntry) {
	if err := os.MkdirAll(h.DataDir, 0o700); err != nil {
		return
	}
	file, err := os.OpenFile(h.logPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	_ = json.NewEncoder(file).Encode(entry)
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
func attempt(active bool) int {
	if active {
		return 2
	}
	return 1
}
func correctionMessage(active bool) string {
	if active {
		return "Jev evaluation passed after one revision."
	}
	return ""
}
func errorText(err error, answer string) string {
	if err != nil {
		return err.Error()
	}
	if answer == "" {
		return "last_assistant_message is empty"
	}
	return ""
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
