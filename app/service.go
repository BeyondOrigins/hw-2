package app

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"pos/domain"
)

var (
	ErrUnknownPolicy     = errors.New("нет такого правила итога")
	ErrUnsupportedFormat = errors.New("неподдерживаемый формат экспорта")
)

type Dependencies struct {
	Students    domain.StudentRepository
	Rubrics     domain.RubricRepository
	Evaluations domain.EvaluationRepository
	Factory     *domain.EvaluationFactory
	Clock       domain.Clock
	Policies    []domain.GradingPolicy
	Exporters   []ResultExporter
}

type Service struct {
	students    domain.StudentRepository
	rubrics     domain.RubricRepository
	evaluations domain.EvaluationRepository
	factory     *domain.EvaluationFactory
	clock       domain.Clock
	policies    []domain.GradingPolicy
	policyIndex int
	calculator  *domain.FinalGradeCalculator
	exporters   map[string]ResultExporter
}

var _ Facade = (*Service)(nil)

func NewService(d Dependencies) (*Service, error) {
	if len(d.Policies) == 0 {
		return nil, errors.New("не задано ни одного правила итога")
	}
	exporters := make(map[string]ResultExporter, len(d.Exporters))
	for _, e := range d.Exporters {
		exporters[strings.ToLower(e.Format())] = e
	}
	return &Service{
		students:    d.Students,
		rubrics:     d.Rubrics,
		evaluations: d.Evaluations,
		factory:     d.Factory,
		clock:       d.Clock,
		policies:    d.Policies,
		calculator:  domain.NewFinalGradeCalculator(d.Policies[0]),
		exporters:   exporters,
	}, nil
}

func (s *Service) ListStudents() ([]StudentView, error) {
	students, err := s.students.All()
	if err != nil {
		return nil, err
	}
	out := make([]StudentView, len(students))
	for i, st := range students {
		out[i] = toStudentView(st)
	}
	return out, nil
}

func (s *Service) AddStudent(name string) (StudentView, error) {
	fullName, err := domain.NewFullName(name)
	if err != nil {
		return StudentView{}, err
	}
	all, err := s.students.All()
	if err != nil {
		return StudentView{}, err
	}
	taken := make(map[string]bool, len(all))
	for _, st := range all {
		taken[st.ID().String()] = true
	}
	id, err := domain.NewStudentID(nextID("s", taken))
	if err != nil {
		return StudentView{}, err
	}
	student, err := domain.NewStudent(id, fullName)
	if err != nil {
		return StudentView{}, err
	}
	if err := s.students.Save(student); err != nil {
		return StudentView{}, err
	}
	return toStudentView(student), nil
}

func (s *Service) RenameStudent(id, name string) error {
	student, err := s.student(id)
	if err != nil {
		return err
	}
	fullName, err := domain.NewFullName(name)
	if err != nil {
		return err
	}
	if err := student.Rename(fullName); err != nil {
		return err
	}
	return s.students.Save(student)
}

func (s *Service) RemoveStudent(id string) error {
	student, err := s.student(id)
	if err != nil {
		return err
	}
	if err := s.evaluations.DeleteByStudent(student.ID()); err != nil {
		return err
	}
	return s.students.Delete(student.ID())
}

func (s *Service) ListCriteria() ([]CriterionView, error) {
	rubric, err := s.rubrics.Load()
	if err != nil {
		return nil, err
	}
	return toCriterionViews(rubric), nil
}

func (s *Service) AddCriterion(name string, maxScore, weight float64) (CriterionView, error) {
	criterionName, err := domain.NewCriterionName(name)
	if err != nil {
		return CriterionView{}, err
	}
	max, err := domain.NewMaxScore(maxScore)
	if err != nil {
		return CriterionView{}, err
	}
	w, err := domain.NewWeight(weight)
	if err != nil {
		return CriterionView{}, err
	}
	rubric, err := s.rubrics.Load()
	if err != nil {
		return CriterionView{}, err
	}
	taken := make(map[string]bool)
	for _, c := range rubric.Criteria() {
		taken[c.ID().String()] = true
	}
	id, err := domain.NewCriterionID(nextID("c", taken))
	if err != nil {
		return CriterionView{}, err
	}
	if err := rubric.AddCriterion(id, criterionName, max, w); err != nil {
		return CriterionView{}, err
	}
	if err := s.rubrics.Save(rubric); err != nil {
		return CriterionView{}, err
	}
	created, err := rubric.Criterion(id)
	if err != nil {
		return CriterionView{}, err
	}
	return toCriterionView(created), nil
}

func (s *Service) RenameCriterion(id, name string) error {
	criterionID, err := domain.NewCriterionID(id)
	if err != nil {
		return err
	}
	criterionName, err := domain.NewCriterionName(name)
	if err != nil {
		return err
	}
	return s.updateRubric(func(r *domain.Rubric) error { return r.RenameCriterion(criterionID, criterionName) })
}

func (s *Service) ChangeCriterionWeight(id string, weight float64) error {
	criterionID, err := domain.NewCriterionID(id)
	if err != nil {
		return err
	}
	w, err := domain.NewWeight(weight)
	if err != nil {
		return err
	}
	return s.updateRubric(func(r *domain.Rubric) error { return r.ChangeWeight(criterionID, w) })
}

func (s *Service) RemoveCriterion(id string) error {
	criterionID, err := domain.NewCriterionID(id)
	if err != nil {
		return err
	}
	rubric, err := s.rubrics.Load()
	if err != nil {
		return err
	}
	if err := rubric.RemoveCriterion(criterionID); err != nil {
		return err
	}
	// Сначала снимаем оценки: иначе в хранилище остались бы оценки по несуществующему критерию.
	if err := s.evaluations.DeleteByCriterion(criterionID); err != nil {
		return err
	}
	return s.rubrics.Save(rubric)
}

func (s *Service) Grade(studentID, criterionID string, score float64, comment string) (EvaluationView, error) {
	student, err := s.student(studentID)
	if err != nil {
		return EvaluationView{}, err
	}
	cid, sc, cm, err := parseEvaluationInput(criterionID, score, comment)
	if err != nil {
		return EvaluationView{}, err
	}
	rubric, err := s.rubrics.Load()
	if err != nil {
		return EvaluationView{}, err
	}
	evaluation, err := s.factory.Create(rubric, student, cid, sc, cm)
	if err != nil {
		return EvaluationView{}, err
	}
	switch _, err := s.evaluations.ByID(evaluation.ID()); {
	case err == nil:
		return EvaluationView{}, fmt.Errorf("%w — её можно исправить или отменить", domain.ErrDuplicateEvaluation)
	case !errors.Is(err, domain.ErrNotFound):
		return EvaluationView{}, err
	}
	if err := s.evaluations.Save(evaluation); err != nil {
		return EvaluationView{}, err
	}
	return s.evaluationView(evaluation, student, rubric)
}

func (s *Service) CorrectEvaluation(studentID, criterionID string, score float64, comment string) (EvaluationView, error) {
	student, err := s.student(studentID)
	if err != nil {
		return EvaluationView{}, err
	}
	cid, sc, cm, err := parseEvaluationInput(criterionID, score, comment)
	if err != nil {
		return EvaluationView{}, err
	}
	id, err := domain.NewEvaluationID(student.ID(), cid)
	if err != nil {
		return EvaluationView{}, err
	}
	evaluation, err := s.evaluations.ByID(id)
	if err != nil {
		return EvaluationView{}, err
	}
	rubric, err := s.rubrics.Load()
	if err != nil {
		return EvaluationView{}, err
	}
	if err := evaluation.Correct(rubric, sc, cm, s.clock.Now()); err != nil {
		return EvaluationView{}, err
	}
	if err := s.evaluations.Save(evaluation); err != nil {
		return EvaluationView{}, err
	}
	return s.evaluationView(evaluation, student, rubric)
}

func (s *Service) CancelEvaluation(studentID, criterionID string) error {
	sid, err := domain.NewStudentID(studentID)
	if err != nil {
		return err
	}
	cid, err := domain.NewCriterionID(criterionID)
	if err != nil {
		return err
	}
	id, err := domain.NewEvaluationID(sid, cid)
	if err != nil {
		return err
	}
	return s.evaluations.Delete(id)
}

// ListEvaluations с пустым studentID возвращает оценки всех студентов.
func (s *Service) ListEvaluations(studentID string) ([]EvaluationView, error) {
	var (
		evaluations []*domain.Evaluation
		students    []*domain.Student
	)
	if strings.TrimSpace(studentID) == "" {
		var err error
		if evaluations, err = s.evaluations.All(); err != nil {
			return nil, err
		}
		if students, err = s.students.All(); err != nil {
			return nil, err
		}
	} else {
		student, err := s.student(studentID)
		if err != nil {
			return nil, err
		}
		if evaluations, err = s.evaluations.ByStudent(student.ID()); err != nil {
			return nil, err
		}
		students = []*domain.Student{student}
	}
	rubric, err := s.rubrics.Load()
	if err != nil {
		return nil, err
	}

	studentOrder := make(map[domain.StudentID]int, len(students))
	studentViews := make(map[domain.StudentID]StudentView, len(students))
	for i, st := range students {
		studentOrder[st.ID()] = i
		studentViews[st.ID()] = toStudentView(st)
	}
	criterionOrder := make(map[domain.CriterionID]int)
	for i, c := range rubric.Criteria() {
		criterionOrder[c.ID()] = i
	}

	slices.SortStableFunc(evaluations, func(a, b *domain.Evaluation) int {
		if d := studentOrder[a.StudentID()] - studentOrder[b.StudentID()]; d != 0 {
			return d
		}
		return criterionOrder[a.CriterionID()] - criterionOrder[b.CriterionID()]
	})

	out := make([]EvaluationView, 0, len(evaluations))
	for _, e := range evaluations {
		criterion, err := rubric.Criterion(e.CriterionID())
		if err != nil {
			return nil, err
		}
		out = append(out, toEvaluationView(e, studentViews[e.StudentID()], criterion))
	}
	return out, nil
}

func (s *Service) FinalGrade(studentID string) (FinalGradeView, error) {
	student, err := s.student(studentID)
	if err != nil {
		return FinalGradeView{}, err
	}
	rubric, err := s.rubrics.Load()
	if err != nil {
		return FinalGradeView{}, err
	}
	return s.finalGrade(student, rubric)
}

func (s *Service) FinalGrades() ([]FinalGradeView, error) {
	_, grades, err := s.results()
	return grades, err
}

func (s *Service) CriterionStats() ([]CriterionStatsView, error) {
	rubric, err := s.rubrics.Load()
	if err != nil {
		return nil, err
	}
	students, err := s.students.All()
	if err != nil {
		return nil, err
	}
	evaluations, err := s.evaluations.All()
	if err != nil {
		return nil, err
	}
	byCriterion := make(map[domain.CriterionID][]float64)
	for _, e := range evaluations {
		byCriterion[e.CriterionID()] = append(byCriterion[e.CriterionID()], e.Score().Value())
	}

	criteria := rubric.Criteria()
	out := make([]CriterionStatsView, len(criteria))
	for i, c := range criteria {
		scores := byCriterion[c.ID()]
		stats := CriterionStatsView{Criterion: toCriterionView(c), Graded: len(scores), Students: len(students)}
		if len(scores) > 0 {
			stats.Min, stats.Max = math.Inf(1), math.Inf(-1)
			var sum float64
			for _, v := range scores {
				sum += v
				stats.Min = min(stats.Min, v)
				stats.Max = max(stats.Max, v)
			}
			stats.Average = sum / float64(len(scores))
		}
		out[i] = stats
	}
	return out, nil
}

func (s *Service) Policies() []string {
	names := make([]string, len(s.policies))
	for i, p := range s.policies {
		names[i] = p.Name()
	}
	return names
}

func (s *Service) CurrentPolicy() int { return s.policyIndex }

func (s *Service) SelectPolicy(index int) error {
	if index < 0 || index >= len(s.policies) {
		return fmt.Errorf("%w: %d", ErrUnknownPolicy, index+1)
	}
	s.policyIndex = index
	s.calculator = domain.NewFinalGradeCalculator(s.policies[index])
	return nil
}

func (s *Service) ExportFormats() []string {
	formats := make([]string, 0, len(s.exporters))
	for f := range s.exporters {
		formats = append(formats, f)
	}
	slices.Sort(formats)
	return formats
}

// ExportResults выбирает формат по расширению файла и возвращает число выгруженных студентов.
func (s *Service) ExportResults(path string) (int, error) {
	format := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	exporter, ok := s.exporters[format]
	if !ok {
		return 0, fmt.Errorf("%w: «%s», доступны: %s", ErrUnsupportedFormat, format, strings.Join(s.ExportFormats(), ", "))
	}
	rubric, grades, err := s.results()
	if err != nil {
		return 0, err
	}
	report := ResultReport{
		Policy:      s.calculator.Policy().Name(),
		GeneratedAt: s.clock.Now(),
		Scale:       domain.GradeScale,
		Criteria:    toCriterionViews(rubric),
		Grades:      grades,
	}
	if err := exporter.Export(path, report); err != nil {
		return 0, err
	}
	return len(grades), nil
}

func (s *Service) results() (*domain.Rubric, []FinalGradeView, error) {
	rubric, err := s.rubrics.Load()
	if err != nil {
		return nil, nil, err
	}
	students, err := s.students.All()
	if err != nil {
		return nil, nil, err
	}
	grades := make([]FinalGradeView, 0, len(students))
	for _, st := range students {
		g, err := s.finalGrade(st, rubric)
		if err != nil {
			return nil, nil, err
		}
		grades = append(grades, g)
	}
	return rubric, grades, nil
}

func (s *Service) finalGrade(student *domain.Student, rubric *domain.Rubric) (FinalGradeView, error) {
	evaluations, err := s.evaluations.ByStudent(student.ID())
	if err != nil {
		return FinalGradeView{}, err
	}
	grade, err := s.calculator.Calculate(student, rubric, evaluations)
	if err != nil {
		return FinalGradeView{}, err
	}
	return toFinalGradeView(student, grade), nil
}

func (s *Service) student(raw string) (*domain.Student, error) {
	id, err := domain.NewStudentID(raw)
	if err != nil {
		return nil, err
	}
	return s.students.ByID(id)
}

func (s *Service) updateRubric(change func(*domain.Rubric) error) error {
	rubric, err := s.rubrics.Load()
	if err != nil {
		return err
	}
	if err := change(rubric); err != nil {
		return err
	}
	return s.rubrics.Save(rubric)
}

func (s *Service) evaluationView(e *domain.Evaluation, student *domain.Student, rubric *domain.Rubric) (EvaluationView, error) {
	criterion, err := rubric.Criterion(e.CriterionID())
	if err != nil {
		return EvaluationView{}, err
	}
	return toEvaluationView(e, toStudentView(student), criterion), nil
}

func parseEvaluationInput(criterionID string, score float64, comment string) (domain.CriterionID, domain.Score, domain.Comment, error) {
	cid, err := domain.NewCriterionID(criterionID)
	if err != nil {
		return domain.CriterionID{}, domain.Score{}, domain.Comment{}, err
	}
	sc, err := domain.NewScore(score)
	if err != nil {
		return domain.CriterionID{}, domain.Score{}, domain.Comment{}, err
	}
	cm, err := domain.NewComment(comment)
	if err != nil {
		return domain.CriterionID{}, domain.Score{}, domain.Comment{}, err
	}
	return cid, sc, cm, nil
}

func nextID(prefix string, taken map[string]bool) string {
	for n := len(taken) + 1; ; n++ {
		if id := prefix + strconv.Itoa(n); !taken[id] {
			return id
		}
	}
}
