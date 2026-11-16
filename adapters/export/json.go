package export

import (
	"encoding/json"
	"io"
	"time"

	"pos/app"
)

type JSONExporter struct{}

func (JSONExporter) Format() string { return "json" }

type jsonReport struct {
	Policy      string       `json:"policy"`
	GeneratedAt time.Time    `json:"generated_at"`
	Scale       float64      `json:"scale"`
	Results     []jsonResult `json:"results"`
}

type jsonResult struct {
	StudentID  string     `json:"student_id"`
	Student    string     `json:"student"`
	FinalGrade float64    `json:"final_grade"`
	Criteria   []jsonLine `json:"criteria"`
}

type jsonLine struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Score    *float64 `json:"score"`
	MaxScore float64  `json:"max_score"`
	Weight   float64  `json:"weight"`
}

func (JSONExporter) Export(path string, report app.ResultReport) error {
	out := jsonReport{
		Policy:      report.Policy,
		GeneratedAt: report.GeneratedAt,
		Scale:       report.Scale,
		Results:     make([]jsonResult, len(report.Grades)),
	}
	for i, g := range report.Grades {
		lines := make([]jsonLine, len(g.Lines))
		for j, l := range g.Lines {
			lines[j] = jsonLine{ID: l.Criterion.ID, Name: l.Criterion.Name, MaxScore: l.Criterion.MaxScore, Weight: l.Criterion.Weight}
			if l.Graded {
				score := l.Score
				lines[j].Score = &score
			}
		}
		out.Results[i] = jsonResult{StudentID: g.StudentID, Student: g.StudentName, FinalGrade: g.Value, Criteria: lines}
	}
	return writeFile(path, func(w io.Writer) error {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	})
}
