package domain

type StudentRepository interface {
	Save(student *Student) error
	ByID(id StudentID) (*Student, error)
	All() ([]*Student, error)
	Delete(id StudentID) error
}

type RubricRepository interface {
	Load() (*Rubric, error)
	Save(rubric *Rubric) error
}

type EvaluationRepository interface {
	Save(evaluation *Evaluation) error
	ByID(id EvaluationID) (*Evaluation, error)
	ByStudent(id StudentID) ([]*Evaluation, error)
	All() ([]*Evaluation, error)
	Delete(id EvaluationID) error
	DeleteByStudent(id StudentID) error
	DeleteByCriterion(id CriterionID) error
}
