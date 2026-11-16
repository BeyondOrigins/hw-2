package app

// Facade — входной порт: всё, что умеет модуль «ПОС», в одной точке для внешних адаптеров.
type Facade interface {
	ListStudents() ([]StudentView, error)
	AddStudent(name string) (StudentView, error)
	RenameStudent(id, name string) error
	RemoveStudent(id string) error

	ListCriteria() ([]CriterionView, error)
	AddCriterion(name string, maxScore, weight float64) (CriterionView, error)
	RenameCriterion(id, name string) error
	ChangeCriterionWeight(id string, weight float64) error
	RemoveCriterion(id string) error

	Grade(studentID, criterionID string, score float64, comment string) (EvaluationView, error)
	CorrectEvaluation(studentID, criterionID string, score float64, comment string) (EvaluationView, error)
	CancelEvaluation(studentID, criterionID string) error
	ListEvaluations(studentID string) ([]EvaluationView, error)

	FinalGrade(studentID string) (FinalGradeView, error)
	FinalGrades() ([]FinalGradeView, error)
	CriterionStats() ([]CriterionStatsView, error)

	Policies() []string
	CurrentPolicy() int
	SelectPolicy(index int) error

	ExportFormats() []string
	ExportResults(path string) (int, error)
}

// ResultExporter — выходной порт выгрузки итогов в конкретный формат.
type ResultExporter interface {
	Format() string
	Export(path string, report ResultReport) error
}
