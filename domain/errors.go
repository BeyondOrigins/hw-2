package domain

import "errors"

var (
	ErrEmptyID   = errors.New("идентификатор не может быть пустым")
	ErrEmptyName = errors.New("название не может быть пустым")
	ErrTooLong   = errors.New("слишком длинный текст")

	ErrInvalidScore    = errors.New("балл должен быть неотрицательным числом")
	ErrInvalidMaxScore = errors.New("максимальный балл должен быть положительным числом")
	ErrInvalidWeight   = errors.New("вес критерия должен быть положительным числом")
	ErrScoreOutOfRange = errors.New("балл выходит за пределы критерия")

	ErrCriterionNotInRubric = errors.New("критерия нет в рубрикаторе")
	ErrDuplicateCriterion   = errors.New("такой критерий уже есть в рубрикаторе")
	ErrEmptyRubric          = errors.New("рубрикатор пуст")

	ErrStudentRequired     = errors.New("не указан студент")
	ErrForeignEvaluation   = errors.New("оценка принадлежит другому студенту")
	ErrDuplicateEvaluation = errors.New("оценка по этому критерию уже выставлена")
	ErrInvalidFinalGrade   = errors.New("итог вне шкалы")

	ErrNotFound = errors.New("не найдено")
)
