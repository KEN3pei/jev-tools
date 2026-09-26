import type { IntentResult } from "./types";
import { INTENT_COLORS } from "./intents";

export interface RGB { r: number; g: number; b: number }
export interface RenderedIntent { intent: IntentResult; weight: number; color: RGB }
export interface RenderStyle { background: string; border: string; visible: RenderedIntent[] }

export function renderStyle(intents: IntentResult[], minimumFit: number, minimumConfidence: number, maxVisible: number): RenderStyle | undefined {
  const visible = intents
    .filter((intent) => intent.assessment !== "insufficient" && intent.evidenceFit !== null && intent.evidenceFit >= minimumFit && intent.confidence >= minimumConfidence)
    .map((intent) => ({ intent, weight: intent.evidenceFit! * intent.confidence, color: parseHex(INTENT_COLORS[intent.intent] ?? "#78909C") }))
    .sort((a, b) => b.weight - a.weight)
    .slice(0, maxVisible);
  if (visible.length === 0) return undefined;

  const total = visible.reduce((sum, item) => sum + item.weight, 0);
  const mixed = visible.reduce<RGB>((result, item) => ({
    r: result.r + item.color.r * item.weight / total,
    g: result.g + item.color.g * item.weight / total,
    b: result.b + item.color.b * item.weight / total
  }), { r: 0, g: 0, b: 0 });
  const strength = Math.max(...visible.map((item) => item.weight));
  const alpha = clamp(0.06 + strength * 0.16, 0.06, 0.22);
  const dominant = visible[0].color;
  return {
    background: `rgba(${Math.round(mixed.r)}, ${Math.round(mixed.g)}, ${Math.round(mixed.b)}, ${alpha.toFixed(3)})`,
    border: `rgb(${dominant.r}, ${dominant.g}, ${dominant.b})`,
    visible
  };
}

function parseHex(value: string): RGB {
  return { r: Number.parseInt(value.slice(1, 3), 16), g: Number.parseInt(value.slice(3, 5), 16), b: Number.parseInt(value.slice(5, 7), 16) };
}

function clamp(value: number, min: number, max: number): number { return Math.min(max, Math.max(min, value)); }
