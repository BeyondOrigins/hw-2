package export

import (
	"encoding/csv"
	"io"
	"strconv"

	"pos/app"
)

type CSVExporter struct{}

func (CSVExporter) Format() string { return "csv" }

func (CSVExporter) Export(path string, report app.ResultReport) error {
	return writeFile(path, func(w io.Writer) error {
		header := []string{"student_id", "student"}
		for _, c := range report.Criteria {
			header = append(header, c.Name+" (из "+formatNumber(c.MaxScore)+")")
		}
		header = append(header, "final_grade", "policy")

		rows := [][]string{header}
		for _, g := range report.Grades {
			row := []string{g.StudentID, g.StudentName}
			for _, l := range g.Lines {
				cell := ""
				if l.Graded {
					cell = formatNumber(l.Score)
				}
				row = append(row, cell)
			}
			row = append(row, strconv.FormatFloat(g.Value, 'f', 2, 64), report.Policy)
			rows = append(rows, row)
		}
		return csv.NewWriter(w).WriteAll(rows)
	})
}
