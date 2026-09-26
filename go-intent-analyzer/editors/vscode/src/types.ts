export interface Position { line: number; character: number }
export interface SourceRange { start: Position; end: Position }

export interface Evidence {
  id: string;
  kind: "source" | "test" | "commit";
  uri?: string;
  range?: SourceRange;
  revision?: string;
  signal: string;
}

export type Assessment = "supported" | "weakly_supported" | "contradicted" | "insufficient";

export interface IntentResult {
  intent: string;
  assessment: Assessment;
  evidenceFit: number | null;
  confidence: number;
  probabilities: Record<Assessment, number>;
  evidence: Evidence[];
}

export interface UnitReport {
  id: string;
  uri: string;
  range: SourceRange;
  kind: "function" | "method" | "type" | "package";
  name: string;
  intents: IntentResult[];
}

export interface AnalysisReport {
  schemaVersion: string;
  analyzerVersion: string;
  generatedAt: string;
  repository: { root: string; revision?: string };
  units: UnitReport[];
  errors?: Array<{ unitId: string; message: string }>;
}
