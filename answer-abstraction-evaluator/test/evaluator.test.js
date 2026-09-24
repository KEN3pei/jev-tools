import assert from "node:assert/strict";
import test from "node:test";

import {
  buildState,
  decideEvaluation,
  evaluateAnswer
} from "../src/evaluator.js";

test("buildState maps function arguments to the stable Jev state contract", () => {
  assert.deepEqual(
    buildState({
      precedingContext: ["Earlier context"],
      userRequest: "Explain the design space.",
      candidateAnswer: "First install this package."
    }),
    {
      evaluation_task:
        "Infer the requested abstraction level and evaluate whether the answer begins and progresses at an appropriate level. Do not answer the user request.",
      preceding_context: ["Earlier context"],
      user_request: "Explain the design space.",
      candidate_answer: "First install this package."
    }
  );
});

test("decideEvaluation asks for clarification before evaluating structure", () => {
  assert.equal(
    decideEvaluation({
      abstractionMismatch: 0.9,
      prematureSpecificity: 0.9,
      progressiveDisclosure: 0.1,
      prerequisiteFit: 0.1,
      clarificationNeeded: 0.8
    }),
    "ask_clarifying_question"
  );
});

test("decideEvaluation routes premature specificity to entry revision", () => {
  assert.equal(
    decideEvaluation({
      abstractionMismatch: 0.55,
      prematureSpecificity: 0.74,
      progressiveDisclosure: 0.6,
      prerequisiteFit: 0.6,
      clarificationNeeded: 0.2
    }),
    "revise_entry"
  );
});

test("evaluateAnswer supports an injected transport", async () => {
  const fakeFetch = async (_url, init) => {
    const request = JSON.parse(init.body);
    assert.equal(request.state.user_request, "Design question");
    assert.ok(request.questions.premature_specificity);
    return {
      ok: true,
      status: 200,
      async text() {
        return JSON.stringify({
          model: "fake-jev",
          answers: {
            requested_level: {
              choice: "architecture_and_design_space",
              confidence: 0.9,
              probabilities: {}
            },
            answer_entry_level: {
              choice: "implementation_mechanics",
              confidence: 0.8,
              probabilities: {}
            },
            abstraction_mismatch: { noul: 0.8 },
            premature_specificity: { noul: 0.85 },
            progressive_disclosure: { noul: 0.2 },
            prerequisite_fit: { noul: 0.4 },
            clarification_needed: { noul: 0.1 }
          },
          usage: { input_tokens: 100, output_tokens: 20 }
        });
      }
    };
  };

  const result = await evaluateAnswer(
    {
      userRequest: "Design question",
      candidateAnswer: "Implementation-first answer"
    },
    { apiKey: "test-key", fetch: fakeFetch }
  );

  assert.equal(result.decision, "restructure");
  assert.equal(result.requestedLevel, "architecture_and_design_space");
  assert.equal(result.answerEntryLevel, "implementation_mechanics");
});

