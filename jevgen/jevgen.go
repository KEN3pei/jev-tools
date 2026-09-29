// Package jevgen validates JEV Question Sets and generates typed Go contracts.
package jevgen

import (
	"github.com/KEN3pei/jev-tools/jevgen/internal/diff"
	"github.com/KEN3pei/jev-tools/jevgen/internal/generate"
	"github.com/KEN3pei/jev-tools/jevgen/internal/schema"
)

type QuestionSet = schema.QuestionSet
type Question = schema.Question
type Change = diff.Change

func LoadQuestionSet(path string) (QuestionSet, error) {
	return schema.Load(path)
}

func GenerateGo(set QuestionSet, packageName string) ([]byte, error) {
	return generate.Go(set, packageName)
}

func Compare(oldSet, newSet QuestionSet) []Change {
	return diff.Compare(oldSet, newSet)
}
