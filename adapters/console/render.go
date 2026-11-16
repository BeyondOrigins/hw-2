package console

import (
	"fmt"
	"math"
	"strconv"
	"text/tabwriter"

	"pos/app"
)

func (u *UI) table(header string, rows func(w *tabwriter.Writer)) {
	w := tabwriter.NewWriter(u.out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, header)
	rows(w)
	w.Flush()
}

func (u *UI) printStudents(students []app.StudentView) {
	if len(students) == 0 {
		u.printf("Студентов пока нет.\n")
		return
	}
	u.table("ID\tФИО", func(w *tabwriter.Writer) {
		for _, s := range students {
			fmt.Fprintf(w, "%s\t%s\n", s.ID, s.Name)
		}
	})
}

func (u *UI) printCriteria(criteria []app.CriterionView) {
	if len(criteria) == 0 {
		u.printf("Рубрикатор пуст.\n")
		return
	}
	var total float64
	for _, c := range criteria {
		total += c.Weight
	}
	u.table("ID\tКритерий\tМакс.\tВес\tДоля в итоге", func(w *tabwriter.Writer) {
		for _, c := range criteria {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.1f%%\n", c.ID, c.Name, num(c.MaxScore), num(c.Weight), 100*c.Weight/total)
		}
	})
}

func (u *UI) printEvaluations(evaluations []app.EvaluationView) {
	if len(evaluations) == 0 {
		u.printf("Оценок нет.\n")
		return
	}
	u.table("Студент\tКритерий\tБалл\tДата\tКомментарий", func(w *tabwriter.Writer) {
		for _, e := range evaluations {
			fmt.Fprintf(w, "%s %s\t%s %s\t%s / %s\t%s\t%s\n",
				e.StudentID, e.StudentName, e.CriterionID, e.CriterionName,
				num(e.Score), num(e.MaxScore), e.GradedAt.Format("02.01.2006 15:04"), e.Comment)
		}
	})
}

func (u *UI) printFinalGrade(g app.FinalGradeView) {
	u.printf("%s %s\n", g.StudentID, g.StudentName)
	u.table("Критерий\tБалл\tВес", func(w *tabwriter.Writer) {
		for _, l := range g.Lines {
			score := "—"
			if l.Graded {
				score = num(l.Score) + " / " + num(l.Criterion.MaxScore)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", l.Criterion.Name, score, num(l.Criterion.Weight))
		}
	})
	u.printf("Итог: %s из %s (оценено критериев: %d из %d)\n", grade(g.Value), num(g.Scale), g.Graded, len(g.Lines))
	u.printf("Правило: %s\n", g.Policy)
}

func (u *UI) printFinalGrades(grades []app.FinalGradeView, policy string) {
	if len(grades) == 0 {
		u.printf("Студентов пока нет.\n")
		return
	}
	u.printf("Правило: %s\n", policy)
	u.table("ID\tФИО\tИтог\tОценено", func(w *tabwriter.Writer) {
		for _, g := range grades {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d / %d\n", g.StudentID, g.StudentName, grade(g.Value), g.Graded, len(g.Lines))
		}
	})
}

func (u *UI) printStats(stats []app.CriterionStatsView) {
	if len(stats) == 0 {
		u.printf("Рубрикатор пуст.\n")
		return
	}
	u.table("Критерий\tОценено\tСредний\tМин\tМакс", func(w *tabwriter.Writer) {
		for _, s := range stats {
			if s.Graded == 0 {
				fmt.Fprintf(w, "%s\t0 / %d\t—\t—\t—\n", s.Criterion.Name, s.Students)
				continue
			}
			fmt.Fprintf(w, "%s\t%d / %d\t%s / %s\t%s\t%s\n", s.Criterion.Name, s.Graded, s.Students,
				grade(s.Average), num(s.Criterion.MaxScore), num(s.Min), num(s.Max))
		}
	})
}

func num(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

func grade(v float64) string { return num(math.Round(v*100) / 100) }
