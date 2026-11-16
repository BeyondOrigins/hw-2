package app

import (
	"time"

	"pos/domain"
)

type StudentView struct {
	ID   string
	Name string
}

type CriterionView struct {
	ID       string
	Name     string
	MaxScore float64
	Weight   float64
}

type EvaluationView struct {
	StudentID     string
	StudentName   string
	CriterionID   string
	CriterionName string
	Score         float64
	MaxScore      float64
	Comment       string
	GradedAt      time.Time
}

type GradeLineView struct {
	Criterion CriterionView
	Score     float64
	Graded    bool
}

type FinalGradeView struct {
	StudentID   string
	StudentName string
	Value       float64
	Scale       float64
	Policy      string
	Graded      int
	Lines       []GradeLineView
}

type CriterionStatsView struct {
	Criterion CriterionView
	Graded    int
	Students  int
	Average   float64
	Min       float64
	Max       float64
}

type ResultReport struct {
	Policy      string
	GeneratedAt time.Time
	Scale       float64
	Criteria    []CriterionView
	Grades      []FinalGradeView
}

func toStudentView(s *domain.Student) StudentView {
	return StudentView{ID: s.ID().String(), Name: s.Name().String()}
}

func toCriterionView(c domain.Criterion) CriterionView {
	return CriterionView{
		ID:       c.ID().String(),
		Name:     c.Name().String(),
		MaxScore: c.MaxScore().Value(),
		Weight:   c.Weight().Value(),
	}
}

func toCriterionViews(r *domain.Rubric) []CriterionView {
	criteria := r.Criteria()
	out := make([]CriterionView, len(criteria))
	for i, c := range criteria {
		out[i] = toCriterionView(c)
	}
	return out
}

func toEvaluationView(e *domain.Evaluation, student StudentView, criterion domain.Criterion) EvaluationView {
	return EvaluationView{
		StudentID:     student.ID,
		StudentName:   student.Name,
		CriterionID:   criterion.ID().String(),
		CriterionName: criterion.Name().String(),
		Score:         e.Score().Value(),
		MaxScore:      criterion.MaxScore().Value(),
		Comment:       e.Comment().String(),
		GradedAt:      e.GradedAt(),
	}
}

func toFinalGradeView(s *domain.Student, g domain.FinalGrade) FinalGradeView {
	lines := g.Lines()
	views := make([]GradeLineView, len(lines))
	for i, l := range lines {
		views[i] = GradeLineView{Criterion: toCriterionView(l.Criterion), Score: l.Score.Value(), Graded: l.Graded}
	}
	return FinalGradeView{
		StudentID:   s.ID().String(),
		StudentName: s.Name().String(),
		Value:       g.Value(),
		Scale:       domain.GradeScale,
		Policy:      g.Policy(),
		Graded:      g.GradedCount(),
		Lines:       views,
	}
}
