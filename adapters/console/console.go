package console

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"pos/app"
)

var (
	errInputClosed = errors.New("ввод завершён")
	errCancelled   = errors.New("отменено")
)

type UI struct {
	app app.Facade
	in  *bufio.Scanner
	out io.Writer
}

func New(facade app.Facade, in io.Reader, out io.Writer) *UI {
	return &UI{app: facade, in: bufio.NewScanner(in), out: out}
}

type menuItem struct {
	title string
	run   func() error
}

type menuGroup struct {
	title string
	items []menuItem
}

func (u *UI) menu() []menuGroup {
	return []menuGroup{
		{"Студенты", []menuItem{
			{"Список студентов", u.listStudents},
			{"Добавить студента", u.addStudent},
			{"Переименовать студента", u.renameStudent},
			{"Удалить студента", u.removeStudent},
		}},
		{"Рубрикатор", []menuItem{
			{"Список критериев", u.listCriteria},
			{"Добавить критерий", u.addCriterion},
			{"Изменить критерий (название, вес)", u.editCriterion},
			{"Удалить критерий", u.removeCriterion},
		}},
		{"Оценки", []menuItem{
			{"Выставить оценку", u.grade},
			{"Исправить оценку", u.correctEvaluation},
			{"Отменить оценку", u.cancelEvaluation},
			{"Посмотреть оценки", u.listEvaluations},
		}},
		{"Итоги", []menuItem{
			{"Итог студента с расшифровкой", u.finalGrade},
			{"Ведомость итогов", u.finalGrades},
			{"Статистика по критериям", u.criterionStats},
			{"Сменить правило итога", u.selectPolicy},
			{"Экспорт итогов", u.exportResults},
		}},
	}
}

func (u *UI) Run() error {
	groups := u.menu()
	var actions []menuItem
	for _, g := range groups {
		actions = append(actions, g.items...)
	}

	u.printf("ПОС — Преподавательская Оценка Студентов\n")
	for {
		u.printMenu(groups)
		choice, err := u.ask("Пункт меню")
		if errors.Is(err, errInputClosed) || choice == "0" {
			u.printf("До встречи!\n")
			return nil
		}
		if err != nil {
			return err
		}
		n, convErr := strconv.Atoi(choice)
		if convErr != nil || n < 1 || n > len(actions) {
			u.printf("Нет такого пункта: %q\n", choice)
			continue
		}
		u.printf("\n— %s —\n", actions[n-1].title)
		switch err := actions[n-1].run(); {
		case err == nil:
		case errors.Is(err, errInputClosed):
			return nil
		case errors.Is(err, errCancelled):
			u.printf("Отменено.\n")
		default:
			u.printf("Ошибка: %v\n", err)
		}
	}
}

func (u *UI) printMenu(groups []menuGroup) {
	n := 1
	for _, g := range groups {
		u.printf("\n%s:\n", g.title)
		for _, item := range g.items {
			u.printf("  %2d. %s\n", n, item.title)
			n++
		}
	}
	u.printf("\n   0. Выход\n")
}

func (u *UI) listStudents() error {
	students, err := u.app.ListStudents()
	if err != nil {
		return err
	}
	u.printStudents(students)
	return nil
}

func (u *UI) addStudent() error {
	name, err := u.ask("ФИО")
	if err != nil {
		return err
	}
	s, err := u.app.AddStudent(name)
	if err != nil {
		return err
	}
	u.printf("Добавлен студент %s — %s\n", s.ID, s.Name)
	return nil
}

func (u *UI) renameStudent() error {
	id, err := u.ask("ID студента")
	if err != nil {
		return err
	}
	name, err := u.ask("Новое ФИО")
	if err != nil {
		return err
	}
	if err := u.app.RenameStudent(id, name); err != nil {
		return err
	}
	u.printf("Готово.\n")
	return nil
}

func (u *UI) removeStudent() error {
	id, err := u.ask("ID студента")
	if err != nil {
		return err
	}
	if err := u.confirm("Удалить студента вместе с его оценками?"); err != nil {
		return err
	}
	if err := u.app.RemoveStudent(id); err != nil {
		return err
	}
	u.printf("Студент %s удалён.\n", id)
	return nil
}

func (u *UI) listCriteria() error {
	criteria, err := u.app.ListCriteria()
	if err != nil {
		return err
	}
	u.printCriteria(criteria)
	return nil
}

func (u *UI) addCriterion() error {
	name, err := u.ask("Название")
	if err != nil {
		return err
	}
	maxScore, err := u.askNumber("Максимальный балл")
	if err != nil {
		return err
	}
	weight, err := u.askNumber("Вес")
	if err != nil {
		return err
	}
	c, err := u.app.AddCriterion(name, maxScore, weight)
	if err != nil {
		return err
	}
	u.printf("Добавлен критерий %s — %s (max %s, вес %s)\n", c.ID, c.Name, num(c.MaxScore), num(c.Weight))
	return nil
}

func (u *UI) editCriterion() error {
	id, err := u.ask("ID критерия")
	if err != nil {
		return err
	}
	name, err := u.ask("Новое название (пусто — оставить)")
	if err != nil {
		return err
	}
	weight, hasWeight, err := u.askOptionalNumber("Новый вес (пусто — оставить)")
	if err != nil {
		return err
	}
	if name != "" {
		if err := u.app.RenameCriterion(id, name); err != nil {
			return err
		}
	}
	if hasWeight {
		if err := u.app.ChangeCriterionWeight(id, weight); err != nil {
			return err
		}
	}
	u.printf("Готово.\n")
	return nil
}

func (u *UI) removeCriterion() error {
	id, err := u.ask("ID критерия")
	if err != nil {
		return err
	}
	if err := u.confirm("Удалить критерий вместе со всеми оценками по нему?"); err != nil {
		return err
	}
	if err := u.app.RemoveCriterion(id); err != nil {
		return err
	}
	u.printf("Критерий %s удалён.\n", id)
	return nil
}

func (u *UI) grade() error {
	return u.evaluate(u.app.Grade, "Выставлено")
}

func (u *UI) correctEvaluation() error {
	return u.evaluate(u.app.CorrectEvaluation, "Исправлено")
}

func (u *UI) evaluate(action func(string, string, float64, string) (app.EvaluationView, error), done string) error {
	studentID, err := u.ask("ID студента")
	if err != nil {
		return err
	}
	criterionID, err := u.ask("ID критерия")
	if err != nil {
		return err
	}
	score, err := u.askNumber("Балл")
	if err != nil {
		return err
	}
	comment, err := u.ask("Комментарий (необязательно)")
	if err != nil {
		return err
	}
	e, err := action(studentID, criterionID, score, comment)
	if err != nil {
		return err
	}
	u.printf("%s: %s — «%s»: %s из %s\n", done, e.StudentName, e.CriterionName, num(e.Score), num(e.MaxScore))
	return nil
}

func (u *UI) cancelEvaluation() error {
	studentID, err := u.ask("ID студента")
	if err != nil {
		return err
	}
	criterionID, err := u.ask("ID критерия")
	if err != nil {
		return err
	}
	if err := u.app.CancelEvaluation(studentID, criterionID); err != nil {
		return err
	}
	u.printf("Оценка отменена.\n")
	return nil
}

func (u *UI) listEvaluations() error {
	studentID, err := u.ask("ID студента (пусто — все)")
	if err != nil {
		return err
	}
	evaluations, err := u.app.ListEvaluations(studentID)
	if err != nil {
		return err
	}
	u.printEvaluations(evaluations)
	return nil
}

func (u *UI) finalGrade() error {
	studentID, err := u.ask("ID студента")
	if err != nil {
		return err
	}
	g, err := u.app.FinalGrade(studentID)
	if err != nil {
		return err
	}
	u.printFinalGrade(g)
	return nil
}

func (u *UI) finalGrades() error {
	grades, err := u.app.FinalGrades()
	if err != nil {
		return err
	}
	u.printFinalGrades(grades, u.app.Policies()[u.app.CurrentPolicy()])
	return nil
}

func (u *UI) criterionStats() error {
	stats, err := u.app.CriterionStats()
	if err != nil {
		return err
	}
	u.printStats(stats)
	return nil
}

func (u *UI) selectPolicy() error {
	current := u.app.CurrentPolicy()
	for i, name := range u.app.Policies() {
		marker := " "
		if i == current {
			marker = "*"
		}
		u.printf(" %s %d. %s\n", marker, i+1, name)
	}
	choice, err := u.ask("Номер правила")
	if err != nil {
		return err
	}
	n, convErr := strconv.Atoi(choice)
	if convErr != nil {
		return fmt.Errorf("«%s» — не номер", choice)
	}
	if err := u.app.SelectPolicy(n - 1); err != nil {
		return err
	}
	u.printf("Правило итога: %s\n", u.app.Policies()[n-1])
	return nil
}

func (u *UI) exportResults() error {
	formats := strings.Join(u.app.ExportFormats(), ", ")
	path, err := u.ask(fmt.Sprintf("Файл (%s; пусто — build/results.csv)", formats))
	if err != nil {
		return err
	}
	if path == "" {
		path = "build/results.csv"
	}
	n, err := u.app.ExportResults(path)
	if err != nil {
		return err
	}
	u.printf("Выгружены итоги %d студ. в %s\n", n, path)
	return nil
}

func (u *UI) ask(prompt string) (string, error) {
	u.printf("%s: ", prompt)
	if !u.in.Scan() {
		u.printf("\n")
		if err := u.in.Err(); err != nil {
			return "", err
		}
		return "", errInputClosed
	}
	return strings.TrimSpace(u.in.Text()), nil
}

func (u *UI) askNumber(prompt string) (float64, error) {
	v, ok, err := u.askOptionalNumber(prompt)
	if err == nil && !ok {
		return 0, fmt.Errorf("нужно ввести число")
	}
	return v, err
}

func (u *UI) askOptionalNumber(prompt string) (float64, bool, error) {
	raw, err := u.ask(prompt)
	if err != nil || raw == "" {
		return 0, false, err
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(raw, ",", "."), 64)
	if err != nil {
		return 0, false, fmt.Errorf("«%s» — не число", raw)
	}
	return v, true, nil
}

func (u *UI) confirm(question string) error {
	answer, err := u.ask(question + " (д/н)")
	if err != nil {
		return err
	}
	switch strings.ToLower(answer) {
	case "д", "да", "y", "yes":
		return nil
	}
	return errCancelled
}

func (u *UI) printf(format string, args ...any) {
	fmt.Fprintf(u.out, format, args...)
}
