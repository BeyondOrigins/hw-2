package domain_test

import (
	"testing"
	"time"

	"pos/domain"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

var now = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func newStudent(t *testing.T, id, name string) *domain.Student {
	t.Helper()
	sid := must(domain.NewStudentID(id))
	fullName := must(domain.NewFullName(name))
	return must(domain.NewStudent(sid, fullName))
}

func criterionID(t *testing.T, id string) domain.CriterionID {
	t.Helper()
	return must(domain.NewCriterionID(id))
}

func score(t *testing.T, v float64) domain.Score {
	t.Helper()
	return must(domain.NewScore(v))
}

func addCriterion(t *testing.T, r *domain.Rubric, id, name string, maxScore, weight float64) {
	t.Helper()
	err := r.AddCriterion(
		criterionID(t, id),
		must(domain.NewCriterionName(name)),
		must(domain.NewMaxScore(maxScore)),
		must(domain.NewWeight(weight)),
	)
	if err != nil {
		t.Fatalf("AddCriterion(%s): %v", id, err)
	}
}

// Рубрикатор: ДЗ (max 10, вес 0.6) и экзамен (max 100, вес 0.4).
func newRubric(t *testing.T) *domain.Rubric {
	t.Helper()
	r := domain.NewRubric()
	addCriterion(t, r, "hw", "Домашние задания", 10, 0.6)
	addCriterion(t, r, "exam", "Экзамен", 100, 0.4)
	return r
}

func newFactory() *domain.EvaluationFactory {
	return domain.NewEvaluationFactory(fixedClock{now})
}

func grade(t *testing.T, r *domain.Rubric, s *domain.Student, criterion string, v float64) *domain.Evaluation {
	t.Helper()
	return must(newFactory().Create(r, s, criterionID(t, criterion), score(t, v), domain.Comment{}))
}
