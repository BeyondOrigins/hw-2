package domain

import "time"

type Evaluation struct {
	id       EvaluationID
	score    Score
	comment  Comment
	gradedAt time.Time
}

func (e *Evaluation) ID() EvaluationID         { return e.id }
func (e *Evaluation) StudentID() StudentID     { return e.id.student }
func (e *Evaluation) CriterionID() CriterionID { return e.id.criterion }
func (e *Evaluation) Score() Score             { return e.score }
func (e *Evaluation) Comment() Comment         { return e.comment }
func (e *Evaluation) GradedAt() time.Time      { return e.gradedAt }

func (e *Evaluation) Correct(rubric *Rubric, score Score, comment Comment, at time.Time) error {
	if rubric == nil {
		return ErrEmptyRubric
	}
	criterion, err := rubric.Criterion(e.id.criterion)
	if err != nil {
		return err
	}
	if err := criterion.CheckScore(score); err != nil {
		return err
	}
	e.score = score
	e.comment = comment
	e.gradedAt = at
	return nil
}

type Clock interface {
	Now() time.Time
}

type EvaluationFactory struct {
	clock Clock
}

func NewEvaluationFactory(clock Clock) *EvaluationFactory {
	return &EvaluationFactory{clock: clock}
}

func (f *EvaluationFactory) Create(
	rubric *Rubric, student *Student, criterionID CriterionID, score Score, comment Comment,
) (*Evaluation, error) {
	return f.build(rubric, student, criterionID, score, comment, f.clock.Now())
}

// повторная проверка инвариантов при загрузке из хранилища
func (f *EvaluationFactory) Restore(
	rubric *Rubric, student *Student, criterionID CriterionID, score Score, comment Comment, gradedAt time.Time,
) (*Evaluation, error) {
	return f.build(rubric, student, criterionID, score, comment, gradedAt)
}

func (f *EvaluationFactory) build(
	rubric *Rubric, student *Student, criterionID CriterionID, score Score, comment Comment, at time.Time,
) (*Evaluation, error) {
	if student == nil {
		return nil, ErrStudentRequired
	}
	if rubric == nil || rubric.IsEmpty() {
		return nil, ErrEmptyRubric
	}
	criterion, err := rubric.Criterion(criterionID)
	if err != nil {
		return nil, err
	}
	if err := criterion.CheckScore(score); err != nil {
		return nil, err
	}
	id, err := NewEvaluationID(student.ID(), criterionID)
	if err != nil {
		return nil, err
	}
	return &Evaluation{id: id, score: score, comment: comment, gradedAt: at}, nil
}
