import assert from "node:assert/strict";
import test from "node:test";
import { renderStyle } from "../render";
import type { IntentResult } from "../types";

function intent(intent: string, evidenceFit: number | null, confidence: number, assessment: IntentResult["assessment"] = "supported"): IntentResult {
  return { intent, assessment, evidenceFit, confidence, probabilities: { supported: 0.7, weakly_supported: 0.1, contradicted: 0.1, insufficient: 0.1 }, evidence: [] };
}

test("does not color insufficient evidence", () => {
  assert.equal(renderStyle([intent("security", null, 0.9, "insufficient")], 0.3, 0.5, 3), undefined);
});

test("blends multiple visible intents and keeps the strongest first", () => {
  const style = renderStyle([intent("security", 0.9, 0.9), intent("extensibility", 0.8, 0.8)], 0.3, 0.5, 3);
  assert.ok(style);
  assert.equal(style.visible.length, 2);
  assert.equal(style.visible[0].intent.intent, "security");
  assert.match(style.background, /^rgba\(/);
});

test("applies fit and confidence thresholds", () => {
  assert.equal(renderStyle([intent("security", 0.2, 0.9)], 0.3, 0.5, 3), undefined);
  assert.equal(renderStyle([intent("security", 0.9, 0.2)], 0.3, 0.5, 3), undefined);
});
