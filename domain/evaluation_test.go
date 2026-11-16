package domain_test

import (
	"errors"
	"testing"
	"time"

	"pos/domain"
)

func TestFactoryCreatesValidEvaluation(t *testing.T) {
	r := newRubric(t)
	s := newStudent(t, "s1", "Иванов")
	comment := must(domain.NewComment("хорошо"))

	e := must(newFactory().Create(r, s, criterionID(t, "hw"), score(t, 10), comment))

	if e.StudentID() != s.ID() || e.CriterionID() != criterionID(t, "hw") {
		t.Fatalf("неверная идентичность оценки: %s", e.ID())
	}
	if e.Score().Value() != 10 || e.Comment().String() != "хорошо" || !e.GradedAt().Equal(now) {
		t.Fatalf("неверные поля оценки: %v %q %v", e.Score(), e.Comment(), e.GradedAt())
	}
}

func TestFactoryEnforcesInvariants(t *testing.T) {
	r := newRubric(t)
	s := newStudent(t, "s1", "Иванов")
	f := newFactory()

	cases := []struct {
		name      string
		rubric    *domain.Rubric
		student   *domain.Student
		criterion string
		score     float64
		want      error
	}{
		{"12 баллов из 10", r, s, "hw", 12, domain.ErrScoreOutOfRange},
		{"101 из 100", r, s, "exam", 101, domain.ErrScoreOutOfRange},
		{"критерия нет в рубрикаторе", r, s, "karma", 1, domain.ErrCriterionNotInRubric},
		{"нет студента", r, nil, "hw", 1, domain.ErrStudentRequired},
		{"пустой рубрикатор", domain.NewRubric(), s, "hw", 1, domain.ErrEmptyRubric},
		{"рубрикатор nil", nil, s, "hw", 1, domain.ErrEmptyRubric},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.Create(tc.rubric, tc.student, criterionID(t, tc.criterion), score(t, tc.score), domain.Comment{})
			if !errors.Is(err, tc.want) {
				t.Fatalf("ожидали %v, получили %v", tc.want, err)
			}
		})
	}
}

func TestFactoryRestoreKeepsDateAndRevalidates(t *testing.T) {
	r := newRubric(t)
	s := newStudent(t, "s1", "Иванов")
	gradedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	e := must(newFactory().Restore(r, s, criterionID(t, "hw"), score(t, 7), domain.Comment{}, gradedAt))
	if !e.GradedAt().Equal(gradedAt) {
		t.Fatalf("дата не сохранена: %v", e.GradedAt())
	}

	_, err := newFactory().Restore(r, s, criterionID(t, "hw"), score(t, 70), domain.Comment{}, gradedAt)
	if !errors.Is(err, domain.ErrScoreOutOfRange) {
		t.Fatalf("испорченные данные должны отклоняться: %v", err)
	}
}

func TestCorrectKeepsInvariants(t *testing.T) {
	r := newRubric(t)
	e := grade(t, r, newStudent(t, "s1", "Иванов"), "hw", 5)
	later := now.Add(time.Hour)

	if err := e.Correct(r, score(t, 11), domain.Comment{}, later); !errors.Is(err, domain.ErrScoreOutOfRange) {
		t.Fatalf("ожидали ErrScoreOutOfRange, получили %v", err)
	}
	if e.Score().Value() != 5 || !e.GradedAt().Equal(now) {
		t.Fatal("неудачное исправление изменило оценку")
	}

	comment := must(domain.NewComment("пересмотр"))
	if err := e.Correct(r, score(t, 8), comment, later); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if e.Score().Value() != 8 || e.Comment().String() != "пересмотр" || !e.GradedAt().Equal(later) {
		t.Fatalf("исправление не применилось: %v %q %v", e.Score(), e.Comment(), e.GradedAt())
	}
}

func TestCorrectRejectsRemovedCriterion(t *testing.T) {
	r := newRubric(t)
	e := grade(t, r, newStudent(t, "s1", "Иванов"), "exam", 50)
	if err := r.RemoveCriterion(criterionID(t, "exam")); err != nil {
		t.Fatal(err)
	}
	if err := e.Correct(r, score(t, 60), domain.Comment{}, now); !errors.Is(err, domain.ErrCriterionNotInRubric) {
		t.Fatalf("ожидали ErrCriterionNotInRubric, получили %v", err)
	}
}
