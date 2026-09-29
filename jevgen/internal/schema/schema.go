package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type QuestionSet struct {
	Schema        string              `json:"$schema,omitempty"`
	SchemaVersion string              `json:"schemaVersion"`
	Name          string              `json:"name"`
	Questions     map[string]Question `json:"questions"`
}

type Question struct {
	Type         string          `json:"type"`
	Instructions string          `json:"instructions"`
	Criteria     json.RawMessage `json:"criteria,omitempty"`
}

func Load(path string) (QuestionSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return QuestionSet{}, err
	}
	var set QuestionSet
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&set); err != nil {
		return QuestionSet{}, fmt.Errorf("decode question set: %w", err)
	}
	if err := set.Validate(); err != nil {
		return QuestionSet{}, err
	}
	return set, nil
}

func (s QuestionSet) Validate() error {
	var problems []string
	if strings.TrimSpace(s.Name) == "" {
		problems = append(problems, "name is required")
	}
	if strings.TrimSpace(s.SchemaVersion) == "" {
		problems = append(problems, "schemaVersion is required")
	}
	if len(s.Questions) == 0 {
		problems = append(problems, "questions must not be empty")
	}
	ids := make([]string, 0, len(s.Questions))
	for id := range s.Questions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		question := s.Questions[id]
		if !identifierPattern.MatchString(id) {
			problems = append(problems, fmt.Sprintf("question %q: ID must match %s", id, identifierPattern))
		}
		if strings.TrimSpace(question.Instructions) == "" {
			problems = append(problems, fmt.Sprintf("question %q: instructions are required", id))
		}
		switch question.Type {
		case "choice":
			criteria, err := question.ChoiceCriteria()
			if err != nil || len(criteria) == 0 {
				problems = append(problems, fmt.Sprintf("question %q: choice criteria must be a non-empty string map", id))
			}
			for value, description := range criteria {
				if !identifierPattern.MatchString(value) || strings.TrimSpace(description) == "" {
					problems = append(problems, fmt.Sprintf("question %q: invalid choice criterion %q", id, value))
				}
			}
		case "score":
			var criteria []string
			if err := json.Unmarshal(question.Criteria, &criteria); err != nil || len(criteria) < 2 {
				problems = append(problems, fmt.Sprintf("question %q: score criteria must contain at least two strings", id))
			}
			for index, value := range criteria {
				if strings.TrimSpace(value) == "" {
					problems = append(problems, fmt.Sprintf("question %q: score criterion %d is empty", id, index))
				}
			}
		case "noul":
			if len(question.Criteria) > 0 && string(question.Criteria) != "null" {
				var criteria map[string]string
				if err := json.Unmarshal(question.Criteria, &criteria); err != nil {
					problems = append(problems, fmt.Sprintf("question %q: noul criteria must be a string map when present", id))
				}
			}
		default:
			problems = append(problems, fmt.Sprintf("question %q: unsupported type %q", id, question.Type))
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("invalid question set:\n- %s", strings.Join(problems, "\n- "))
	}
	return nil
}

func (q Question) ChoiceCriteria() (map[string]string, error) {
	var criteria map[string]string
	if err := json.Unmarshal(q.Criteria, &criteria); err != nil {
		return nil, err
	}
	return criteria, nil
}

func (s QuestionSet) Hash() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
