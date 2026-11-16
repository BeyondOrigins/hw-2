package app_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"pos/adapters/jsonstore"
	"pos/app"
	"pos/domain"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

type recordingExporter struct {
	path   string
	report app.ResultReport
}

func (e *recordingExporter) Format() string { return "csv" }

func (e *recordingExporter) Export(path string, report app.ResultReport) error {
	e.path, e.report = path, report
	return nil
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func check(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
}

func newService(t *testing.T, exporters ...app.ResultExporter) *app.Service {
	t.Helper()
	store := must(jsonstore.Open(""))
	clock := fixedClock{time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)}
	factory := domain.NewEvaluationFactory(clock)
	return must(app.NewService(app.Dependencies{
		Students:    jsonstore.NewStudentRepository(store),
		Rubrics:     jsonstore.NewRubricRepository(store),
		Evaluations: jsonstore.NewEvaluationRepository(store, factory),
		Factory:     factory,
		Clock:       clock,
		Policies: []domain.GradingPolicy{
			domain.WeightedAveragePolicy{},
			domain.GradedOnlyPolicy{},
		},
		Exporters: exporters,
	}))
}

// Студент s1, критерии c1 «ДЗ» (max 10, вес 0.6) и c2 «Экзамен» (max 100, вес 0.4).
func seed(t *testing.T, s *app.Service) {
	t.Helper()
	must(s.AddStudent("Иванов Иван"))
	must(s.AddCriterion("ДЗ", 10, 0.6))
	must(s.AddCriterion("Экзамен", 100, 0.4))
}

func TestNewServiceRequiresPolicy(t *testing.T) {
	if _, err := app.NewService(app.Dependencies{}); err == nil {
		t.Fatal("сервис без правил итога создаваться не должен")
	}
}

func TestAddStudentsAndCriteriaAssignsIDs(t *testing.T) {
	s := newService(t)
	seed(t, s)
	must(s.AddStudent("Петрова Анна"))

	students := must(s.ListStudents())
	if len(students) != 2 || students[0].ID != "s1" || students[1].ID != "s2" {
		t.Fatalf("неверный список студентов: %+v", students)
	}
	criteria := must(s.ListCriteria())
	if len(criteria) != 2 || criteria[0].ID != "c1" || criteria[1].Name != "Экзамен" {
		t.Fatalf("неверный список критериев: %+v", criteria)
	}
}

func TestAddRejectsInvalidInput(t *testing.T) {
	s := newService(t)
	if _, err := s.AddStudent("  "); !errors.Is(err, domain.ErrEmptyName) {
		t.Fatalf("ожидали ErrEmptyName, получили %v", err)
	}
	if _, err := s.AddCriterion("ДЗ", 0, 1); !errors.Is(err, domain.ErrInvalidMaxScore) {
		t.Fatalf("ожидали ErrInvalidMaxScore, получили %v", err)
	}
	if _, err := s.AddCriterion("ДЗ", 10, -1); !errors.Is(err, domain.ErrInvalidWeight) {
		t.Fatalf("ожидали ErrInvalidWeight, получили %v", err)
	}
	must(s.AddCriterion("ДЗ", 10, 1))
	if _, err := s.AddCriterion("дз", 5, 1); !errors.Is(err, domain.ErrDuplicateCriterion) {
		t.Fatalf("ожидали ErrDuplicateCriterion, получили %v", err)
	}
}

func TestGradeEnforcesDomainRules(t *testing.T) {
	s := newService(t)
	seed(t, s)

	cases := []struct {
		name               string
		student, criterion string
		score              float64
		want               error
	}{
		{"12 из 10", "s1", "c1", 12, domain.ErrScoreOutOfRange},
		{"отрицательный балл", "s1", "c1", -1, domain.ErrInvalidScore},
		{"критерия нет в рубрикаторе", "s1", "c99", 5, domain.ErrCriterionNotInRubric},
		{"нет такого студента", "s42", "c1", 5, domain.ErrNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := s.Grade(tc.student, tc.criterion, tc.score, ""); !errors.Is(err, tc.want) {
				t.Fatalf("ожидали %v, получили %v", tc.want, err)
			}
		})
	}
	if evaluations := must(s.ListEvaluations("")); len(evaluations) != 0 {
		t.Fatalf("отклонённые оценки не должны сохраняться: %+v", evaluations)
	}
}

func TestGradeTwiceIsRejected(t *testing.T) {
	s := newService(t)
	seed(t, s)
	must(s.Grade("s1", "c1", 7, ""))

	if _, err := s.Grade("s1", "c1", 9, ""); !errors.Is(err, domain.ErrDuplicateEvaluation) {
		t.Fatalf("ожидали ErrDuplicateEvaluation, получили %v", err)
	}
	evaluations := must(s.ListEvaluations("s1"))
	if len(evaluations) != 1 || evaluations[0].Score != 7 {
		t.Fatalf("первая оценка не должна перезаписываться: %+v", evaluations)
	}
}

func TestCorrectAndCancelEvaluation(t *testing.T) {
	s := newService(t)
	seed(t, s)
	must(s.Grade("s1", "c1", 7, ""))

	if _, err := s.CorrectEvaluation("s1", "c1", 11, ""); !errors.Is(err, domain.ErrScoreOutOfRange) {
		t.Fatalf("ожидали ErrScoreOutOfRange, получили %v", err)
	}
	corrected := must(s.CorrectEvaluation("s1", "c1", 9.5, "апелляция"))
	if corrected.Score != 9.5 || corrected.Comment != "апелляция" {
		t.Fatalf("исправление не применилось: %+v", corrected)
	}
	if _, err := s.CorrectEvaluation("s1", "c2", 50, ""); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("исправить можно только выставленную оценку: %v", err)
	}

	check(t, s.CancelEvaluation("s1", "c1"))
	if evaluations := must(s.ListEvaluations("s1")); len(evaluations) != 0 {
		t.Fatalf("оценка не отменена: %+v", evaluations)
	}
	if err := s.CancelEvaluation("s1", "c1"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("повторная отмена: ожидали ErrNotFound, получили %v", err)
	}
}

func TestFinalGradeAndPolicySwitch(t *testing.T) {
	s := newService(t)
	seed(t, s)
	must(s.Grade("s1", "c1", 8, ""))

	g := must(s.FinalGrade("s1"))
	if math.Abs(g.Value-4.8) > 1e-9 || g.Graded != 1 || len(g.Lines) != 2 {
		t.Fatalf("неверный итог: %+v", g)
	}

	check(t, s.SelectPolicy(1))
	if s.CurrentPolicy() != 1 {
		t.Fatalf("правило не переключилось: %d", s.CurrentPolicy())
	}
	if g := must(s.FinalGrade("s1")); math.Abs(g.Value-8) > 1e-9 {
		t.Fatalf("итог по оценённым: ожидали 8, получили %v", g.Value)
	}
	if err := s.SelectPolicy(5); !errors.Is(err, app.ErrUnknownPolicy) {
		t.Fatalf("ожидали ErrUnknownPolicy, получили %v", err)
	}
}

func TestRemoveCriterionCascadesEvaluations(t *testing.T) {
	s := newService(t)
	seed(t, s)
	must(s.Grade("s1", "c1", 8, ""))
	must(s.Grade("s1", "c2", 50, ""))

	check(t, s.RemoveCriterion("c2"))

	evaluations := must(s.ListEvaluations(""))
	if len(evaluations) != 1 || evaluations[0].CriterionID != "c1" {
		t.Fatalf("оценки по удалённому критерию должны быть сняты: %+v", evaluations)
	}
	if g := must(s.FinalGrade("s1")); math.Abs(g.Value-8) > 1e-9 {
		t.Fatalf("итог должен считаться по оставшимся критериям: %v", g.Value)
	}
}

func TestRemoveStudentCascadesEvaluations(t *testing.T) {
	s := newService(t)
	seed(t, s)
	must(s.AddStudent("Петрова Анна"))
	must(s.Grade("s1", "c1", 8, ""))
	must(s.Grade("s2", "c1", 9, ""))

	check(t, s.RemoveStudent("s1"))

	evaluations := must(s.ListEvaluations(""))
	if len(evaluations) != 1 || evaluations[0].StudentID != "s2" {
		t.Fatalf("оценки удалённого студента должны быть сняты: %+v", evaluations)
	}
	if _, err := s.FinalGrade("s1"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ожидали ErrNotFound, получили %v", err)
	}
}

func TestEditCriterionAffectsFinalGrade(t *testing.T) {
	s := newService(t)
	seed(t, s)
	must(s.Grade("s1", "c1", 10, ""))

	check(t, s.RenameCriterion("c1", "Домашние задания"))
	check(t, s.ChangeCriterionWeight("c1", 0.4))
	if err := s.ChangeCriterionWeight("c1", 0); !errors.Is(err, domain.ErrInvalidWeight) {
		t.Fatalf("ожидали ErrInvalidWeight, получили %v", err)
	}

	g := must(s.FinalGrade("s1"))
	if math.Abs(g.Value-5) > 1e-9 || g.Lines[0].Criterion.Name != "Домашние задания" {
		t.Fatalf("итог должен учитывать новые вес и название: %+v", g)
	}
}

func TestCriterionStats(t *testing.T) {
	s := newService(t)
	seed(t, s)
	must(s.AddStudent("Петрова Анна"))
	must(s.Grade("s1", "c1", 6, ""))
	must(s.Grade("s2", "c1", 9, ""))

	stats := must(s.CriterionStats())
	hw, exam := stats[0], stats[1]
	if hw.Graded != 2 || hw.Students != 2 || hw.Average != 7.5 || hw.Min != 6 || hw.Max != 9 {
		t.Fatalf("неверная статистика ДЗ: %+v", hw)
	}
	if exam.Graded != 0 {
		t.Fatalf("экзамен никто не сдавал: %+v", exam)
	}
}

func TestExportResults(t *testing.T) {
	exporter := &recordingExporter{}
	s := newService(t, exporter)
	seed(t, s)
	must(s.Grade("s1", "c1", 10, ""))

	n := must(s.ExportResults("out/results.CSV"))
	if n != 1 || exporter.path != "out/results.CSV" {
		t.Fatalf("экспорт не вызван: n=%d path=%q", n, exporter.path)
	}
	r := exporter.report
	if len(r.Grades) != 1 || len(r.Criteria) != 2 || r.Policy == "" || r.Scale != domain.GradeScale {
		t.Fatalf("неполный отчёт: %+v", r)
	}
	if _, err := s.ExportResults("results.txt"); !errors.Is(err, app.ErrUnsupportedFormat) {
		t.Fatalf("ожидали ErrUnsupportedFormat, получили %v", err)
	}
}
