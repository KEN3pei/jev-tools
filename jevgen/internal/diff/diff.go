package diff

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/KEN3pei/jev-tools/jevgen/internal/schema"
)

type Change struct {
	Severity string `json:"severity"`
	Question string `json:"question,omitempty"`
	Message  string `json:"message"`
}

func Compare(oldSet, newSet schema.QuestionSet) []Change {
	var changes []Change
	for id, oldQuestion := range oldSet.Questions {
		newQuestion, ok := newSet.Questions[id]
		if !ok {
			changes = append(changes, Change{"breaking", id, "question removed"})
			continue
		}
		if oldQuestion.Type != newQuestion.Type {
			changes = append(changes, Change{"breaking", id, fmt.Sprintf("type changed from %s to %s", oldQuestion.Type, newQuestion.Type)})
			continue
		}
		if string(oldQuestion.Criteria) != string(newQuestion.Criteria) {
			severity := "semantic"
			if oldQuestion.Type == "choice" {
				severity = choiceSeverity(oldQuestion, newQuestion)
			}
			changes = append(changes, Change{severity, id, "criteria changed"})
		}
		if oldQuestion.Instructions != newQuestion.Instructions {
			changes = append(changes, Change{"semantic", id, "instructions changed"})
		}
	}
	for id := range newSet.Questions {
		if _, ok := oldSet.Questions[id]; !ok {
			changes = append(changes, Change{"additive", id, "question added"})
		}
	}
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Question == changes[j].Question {
			return changes[i].Message < changes[j].Message
		}
		return changes[i].Question < changes[j].Question
	})
	return changes
}

func choiceSeverity(oldQuestion, newQuestion schema.Question) string {
	oldCriteria, _ := oldQuestion.ChoiceCriteria()
	newCriteria, _ := newQuestion.ChoiceCriteria()
	for value := range oldCriteria {
		if _, ok := newCriteria[value]; !ok {
			return "breaking"
		}
	}
	for value := range newCriteria {
		if _, ok := oldCriteria[value]; !ok {
			return "breaking"
		}
	}
	oldJSON, _ := json.Marshal(oldCriteria)
	newJSON, _ := json.Marshal(newCriteria)
	if string(oldJSON) != string(newJSON) {
		return "semantic"
	}
	return "compatible"
}
