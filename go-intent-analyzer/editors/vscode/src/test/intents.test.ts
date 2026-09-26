import assert from "node:assert/strict";
import test from "node:test";
import { INTENT_COLORS, INTENT_VISUALS } from "../intents";

test("defines one unique color for every intent category", () => {
  assert.equal(INTENT_VISUALS.length, 11);
  assert.equal(new Set(INTENT_VISUALS.map((intent) => intent.id)).size, 11);
  assert.equal(new Set(INTENT_VISUALS.map((intent) => intent.color)).size, 11);
  for (const intent of INTENT_VISUALS) {
    assert.match(intent.color, /^#[0-9A-F]{6}$/);
    assert.equal(INTENT_COLORS[intent.id], intent.color);
  }
});
