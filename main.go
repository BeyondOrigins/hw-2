package main

import (
	"flag"
	"fmt"
	"os"

	"go.uber.org/dig"

	"pos/adapters/console"
	"pos/adapters/export"
	"pos/adapters/jsonstore"
	"pos/adapters/system"
	"pos/app"
	"pos/domain"
)

func main() {
	dataPath := flag.String("data", "build/pos-data.json", "файл с данными")
	demo := flag.Bool("demo", false, "прогнать сквозной сценарий на данных в памяти")
	flag.Parse()

	if *demo {
		*dataPath = ""
	}
	if err := run(*dataPath, *demo); err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", dig.RootCause(err))
		os.Exit(1)
	}
}

func run(dataPath string, demo bool) error {
	container, err := buildContainer(dataPath)
	if err != nil {
		return err
	}
	return container.Invoke(func(ui *console.UI) error {
		if demo {
			return ui.RunDemo("build")
		}
		return ui.Run()
	})
}

type provider struct {
	constructor any
	options     []dig.ProvideOption
}

func buildContainer(dataPath string) (*dig.Container, error) {
	exporter := []dig.ProvideOption{dig.Group("exporters")}
	providers := []provider{
		{func() domain.Clock { return system.Clock{} }, nil},
		{domain.NewEvaluationFactory, nil},
		{gradingPolicies, nil},

		{func() (*jsonstore.Store, error) { return jsonstore.Open(dataPath) }, nil},
		{jsonstore.NewStudentRepository, []dig.ProvideOption{dig.As(new(domain.StudentRepository))}},
		{jsonstore.NewRubricRepository, []dig.ProvideOption{dig.As(new(domain.RubricRepository))}},
		{jsonstore.NewEvaluationRepository, []dig.ProvideOption{dig.As(new(domain.EvaluationRepository))}},

		{func() app.ResultExporter { return export.CSVExporter{} }, exporter},
		{func() app.ResultExporter { return export.JSONExporter{} }, exporter},

		{newService, []dig.ProvideOption{dig.As(new(app.Facade))}},
		{func(f app.Facade) *console.UI { return console.New(f, os.Stdin, os.Stdout) }, nil},
	}

	c := dig.New()
	for _, p := range providers {
		if err := c.Provide(p.constructor, p.options...); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// Первое правило — правило по умолчанию.
func gradingPolicies() []domain.GradingPolicy {
	return []domain.GradingPolicy{
		domain.NewRoundedPolicy(domain.WeightedAveragePolicy{}, 0),
		domain.WeightedAveragePolicy{},
		domain.NewRoundedPolicy(domain.GradedOnlyPolicy{}, 2),
	}
}

type serviceParams struct {
	dig.In

	Students    domain.StudentRepository
	Rubrics     domain.RubricRepository
	Evaluations domain.EvaluationRepository
	Factory     *domain.EvaluationFactory
	Clock       domain.Clock
	Policies    []domain.GradingPolicy
	Exporters   []app.ResultExporter `group:"exporters"`
}

func newService(p serviceParams) (*app.Service, error) {
	return app.NewService(app.Dependencies{
		Students:    p.Students,
		Rubrics:     p.Rubrics,
		Evaluations: p.Evaluations,
		Factory:     p.Factory,
		Clock:       p.Clock,
		Policies:    p.Policies,
		Exporters:   p.Exporters,
	})
}
