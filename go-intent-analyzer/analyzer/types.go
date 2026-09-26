package analyzer

import "time"

const SchemaVersion = "1.0"

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Evidence is an input signal relevant to an intent assessment. It does not
// claim that Jev exposed an internal chain of reasoning.
type Evidence struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"` // source | test | commit
	URI      string `json:"uri,omitempty"`
	Range    *Range `json:"range,omitempty"`
	Revision string `json:"revision,omitempty"`
	Signal   string `json:"signal"`
}

type IntentResult struct {
	Intent        string             `json:"intent"`
	Assessment    string             `json:"assessment"`
	EvidenceFit   *float64           `json:"evidenceFit"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
	Evidence      []Evidence         `json:"evidence"`
}

type UnitReport struct {
	ID      string         `json:"id"`
	URI     string         `json:"uri"`
	Range   Range          `json:"range"`
	Kind    string         `json:"kind"`
	Name    string         `json:"name"`
	Intents []IntentResult `json:"intents"`
}

type Repository struct {
	Root     string `json:"root"`
	Revision string `json:"revision,omitempty"`
}

type Report struct {
	SchemaVersion   string          `json:"schemaVersion"`
	AnalyzerVersion string          `json:"analyzerVersion"`
	GeneratedAt     time.Time       `json:"generatedAt"`
	Repository      Repository      `json:"repository"`
	Units           []UnitReport    `json:"units"`
	Errors          []AnalysisError `json:"errors,omitempty"`
}

type AnalysisError struct {
	UnitID  string `json:"unitId"`
	Message string `json:"message"`
}

type Answer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
}

type UnitContext struct {
	ID               string
	Kind             string
	Name             string
	URI              string
	Range            Range
	Declaration      string
	DocComment       string
	Callees          []CallRef
	Callers          []CallRef
	Interfaces       []InterfaceRef
	RelatedTests     []TestRef
	PackageScope     []string
	GitHistory       []GitEntry
	EvidenceByIntent map[string][]Evidence
}

type CallRef struct {
	Name    string `json:"name"`
	URI     string `json:"uri,omitempty"`
	Range   Range  `json:"range,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}
type InterfaceRef struct {
	Name    string `json:"name"`
	URI     string `json:"uri"`
	Range   Range  `json:"range"`
	Snippet string `json:"snippet"`
}
type TestRef struct {
	Name    string `json:"name"`
	URI     string `json:"uri"`
	Range   Range  `json:"range"`
	Snippet string `json:"snippet"`
}
type GitEntry struct {
	Revision string `json:"revision"`
	Message  string `json:"message"`
}
