package seed_test

import (
	"strings"
	"testing"

	"easyoffer/question/internal/seed"
)

func TestSystemQuestionsCountIsAround150(t *testing.T) {
	questions := seed.SystemQuestions()
	if len(questions) < 145 || len(questions) > 155 {
		t.Fatalf("expected around 150 questions, got %d", len(questions))
	}
}

func TestSystemQuestionsContainNoDuplicateTitleContentPairs(t *testing.T) {
	questions := seed.SystemQuestions()
	seen := make(map[string]struct{}, len(questions))

	for i := range questions {
		key := strings.TrimSpace(questions[i].Title) + "|" + strings.TrimSpace(questions[i].Content)
		if _, ok := seen[key]; ok {
			t.Fatalf("duplicate question found: %q", questions[i].Title)
		}
		seen[key] = struct{}{}
	}
}

func TestSystemCodeQuestionsHaveStarterCode(t *testing.T) {
	questions := seed.SystemQuestions()

	for i := range questions {
		if questions[i].AnswerFormat != "code" {
			continue
		}
		if strings.TrimSpace(questions[i].StarterCode) == "" {
			t.Fatalf("code question %q has empty starter code", questions[i].Title)
		}
	}
}
