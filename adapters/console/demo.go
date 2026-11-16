package console

import (
	"path/filepath"

	"pos/app"
)

func (u *UI) RunDemo(exportDir string) error {
	step := func(title string) { u.printf("\n=== %s ===\n", title) }

	step("1. Заводим студентов")
	var students []app.StudentView
	for _, name := range []string{"Иванов Иван", "Петрова Анна", "Сидоров Пётр"} {
		s, err := u.app.AddStudent(name)
		if err != nil {
			return err
		}
		students = append(students, s)
	}
	u.printStudents(students)

	step("2. Собираем рубрикатор")
	var criteria []app.CriterionView
	for _, c := range []struct {
		name        string
		max, weight float64
	}{
		{"Домашние задания", 10, 0.4},
		{"Контрольная работа", 10, 0.2},
		{"Экзамен", 100, 0.4},
	} {
		created, err := u.app.AddCriterion(c.name, c.max, c.weight)
		if err != nil {
			return err
		}
		criteria = append(criteria, created)
	}
	u.printCriteria(criteria)

	step("3. Выставляем оценки")
	hw, test, exam := criteria[0].ID, criteria[1].ID, criteria[2].ID
	for _, g := range []struct {
		student, criterion string
		score              float64
		comment            string
	}{
		{students[0].ID, hw, 9, ""},
		{students[0].ID, test, 8, ""},
		{students[0].ID, exam, 85, "уверенный ответ"},
		{students[1].ID, hw, 10, "все ДЗ в срок"},
		{students[1].ID, test, 7.5, ""},
		{students[1].ID, exam, 92, ""},
		{students[2].ID, hw, 6, ""},
		{students[2].ID, test, 4, ""},
	} {
		if _, err := u.app.Grade(g.student, g.criterion, g.score, g.comment); err != nil {
			return err
		}
	}
	all, err := u.app.ListEvaluations("")
	if err != nil {
		return err
	}
	u.printEvaluations(all)

	step("4. Домен не даёт нарушить правила")
	expectRejected := func(title string, err error) {
		if err == nil {
			u.printf("  %s: неожиданно принято!\n", title)
			return
		}
		u.printf("  %s → отклонено: %v\n", title, err)
	}
	_, err = u.app.Grade(students[0].ID, hw, 12, "")
	expectRejected("12 баллов из 10", err)
	_, err = u.app.Grade(students[0].ID, "c99", 5, "")
	expectRejected("критерий не из рубрикатора", err)
	_, err = u.app.Grade(students[0].ID, hw, 7, "")
	expectRejected("повторная оценка по тому же критерию", err)
	_, err = u.app.CorrectEvaluation(students[1].ID, exam, 120, "")
	expectRejected("исправление на 120 из 100", err)

	step("5. Исправляем оценку")
	corrected, err := u.app.CorrectEvaluation(students[2].ID, test, 6, "пересмотр работы")
	if err != nil {
		return err
	}
	u.printf("%s — «%s»: теперь %s из %s (%s)\n",
		corrected.StudentName, corrected.CriterionName, num(corrected.Score), num(corrected.MaxScore), corrected.Comment)

	step("6. Итог студента с расшифровкой")
	g, err := u.app.FinalGrade(students[0].ID)
	if err != nil {
		return err
	}
	u.printFinalGrade(g)

	step("7. Ведомость итогов при разных правилах")
	current := u.app.CurrentPolicy()
	for i, name := range u.app.Policies() {
		if err := u.app.SelectPolicy(i); err != nil {
			return err
		}
		grades, err := u.app.FinalGrades()
		if err != nil {
			return err
		}
		u.printf("\n")
		u.printFinalGrades(grades, name)
	}
	if err := u.app.SelectPolicy(current); err != nil {
		return err
	}

	step("8. Статистика по критериям")
	stats, err := u.app.CriterionStats()
	if err != nil {
		return err
	}
	u.printStats(stats)

	step("9. Экспорт итогов")
	for _, format := range u.app.ExportFormats() {
		path := filepath.Join(exportDir, "results."+format)
		n, err := u.app.ExportResults(path)
		if err != nil {
			return err
		}
		u.printf("Итоги %d студ. выгружены в %s\n", n, path)
	}
	return nil
}
