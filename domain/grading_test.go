package domain_test

import (
	"errors"
	"math"
	"testing"

	"pos/domain"
)

func assertGrade(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("итог %v, ожидали %v", got, want)
	}
}

func TestCalculatorPolicies(t *testing.T) {
	r := newRubric(t)
	s := newStudent(t, "s1", "Иванов")
	evaluations := []*domain.Evaluation{grade(t, r, s, "hw", 8)}

	cases := []struct {
		name   string
		policy domain.GradingPolicy
		want   float64
	}{
		// 10 · (0.6·0.8 + 0.4·0) / (0.6 + 0.4)
		{"нет оценки = 0", domain.WeightedAveragePolicy{}, 4.8},
		// 10 · (0.6·0.8) / 0.6
		{"только оценённые", domain.GradedOnlyPolicy{}, 8},
		{"нет оценки = 0 с округлением", domain.NewRoundedPolicy(domain.WeightedAveragePolicy{}, 0), 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := must(domain.NewFinalGradeCalculator(tc.policy).Calculate(s, r, evaluations))
			assertGrade(t, g.Value(), tc.want)
			if g.Policy() != tc.policy.Name() {
				t.Fatalf("в итоге записано правило %q", g.Policy())
			}
		})
	}
}

func TestCalculatorBreakdown(t *testing.T) {
	r := newRubric(t)
	s := newStudent(t, "s1", "Иванов")
	g := must(domain.NewFinalGradeCalculator(domain.WeightedAveragePolicy{}).
		Calculate(s, r, []*domain.Evaluation{grade(t, r, s, "exam", 50)}))

	lines := g.Lines()
	if len(lines) != 2 || g.GradedCount() != 1 || g.Student() != s.ID() {
		t.Fatalf("неверная расшифровка: %+v", lines)
	}
	if lines[0].Criterion.ID().String() != "hw" || lines[0].Graded {
		t.Fatalf("ДЗ должно быть первым и неоценённым: %+v", lines[0])
	}
	if !lines[1].Graded || lines[1].Score.Value() != 50 {
		t.Fatalf("экзамен должен быть оценён на 50: %+v", lines[1])
	}
}

func TestCalculatorBoundaryGrades(t *testing.T) {
	r := newRubric(t)
	s := newStudent(t, "s1", "Иванов")
	calc := domain.NewFinalGradeCalculator(domain.WeightedAveragePolicy{})

	full := []*domain.Evaluation{grade(t, r, s, "hw", 10), grade(t, r, s, "exam", 100)}
	assertGrade(t, must(calc.Calculate(s, r, full)).Value(), 10)

	assertGrade(t, must(calc.Calculate(s, r, nil)).Value(), 0)
	assertGrade(t, must(domain.NewFinalGradeCalculator(domain.GradedOnlyPolicy{}).Calculate(s, r, nil)).Value(), 0)
}

func TestCalculatorWeightsAreNormalized(t *testing.T) {
	r := domain.NewRubric()
	addCriterion(t, r, "hw", "ДЗ", 10, 3)
	addCriterion(t, r, "exam", "Экзамен", 100, 2)
	s := newStudent(t, "s1", "Иванов")
	evaluations := []*domain.Evaluation{grade(t, r, s, "hw", 8), grade(t, r, s, "exam", 50)}

	g := must(domain.NewFinalGradeCalculator(domain.WeightedAveragePolicy{}).Calculate(s, r, evaluations))
	assertGrade(t, g.Value(), 6.8) // 10 · (0.6·0.8 + 0.4·0.5)
}

func TestCalculatorFollowsWeightChange(t *testing.T) {
	r := newRubric(t)
	s := newStudent(t, "s1", "Иванов")
	evaluations := []*domain.Evaluation{grade(t, r, s, "hw", 10), grade(t, r, s, "exam", 0)}
	calc := domain.NewFinalGradeCalculator(domain.WeightedAveragePolicy{})

	assertGrade(t, must(calc.Calculate(s, r, evaluations)).Value(), 6)
	if err := r.ChangeWeight(criterionID(t, "exam"), must(domain.NewWeight(0.6))); err != nil {
		t.Fatal(err)
	}
	assertGrade(t, must(calc.Calculate(s, r, evaluations)).Value(), 5)
}

func TestCalculatorRejectsInconsistentEvaluations(t *testing.T) {
	s := newStudent(t, "s1", "Иванов")
	other := newStudent(t, "s2", "Петров")
	calc := domain.NewFinalGradeCalculator(domain.WeightedAveragePolicy{})

	t.Run("оценка другого студента", func(t *testing.T) {
		r := newRubric(t)
		_, err := calc.Calculate(s, r, []*domain.Evaluation{grade(t, r, other, "hw", 5)})
		if !errors.Is(err, domain.ErrForeignEvaluation) {
			t.Fatalf("ожидали ErrForeignEvaluation, получили %v", err)
		}
	})

	t.Run("две оценки по одному критерию", func(t *testing.T) {
		r := newRubric(t)
		_, err := calc.Calculate(s, r, []*domain.Evaluation{grade(t, r, s, "hw", 5), grade(t, r, s, "hw", 6)})
		if !errors.Is(err, domain.ErrDuplicateEvaluation) {
			t.Fatalf("ожидали ErrDuplicateEvaluation, получили %v", err)
		}
	})

	t.Run("критерий удалён из рубрикатора", func(t *testing.T) {
		r := newRubric(t)
		e := grade(t, r, s, "exam", 50)
		if err := r.RemoveCriterion(criterionID(t, "exam")); err != nil {
			t.Fatal(err)
		}
		if _, err := calc.Calculate(s, r, []*domain.Evaluation{e}); !errors.Is(err, domain.ErrCriterionNotInRubric) {
			t.Fatalf("ожидали ErrCriterionNotInRubric, получили %v", err)
		}
	})

	t.Run("пустой рубрикатор", func(t *testing.T) {
		if _, err := calc.Calculate(s, domain.NewRubric(), nil); !errors.Is(err, domain.ErrEmptyRubric) {
			t.Fatalf("ожидали ErrEmptyRubric, получили %v", err)
		}
	})

	t.Run("нет студента", func(t *testing.T) {
		if _, err := calc.Calculate(nil, newRubric(t), nil); !errors.Is(err, domain.ErrStudentRequired) {
			t.Fatalf("ожидали ErrStudentRequired, получили %v", err)
		}
	})
}

type constPolicy float64

func (c constPolicy) Name() string                         { return "const" }
func (c constPolicy) Compute(_ []domain.GradeLine) float64 { return float64(c) }

func TestCalculatorRejectsGradeOutsideScale(t *testing.T) {
	for _, v := range []float64{-1, 10.5, math.NaN()} {
		_, err := domain.NewFinalGradeCalculator(constPolicy(v)).Calculate(newStudent(t, "s1", "Иванов"), newRubric(t), nil)
		if !errors.Is(err, domain.ErrInvalidFinalGrade) {
			t.Errorf("итог %v: ожидали ErrInvalidFinalGrade, получили %v", v, err)
		}
	}
}

func TestRoundedPolicy(t *testing.T) {
	cases := []struct {
		name     string
		value    float64
		decimals int
		want     float64
	}{
		{"половина вверх", 7.5, 0, 8},
		{"меньше половины вниз", 7.49, 0, 7},
		{"погрешность float", 7.4999999999999, 0, 8},
		{"до двух знаков", 6.666666, 2, 6.67},
		{"отрицательная точность = до целого", 3.5, -2, 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := domain.NewRoundedPolicy(constPolicy(tc.value), tc.decimals)
			assertGrade(t, p.Compute(nil), tc.want)
		})
	}
}
