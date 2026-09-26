package analyzer

import (
	"context"
	"fmt"

	"github.com/KEN3pei/jev-tools/go-intent-analyzer/analyzer/questions"
)

type Analyzer struct {
	Client      Client
	Language    string
	Task        string
	Definitions []questions.Definition
	Questions   map[string]questions.Question
}

func (a Analyzer) AnalyzeUnit(ctx context.Context, unit UnitContext) ([]IntentResult, error) {
	answers, err := a.Client.Evaluate(ctx, a.buildState(unit), a.Questions)
	if err != nil {
		return nil, err
	}
	return a.buildIntentResults(answers, unit)
}

func (a Analyzer) buildState(unit UnitContext) map[string]any {
	return map[string]any{
		"language": a.Language,
		"task":     a.Task,
		"unit":     map[string]any{"id": unit.ID, "kind": unit.Kind, "name": unit.Name, "declaration": unit.Declaration, "doc_comment": unit.DocComment},
		"callees":  unit.Callees, "callers": unit.Callers, "interfaces": unit.Interfaces,
		"related_tests": unit.RelatedTests, "package_scope": unit.PackageScope,
		"git_history": unit.GitHistory, "intent_relevant_evidence": unit.EvidenceByIntent,
	}
}

func (a Analyzer) buildIntentResults(answers map[string]Answer, unit UnitContext) ([]IntentResult, error) {
	results := make([]IntentResult, 0, len(a.Definitions))
	for _, definition := range a.Definitions {
		answer, ok := answers[definition.ID]
		if !ok {
			return nil, fmt.Errorf("Jev response missing answer %q", definition.ID)
		}
		if answer.Type != "choice" {
			return nil, fmt.Errorf("Jev answer %q has type %q, want choice", definition.ID, answer.Type)
		}
		if _, ok := questions.AssessmentCriteria[answer.Choice]; !ok {
			return nil, fmt.Errorf("Jev answer %q returned unknown assessment %q", definition.ID, answer.Choice)
		}
		for choice := range questions.AssessmentCriteria {
			if _, ok := answer.Probabilities[choice]; !ok {
				return nil, fmt.Errorf("Jev answer %q missing probability %q", definition.ID, choice)
			}
		}
		var evidenceFit *float64
		var evidence []Evidence
		if answer.Choice != "insufficient" {
			fit := answer.Probabilities["supported"] + 0.5*answer.Probabilities["weakly_supported"]
			if fit > 1 {
				fit = 1
			}
			evidenceFit = &fit
			evidence = append(evidence, unit.EvidenceByIntent[definition.ID]...)
		}
		results = append(results, IntentResult{Intent: definition.ID, Assessment: answer.Choice, EvidenceFit: evidenceFit, Confidence: answer.Confidence, Probabilities: answer.Probabilities, Evidence: evidence})
	}
	return results, nil
}
