package hook

import (
	"context"
	"strings"
	"testing"

	"github.com/KEN3pei/jev-tools/answer-abstraction-evaluator/evaluator"
)

type fakeClient struct{ result evaluator.Result }

func (f fakeClient) Evaluate(context.Context, evaluator.Input) (evaluator.Result, error) {
	return f.result, nil
}

func TestNotApplicableIsDisplayedWithoutRevision(t *testing.T) {
	dir := t.TempDir()
	h := Handler{DataDir: dir, Client: fakeClient{result: evaluator.Result{Decision: "not_applicable"}}}
	if err := h.CapturePrompt(Event{SessionID: "s1", Prompt: "give me the command"}); err != nil {
		t.Fatal(err)
	}
	answer := "run this command"
	out := h.EvaluateStop(context.Background(), Event{SessionID: "s1", LastAssistantMessage: &answer})
	if !out.Continue || out.Decision == "block" || out.SystemMessage != "Jev evaluation: not applicable." {
		t.Fatalf("got %#v", out)
	}
}

func TestStopRequestsOneRevision(t *testing.T) {
	dir := t.TempDir()
	h := Handler{DataDir: dir, Client: fakeClient{result: evaluator.Result{Decision: "restructure", RequestedLevel: "architecture_and_design_space", AnswerEntryLevel: "implementation_mechanics"}}}
	if err := h.CapturePrompt(Event{SessionID: "s1", Prompt: "What designs exist?"}); err != nil {
		t.Fatal(err)
	}
	answer := "Install this package."
	out := h.EvaluateStop(context.Background(), Event{SessionID: "s1", TurnID: "t1", LastAssistantMessage: &answer})
	if out.Decision != "block" {
		t.Fatalf("got %#v", out)
	}
}

func TestStopDoesNotLoop(t *testing.T) {
	dir := t.TempDir()
	h := Handler{DataDir: dir, Client: fakeClient{result: evaluator.Result{Decision: "restructure"}}}
	if err := h.CapturePrompt(Event{SessionID: "s1", Prompt: "question"}); err != nil {
		t.Fatal(err)
	}
	answer := "answer"
	out := h.EvaluateStop(context.Background(), Event{SessionID: "s1", StopHookActive: true, LastAssistantMessage: &answer})
	if !out.Continue || out.Decision == "block" {
		t.Fatalf("got %#v", out)
	}
}

func TestAlignedLevelsProduceNonContradictoryReason(t *testing.T) {
	result := evaluator.Result{Decision: "restructure", RequestedLevel: "implementation_mechanics", AnswerEntryLevel: "implementation_mechanics"}
	reason := revisionReason(result)
	if strings.Contains(reason, "It currently begins at") {
		t.Fatalf("contradictory reason: %s", reason)
	}
}

func TestContinuationPromptDoesNotReplaceOriginalQuestion(t *testing.T) {
	dir := t.TempDir()
	h := Handler{DataDir: dir, Client: fakeClient{result: evaluator.Result{Decision: "restructure"}}}
	if err := h.CapturePrompt(Event{SessionID: "s1", Prompt: "original question"}); err != nil {
		t.Fatal(err)
	}
	answer := "first answer"
	h.EvaluateStop(context.Background(), Event{SessionID: "s1", LastAssistantMessage: &answer})
	if err := h.CapturePrompt(Event{SessionID: "s1", Prompt: "automatic revision instruction"}); err != nil {
		t.Fatal(err)
	}
	state, err := h.readPromptState("s1")
	if err != nil {
		t.Fatal(err)
	}
	if state.Prompt != "original question" {
		t.Fatalf("prompt was overwritten: %q", state.Prompt)
	}
}

func TestPassIsDisplayed(t *testing.T) {
	dir := t.TempDir()
	h := Handler{DataDir: dir, Client: fakeClient{result: evaluator.Result{Decision: "pass", RequestedLevel: "conceptual_orientation", AnswerEntryLevel: "conceptual_orientation"}}}
	if err := h.CapturePrompt(Event{SessionID: "s1", Prompt: "question"}); err != nil {
		t.Fatal(err)
	}
	answer := "answer"
	out := h.EvaluateStop(context.Background(), Event{SessionID: "s1", LastAssistantMessage: &answer})
	if out.SystemMessage == "" {
		t.Fatal("pass result should be displayed")
	}
}
