package domain

import (
	"fmt"
	"math"
)

const GradeScale = 10.0

type GradeLine struct {
	Criterion Criterion
	Score     Score
	Graded    bool
}

type FinalGrade struct {
	student StudentID
	value   float64
	policy  string
	lines   []GradeLine
}

func (g FinalGrade) Student() StudentID { return g.student }
func (g FinalGrade) Value() float64     { return g.value }
func (g FinalGrade) Policy() string     { return g.policy }

func (g FinalGrade) Lines() []GradeLine {
	return append([]GradeLine(nil), g.lines...)
}

func (g FinalGrade) GradedCount() int {
	n := 0
	for _, l := range g.lines {
		if l.Graded {
			n++
		}
	}
	return n
}

// GradingPolicy возвращает итог по шкале [0, GradeScale].
type GradingPolicy interface {
	Name() string
	Compute(lines []GradeLine) float64
}

// WeightedAveragePolicy: неоценённый критерий считается за 0.
type WeightedAveragePolicy struct{}

func (WeightedAveragePolicy) Name() string {
	return "взвешенное среднее, нет оценки = 0"
}

func (WeightedAveragePolicy) Compute(lines []GradeLine) float64 {
	return weightedAverage(lines, false)
}

// GradedOnlyPolicy: учитываются только оценённые критерии, веса нормируются по ним.
type GradedOnlyPolicy struct{}

func (GradedOnlyPolicy) Name() string {
	return "взвешенное среднее по оценённым критериям"
}

func (GradedOnlyPolicy) Compute(lines []GradeLine) float64 {
	return weightedAverage(lines, true)
}

func weightedAverage(lines []GradeLine, gradedOnly bool) float64 {
	var sum, totalWeight float64
	for _, l := range lines {
		if gradedOnly && !l.Graded {
			continue
		}
		w := l.Criterion.weight.value
		totalWeight += w
		if l.Graded {
			sum += w * l.Criterion.Normalized(l.Score)
		}
	}
	if totalWeight == 0 {
		return 0
	}
	return GradeScale * sum / totalWeight
}

type RoundedPolicy struct {
	inner    GradingPolicy
	decimals int
}

func NewRoundedPolicy(inner GradingPolicy, decimals int) RoundedPolicy {
	return RoundedPolicy{inner: inner, decimals: max(decimals, 0)}
}

func (p RoundedPolicy) Name() string {
	if p.decimals == 0 {
		return p.inner.Name() + ", округление до целого"
	}
	return fmt.Sprintf("%s, округление до %d знак.", p.inner.Name(), p.decimals)
}

func (p RoundedPolicy) Compute(lines []GradeLine) float64 {
	const eps = 1e-9
	factor := math.Pow(10, float64(p.decimals))
	return math.Floor(p.inner.Compute(lines)*factor+0.5+eps) / factor
}

type FinalGradeCalculator struct {
	policy GradingPolicy
}

func NewFinalGradeCalculator(policy GradingPolicy) *FinalGradeCalculator {
	return &FinalGradeCalculator{policy: policy}
}

func (c *FinalGradeCalculator) Policy() GradingPolicy { return c.policy }

func (c *FinalGradeCalculator) Calculate(student *Student, rubric *Rubric, evaluations []*Evaluation) (FinalGrade, error) {
	if student == nil {
		return FinalGrade{}, ErrStudentRequired
	}
	if rubric == nil || rubric.IsEmpty() {
		return FinalGrade{}, ErrEmptyRubric
	}

	scores := make(map[CriterionID]Score, len(evaluations))
	for _, e := range evaluations {
		if e.StudentID() != student.ID() {
			return FinalGrade{}, fmt.Errorf("%w: %s", ErrForeignEvaluation, e.ID())
		}
		criterion, err := rubric.Criterion(e.CriterionID())
		if err != nil {
			return FinalGrade{}, err
		}
		if err := criterion.CheckScore(e.Score()); err != nil {
			return FinalGrade{}, err
		}
		if _, dup := scores[e.CriterionID()]; dup {
			return FinalGrade{}, fmt.Errorf("%w: %s", ErrDuplicateEvaluation, e.ID())
		}
		scores[e.CriterionID()] = e.Score()
	}

	criteria := rubric.Criteria()
	lines := make([]GradeLine, len(criteria))
	for i, cr := range criteria {
		score, graded := scores[cr.ID()]
		lines[i] = GradeLine{Criterion: cr, Score: score, Graded: graded}
	}

	value := c.policy.Compute(lines)
	if !isFinite(value) || value < 0 || value > GradeScale {
		return FinalGrade{}, fmt.Errorf("%w: %v", ErrInvalidFinalGrade, value)
	}
	return FinalGrade{student: student.ID(), value: value, policy: c.policy.Name(), lines: lines}, nil
}
