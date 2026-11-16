package jsonstore

import (
	"errors"
	"fmt"

	"pos/domain"
)

var errCorrupted = errors.New("повреждённые данные в хранилище")

func toStudent(d studentDTO) (*domain.Student, error) {
	id, err := domain.NewStudentID(d.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errCorrupted, err)
	}
	name, err := domain.NewFullName(d.Name)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errCorrupted, err)
	}
	return domain.NewStudent(id, name)
}

func fromStudent(s *domain.Student) studentDTO {
	return studentDTO{ID: s.ID().String(), Name: s.Name().String()}
}

func toRubric(dtos []criterionDTO) (*domain.Rubric, error) {
	rubric := domain.NewRubric()
	for _, d := range dtos {
		if err := addCriterion(rubric, d); err != nil {
			return nil, fmt.Errorf("%w: %w", errCorrupted, err)
		}
	}
	return rubric, nil
}

func addCriterion(r *domain.Rubric, d criterionDTO) error {
	id, err := domain.NewCriterionID(d.ID)
	if err != nil {
		return err
	}
	name, err := domain.NewCriterionName(d.Name)
	if err != nil {
		return err
	}
	maxScore, err := domain.NewMaxScore(d.MaxScore)
	if err != nil {
		return err
	}
	weight, err := domain.NewWeight(d.Weight)
	if err != nil {
		return err
	}
	return r.AddCriterion(id, name, maxScore, weight)
}

func fromRubric(r *domain.Rubric) []criterionDTO {
	criteria := r.Criteria()
	out := make([]criterionDTO, len(criteria))
	for i, c := range criteria {
		out[i] = criterionDTO{
			ID:       c.ID().String(),
			Name:     c.Name().String(),
			MaxScore: c.MaxScore().Value(),
			Weight:   c.Weight().Value(),
		}
	}
	return out
}

func fromEvaluation(e *domain.Evaluation) evaluationDTO {
	return evaluationDTO{
		StudentID:   e.StudentID().String(),
		CriterionID: e.CriterionID().String(),
		Score:       e.Score().Value(),
		Comment:     e.Comment().String(),
		GradedAt:    e.GradedAt(),
	}
}

func toEvaluation(
	f *domain.EvaluationFactory, rubric *domain.Rubric, student *domain.Student, d evaluationDTO,
) (*domain.Evaluation, error) {
	criterionID, err := domain.NewCriterionID(d.CriterionID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errCorrupted, err)
	}
	score, err := domain.NewScore(d.Score)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errCorrupted, err)
	}
	comment, err := domain.NewComment(d.Comment)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errCorrupted, err)
	}
	e, err := f.Restore(rubric, student, criterionID, score, comment, d.GradedAt)
	if err != nil {
		return nil, fmt.Errorf("%w: оценка %s/%s: %w", errCorrupted, d.StudentID, d.CriterionID, err)
	}
	return e, nil
}
