package hook

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/KEN3pei/jev-tools/answer-abstraction-evaluator/evaluator"
)

type fakeClient struct{ result evaluator.Result }

func (f fakeClient) Evaluate(context.Context, evaluator.Input) (evaluator.Result, error) {
	return f.result, nil
}

func TestStopRequestsOneRevisionAndLogs(t *testing.T) {
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
	data, err := os.ReadFile(h.logPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"correctionRequested":true`) {
		t.Fatalf("unexpected log: %s", data)
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
