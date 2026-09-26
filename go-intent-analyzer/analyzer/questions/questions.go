package questions

type Question struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}

type Definition struct {
	ID          string
	Instruction string
	Keywords    []string
}

var AssessmentCriteria = map[string]string{
	"supported":        "At least two independent evidence sources consistently indicate this deliberate intent.",
	"weakly_supported": "Some evidence indicates this intent, but it is limited to one source or remains ambiguous.",
	"contradicted":     "Evidence about the intended priority conflicts with other evidence. Do not use this merely because execution appears imperfect.",
	"insufficient":     "The supplied context does not contain enough evidence to infer this intent.",
}

func Build(definitions []Definition) map[string]Question {
	result := make(map[string]Question, len(definitions))
	for _, definition := range definitions {
		result[definition.ID] = Question{
			Type:         "choice",
			Instructions: definition.Instruction + " Judge only intent, not implementation quality or whether the intent was successfully realized.",
			Criteria:     AssessmentCriteria,
		}
	}
	return result
}
