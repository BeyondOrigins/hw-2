package jsonstore_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pos/adapters/jsonstore"
	"pos/domain"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

var now = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestStorePersistsBetweenOpens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "pos.json")
	factory := domain.NewEvaluationFactory(fixedClock{now})

	store := must(jsonstore.Open(path))
	student := must(domain.NewStudent(must(domain.NewStudentID("s1")), must(domain.NewFullName("Иванов"))))
	rubric := domain.NewRubric()
	cid := must(domain.NewCriterionID("c1"))
	err := rubric.AddCriterion(cid, must(domain.NewCriterionName("ДЗ")),
		must(domain.NewMaxScore(10)), must(domain.NewWeight(1)))
	if err != nil {
		t.Fatal(err)
	}
	evaluation := must(factory.Create(rubric, student, cid,
		must(domain.NewScore(7.5)), must(domain.NewComment("ок"))))

	if err := jsonstore.NewStudentRepository(store).Save(student); err != nil {
		t.Fatal(err)
	}
	if err := jsonstore.NewRubricRepository(store).Save(rubric); err != nil {
		t.Fatal(err)
	}
	if err := jsonstore.NewEvaluationRepository(store, factory).Save(evaluation); err != nil {
		t.Fatal(err)
	}

	reopened := must(jsonstore.Open(path))
	students := must(jsonstore.NewStudentRepository(reopened).All())
	if len(students) != 1 || students[0].Name().String() != "Иванов" {
		t.Fatalf("студенты не восстановились: %v", students)
	}
	criteria := must(jsonstore.NewRubricRepository(reopened).Load()).Criteria()
	if len(criteria) != 1 || criteria[0].MaxScore().Value() != 10 {
		t.Fatalf("рубрикатор не восстановился: %v", criteria)
	}
	restored := must(jsonstore.NewEvaluationRepository(reopened, factory).ByID(evaluation.ID()))
	if restored.Score().Value() != 7.5 || restored.Comment().String() != "ок" || !restored.GradedAt().Equal(now) {
		t.Fatalf("оценка не восстановилась: %v %q %v", restored.Score(), restored.Comment(), restored.GradedAt())
	}
}

func TestStoreRejectsCorruptedEvaluation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pos.json")
	data := `{
		"students": [{"id": "s1", "name": "Иванов"}],
		"criteria": [{"id": "c1", "name": "ДЗ", "max_score": 10, "weight": 1}],
		"evaluations": [{"student_id": "s1", "criterion_id": "c1", "score": 12, "graded_at": "2026-11-16T12:00:00Z"}]
	}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	store := must(jsonstore.Open(path))
	repo := jsonstore.NewEvaluationRepository(store, domain.NewEvaluationFactory(fixedClock{now}))

	if _, err := repo.All(); !errors.Is(err, domain.ErrScoreOutOfRange) {
		t.Fatalf("12 из 10 из файла должно отклоняться: %v", err)
	}
}

func TestStoreReturnsNotFound(t *testing.T) {
	store := must(jsonstore.Open(""))
	id := must(domain.NewStudentID("s404"))
	if _, err := jsonstore.NewStudentRepository(store).ByID(id); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ожидали ErrNotFound, получили %v", err)
	}
	if err := jsonstore.NewStudentRepository(store).Delete(id); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ожидали ErrNotFound, получили %v", err)
	}
}
