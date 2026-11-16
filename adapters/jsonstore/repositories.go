package jsonstore

import (
	"fmt"
	"slices"

	"pos/domain"
)

type StudentRepository struct{ store *Store }

func NewStudentRepository(store *Store) *StudentRepository {
	return &StudentRepository{store: store}
}

func (r *StudentRepository) Save(student *domain.Student) error {
	dto := fromStudent(student)
	return r.store.write(func(doc *document) error {
		if i := indexStudent(doc.Students, dto.ID); i >= 0 {
			doc.Students[i] = dto
		} else {
			doc.Students = append(doc.Students, dto)
		}
		return nil
	})
}

func (r *StudentRepository) ByID(id domain.StudentID) (*domain.Student, error) {
	var student *domain.Student
	err := r.store.read(func(doc *document) error {
		i := indexStudent(doc.Students, id.String())
		if i < 0 {
			return studentNotFound(id.String())
		}
		var err error
		student, err = toStudent(doc.Students[i])
		return err
	})
	return student, err
}

func (r *StudentRepository) All() ([]*domain.Student, error) {
	var students []*domain.Student
	err := r.store.read(func(doc *document) error {
		students = make([]*domain.Student, 0, len(doc.Students))
		for _, d := range doc.Students {
			s, err := toStudent(d)
			if err != nil {
				return err
			}
			students = append(students, s)
		}
		return nil
	})
	return students, err
}

func (r *StudentRepository) Delete(id domain.StudentID) error {
	return r.store.write(func(doc *document) error {
		i := indexStudent(doc.Students, id.String())
		if i < 0 {
			return studentNotFound(id.String())
		}
		doc.Students = slices.Delete(doc.Students, i, i+1)
		return nil
	})
}

type RubricRepository struct{ store *Store }

func NewRubricRepository(store *Store) *RubricRepository {
	return &RubricRepository{store: store}
}

func (r *RubricRepository) Load() (*domain.Rubric, error) {
	var rubric *domain.Rubric
	err := r.store.read(func(doc *document) error {
		var err error
		rubric, err = toRubric(doc.Criteria)
		return err
	})
	return rubric, err
}

func (r *RubricRepository) Save(rubric *domain.Rubric) error {
	dtos := fromRubric(rubric)
	return r.store.write(func(doc *document) error {
		doc.Criteria = dtos
		return nil
	})
}

type EvaluationRepository struct {
	store   *Store
	factory *domain.EvaluationFactory
}

func NewEvaluationRepository(store *Store, factory *domain.EvaluationFactory) *EvaluationRepository {
	return &EvaluationRepository{store: store, factory: factory}
}

func (r *EvaluationRepository) Save(evaluation *domain.Evaluation) error {
	dto := fromEvaluation(evaluation)
	return r.store.write(func(doc *document) error {
		if i := indexEvaluation(doc.Evaluations, dto.StudentID, dto.CriterionID); i >= 0 {
			doc.Evaluations[i] = dto
		} else {
			doc.Evaluations = append(doc.Evaluations, dto)
		}
		return nil
	})
}

func (r *EvaluationRepository) ByID(id domain.EvaluationID) (*domain.Evaluation, error) {
	var evaluation *domain.Evaluation
	err := r.store.read(func(doc *document) error {
		i := indexEvaluation(doc.Evaluations, id.Student().String(), id.Criterion().String())
		if i < 0 {
			return evaluationNotFound(id)
		}
		restored, err := r.restore(doc, doc.Evaluations[i:i+1])
		if err != nil {
			return err
		}
		evaluation = restored[0]
		return nil
	})
	return evaluation, err
}

func (r *EvaluationRepository) ByStudent(id domain.StudentID) ([]*domain.Evaluation, error) {
	return r.filter(func(d evaluationDTO) bool { return d.StudentID == id.String() })
}

func (r *EvaluationRepository) All() ([]*domain.Evaluation, error) {
	return r.filter(func(evaluationDTO) bool { return true })
}

func (r *EvaluationRepository) Delete(id domain.EvaluationID) error {
	return r.store.write(func(doc *document) error {
		i := indexEvaluation(doc.Evaluations, id.Student().String(), id.Criterion().String())
		if i < 0 {
			return evaluationNotFound(id)
		}
		doc.Evaluations = slices.Delete(doc.Evaluations, i, i+1)
		return nil
	})
}

func (r *EvaluationRepository) DeleteByStudent(id domain.StudentID) error {
	return r.deleteWhere(func(d evaluationDTO) bool { return d.StudentID == id.String() })
}

func (r *EvaluationRepository) DeleteByCriterion(id domain.CriterionID) error {
	return r.deleteWhere(func(d evaluationDTO) bool { return d.CriterionID == id.String() })
}

func (r *EvaluationRepository) deleteWhere(match func(evaluationDTO) bool) error {
	return r.store.write(func(doc *document) error {
		doc.Evaluations = slices.DeleteFunc(doc.Evaluations, match)
		return nil
	})
}

func (r *EvaluationRepository) filter(match func(evaluationDTO) bool) ([]*domain.Evaluation, error) {
	var evaluations []*domain.Evaluation
	err := r.store.read(func(doc *document) error {
		var selected []evaluationDTO
		for _, d := range doc.Evaluations {
			if match(d) {
				selected = append(selected, d)
			}
		}
		var err error
		evaluations, err = r.restore(doc, selected)
		return err
	})
	return evaluations, err
}

func (r *EvaluationRepository) restore(doc *document, dtos []evaluationDTO) ([]*domain.Evaluation, error) {
	if len(dtos) == 0 {
		return nil, nil
	}
	rubric, err := toRubric(doc.Criteria)
	if err != nil {
		return nil, err
	}
	students := make(map[string]*domain.Student)
	out := make([]*domain.Evaluation, 0, len(dtos))
	for _, d := range dtos {
		student, ok := students[d.StudentID]
		if !ok {
			i := indexStudent(doc.Students, d.StudentID)
			if i < 0 {
				return nil, fmt.Errorf("%w: оценка ссылается на %w", errCorrupted, studentNotFound(d.StudentID))
			}
			if student, err = toStudent(doc.Students[i]); err != nil {
				return nil, err
			}
			students[d.StudentID] = student
		}
		e, err := toEvaluation(r.factory, rubric, student, d)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func indexStudent(students []studentDTO, id string) int {
	return slices.IndexFunc(students, func(d studentDTO) bool { return d.ID == id })
}

func indexEvaluation(evaluations []evaluationDTO, studentID, criterionID string) int {
	return slices.IndexFunc(evaluations, func(d evaluationDTO) bool {
		return d.StudentID == studentID && d.CriterionID == criterionID
	})
}

func studentNotFound(id string) error {
	return fmt.Errorf("студент %s: %w", id, domain.ErrNotFound)
}

func evaluationNotFound(id domain.EvaluationID) error {
	return fmt.Errorf("оценка студента %s по критерию %s: %w", id.Student(), id.Criterion(), domain.ErrNotFound)
}
