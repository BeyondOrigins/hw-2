package domain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maxNameLength    = 200
	maxCommentLength = 500
)

type StudentID struct{ value string }

func NewStudentID(raw string) (StudentID, error) {
	v, err := requireID(raw)
	if err != nil {
		return StudentID{}, fmt.Errorf("студент: %w", err)
	}
	return StudentID{value: v}, nil
}

func (id StudentID) String() string { return id.value }
func (id StudentID) IsZero() bool   { return id.value == "" }

type CriterionID struct{ value string }

func NewCriterionID(raw string) (CriterionID, error) {
	v, err := requireID(raw)
	if err != nil {
		return CriterionID{}, fmt.Errorf("критерий: %w", err)
	}
	return CriterionID{value: v}, nil
}

func (id CriterionID) String() string { return id.value }
func (id CriterionID) IsZero() bool   { return id.value == "" }

// EvaluationID: у студента не более одной оценки по каждому критерию.
type EvaluationID struct {
	student   StudentID
	criterion CriterionID
}

func NewEvaluationID(student StudentID, criterion CriterionID) (EvaluationID, error) {
	if student.IsZero() || criterion.IsZero() {
		return EvaluationID{}, fmt.Errorf("оценка: %w", ErrEmptyID)
	}
	return EvaluationID{student: student, criterion: criterion}, nil
}

func (id EvaluationID) Student() StudentID     { return id.student }
func (id EvaluationID) Criterion() CriterionID { return id.criterion }
func (id EvaluationID) String() string         { return id.student.value + "/" + id.criterion.value }

type FullName struct{ value string }

func NewFullName(raw string) (FullName, error) {
	v, err := requireText(raw)
	if err != nil {
		return FullName{}, fmt.Errorf("ФИО: %w", err)
	}
	return FullName{value: v}, nil
}

func (n FullName) String() string { return n.value }
func (n FullName) IsZero() bool   { return n.value == "" }

type CriterionName struct{ value string }

func NewCriterionName(raw string) (CriterionName, error) {
	v, err := requireText(raw)
	if err != nil {
		return CriterionName{}, fmt.Errorf("название критерия: %w", err)
	}
	return CriterionName{value: v}, nil
}

func (n CriterionName) String() string { return n.value }
func (n CriterionName) IsZero() bool   { return n.value == "" }

func (n CriterionName) SameAs(other CriterionName) bool {
	return strings.EqualFold(n.value, other.value)
}

type Comment struct{ value string }

func NewComment(raw string) (Comment, error) {
	v := strings.TrimSpace(raw)
	if utf8.RuneCountInString(v) > maxCommentLength {
		return Comment{}, fmt.Errorf("комментарий: %w (не более %d символов)", ErrTooLong, maxCommentLength)
	}
	return Comment{value: v}, nil
}

func (c Comment) String() string { return c.value }

type Score struct{ value float64 }

func NewScore(v float64) (Score, error) {
	if !isFinite(v) || v < 0 {
		return Score{}, fmt.Errorf("%w: %s", ErrInvalidScore, formatNumber(v))
	}
	return Score{value: v}, nil
}

func (s Score) Value() float64 { return s.value }
func (s Score) String() string { return formatNumber(s.value) }

type MaxScore struct{ value float64 }

func NewMaxScore(v float64) (MaxScore, error) {
	if !isFinite(v) || v <= 0 {
		return MaxScore{}, fmt.Errorf("%w: %s", ErrInvalidMaxScore, formatNumber(v))
	}
	return MaxScore{value: v}, nil
}

func (m MaxScore) Value() float64      { return m.value }
func (m MaxScore) String() string      { return formatNumber(m.value) }
func (m MaxScore) IsZero() bool        { return m.value == 0 }
func (m MaxScore) Admits(s Score) bool { return s.value <= m.value }

type Weight struct{ value float64 }

func NewWeight(v float64) (Weight, error) {
	if !isFinite(v) || v <= 0 {
		return Weight{}, fmt.Errorf("%w: %s", ErrInvalidWeight, formatNumber(v))
	}
	return Weight{value: v}, nil
}

func (w Weight) Value() float64 { return w.value }
func (w Weight) String() string { return formatNumber(w.value) }
func (w Weight) IsZero() bool   { return w.value == 0 }

func requireID(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", ErrEmptyID
	}
	return v, nil
}

func requireText(raw string) (string, error) {
	v := strings.Join(strings.Fields(raw), " ")
	if v == "" {
		return "", ErrEmptyName
	}
	if utf8.RuneCountInString(v) > maxNameLength {
		return "", fmt.Errorf("%w (не более %d символов)", ErrTooLong, maxNameLength)
	}
	return v, nil
}

func isFinite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func formatNumber(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
