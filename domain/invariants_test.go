package domain_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"pos/domain"
)

func errOf[T any](_ T, err error) error { return err }

func TestValueObjectsRejectInvalidInput(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want error
	}{
		{"пустой ID студента", errOf(domain.NewStudentID("   ")), domain.ErrEmptyID},
		{"пустой ID критерия", errOf(domain.NewCriterionID("")), domain.ErrEmptyID},
		{"пустое ФИО", errOf(domain.NewFullName(" \t ")), domain.ErrEmptyName},
		{"слишком длинное ФИО", errOf(domain.NewFullName(strings.Repeat("я", 201))), domain.ErrTooLong},
		{"пустое название критерия", errOf(domain.NewCriterionName("")), domain.ErrEmptyName},
		{"слишком длинный комментарий", errOf(domain.NewComment(strings.Repeat("a", 501))), domain.ErrTooLong},
		{"отрицательный балл", errOf(domain.NewScore(-0.5)), domain.ErrInvalidScore},
		{"балл NaN", errOf(domain.NewScore(math.NaN())), domain.ErrInvalidScore},
		{"балл +Inf", errOf(domain.NewScore(math.Inf(1))), domain.ErrInvalidScore},
		{"нулевой максимум", errOf(domain.NewMaxScore(0)), domain.ErrInvalidMaxScore},
		{"отрицательный максимум", errOf(domain.NewMaxScore(-10)), domain.ErrInvalidMaxScore},
		{"нулевой вес", errOf(domain.NewWeight(0)), domain.ErrInvalidWeight},
		{"отрицательный вес", errOf(domain.NewWeight(-1)), domain.ErrInvalidWeight},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !errors.Is(tc.err, tc.want) {
				t.Fatalf("ожидали %v, получили %v", tc.want, tc.err)
			}
		})
	}
}

func TestValueObjectsNormalizeInput(t *testing.T) {
	name := must(domain.NewFullName("  Иванов   Иван  "))
	if name.String() != "Иванов Иван" {
		t.Fatalf("ФИО не нормализовано: %q", name)
	}
	if score(t, 0).Value() != 0 {
		t.Fatal("ноль — допустимый балл")
	}
	if must(domain.NewComment("")).String() != "" {
		t.Fatal("пустой комментарий допустим")
	}
}

func TestStudentRequiresValidValues(t *testing.T) {
	name := must(domain.NewFullName("Иванов"))
	if _, err := domain.NewStudent(domain.StudentID{}, name); !errors.Is(err, domain.ErrEmptyID) {
		t.Fatalf("ожидали ErrEmptyID, получили %v", err)
	}
	s := newStudent(t, "s1", "Иванов")
	if err := s.Rename(domain.FullName{}); !errors.Is(err, domain.ErrEmptyName) {
		t.Fatalf("ожидали ErrEmptyName, получили %v", err)
	}
	if s.Name().String() != "Иванов" {
		t.Fatal("неудачное переименование изменило ФИО")
	}
}

func TestRubricRejectsDuplicates(t *testing.T) {
	r := newRubric(t)
	maxScore := must(domain.NewMaxScore(5))
	weight := must(domain.NewWeight(1))

	err := r.AddCriterion(criterionID(t, "hw"), must(domain.NewCriterionName("Другое")), maxScore, weight)
	if !errors.Is(err, domain.ErrDuplicateCriterion) {
		t.Fatalf("дубль ID: ожидали ErrDuplicateCriterion, получили %v", err)
	}
	err = r.AddCriterion(criterionID(t, "new"), must(domain.NewCriterionName("ЭКЗАМЕН")), maxScore, weight)
	if !errors.Is(err, domain.ErrDuplicateCriterion) {
		t.Fatalf("дубль названия: ожидали ErrDuplicateCriterion, получили %v", err)
	}
	err = r.RenameCriterion(criterionID(t, "hw"), must(domain.NewCriterionName("экзамен")))
	if !errors.Is(err, domain.ErrDuplicateCriterion) {
		t.Fatalf("переименование в существующее: ожидали ErrDuplicateCriterion, получили %v", err)
	}
	if len(r.Criteria()) != 2 {
		t.Fatalf("рубрикатор изменился: %d критериев", len(r.Criteria()))
	}
}

func TestRubricRejectsZeroValueObjects(t *testing.T) {
	r := domain.NewRubric()
	name := must(domain.NewCriterionName("X"))
	err := r.AddCriterion(criterionID(t, "x"), name, domain.MaxScore{}, must(domain.NewWeight(1)))
	if !errors.Is(err, domain.ErrInvalidMaxScore) {
		t.Fatalf("ожидали ErrInvalidMaxScore, получили %v", err)
	}
	err = r.AddCriterion(criterionID(t, "x"), name, must(domain.NewMaxScore(1)), domain.Weight{})
	if !errors.Is(err, domain.ErrInvalidWeight) {
		t.Fatalf("ожидали ErrInvalidWeight, получили %v", err)
	}
}

func TestRubricOperationsOnMissingCriterion(t *testing.T) {
	r := newRubric(t)
	missing := criterionID(t, "missing")
	if _, err := r.Criterion(missing); !errors.Is(err, domain.ErrCriterionNotInRubric) {
		t.Fatalf("Criterion: %v", err)
	}
	if err := r.ChangeWeight(missing, must(domain.NewWeight(1))); !errors.Is(err, domain.ErrCriterionNotInRubric) {
		t.Fatalf("ChangeWeight: %v", err)
	}
	if err := r.RemoveCriterion(missing); !errors.Is(err, domain.ErrCriterionNotInRubric) {
		t.Fatalf("RemoveCriterion: %v", err)
	}
}

func TestRubricCriteriaReturnsCopy(t *testing.T) {
	r := newRubric(t)
	criteria := r.Criteria()
	criteria[0] = domain.Criterion{}
	if r.Criteria()[0].ID().String() != "hw" {
		t.Fatal("изменение копии повлияло на рубрикатор")
	}
}

func TestCriterionScoreBounds(t *testing.T) {
	c := must(newRubric(t).Criterion(criterionID(t, "hw")))
	for _, v := range []float64{0, 5, 10} {
		if err := c.CheckScore(score(t, v)); err != nil {
			t.Errorf("балл %v должен быть допустим: %v", v, err)
		}
	}
	if err := c.CheckScore(score(t, 10.01)); !errors.Is(err, domain.ErrScoreOutOfRange) {
		t.Errorf("ожидали ErrScoreOutOfRange, получили %v", err)
	}
}
