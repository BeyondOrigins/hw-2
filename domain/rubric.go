package domain

import "fmt"

type Criterion struct {
	id       CriterionID
	name     CriterionName
	maxScore MaxScore
	weight   Weight
}

func (c Criterion) ID() CriterionID     { return c.id }
func (c Criterion) Name() CriterionName { return c.name }
func (c Criterion) MaxScore() MaxScore  { return c.maxScore }
func (c Criterion) Weight() Weight      { return c.weight }

func (c Criterion) CheckScore(s Score) error {
	if !c.maxScore.Admits(s) {
		return fmt.Errorf("%w: %s из %s по критерию «%s»", ErrScoreOutOfRange, s, c.maxScore, c.name)
	}
	return nil
}

func (c Criterion) Normalized(s Score) float64 {
	return s.value / c.maxScore.value
}

type Rubric struct {
	criteria []*Criterion
}

func NewRubric() *Rubric { return &Rubric{} }

func (r *Rubric) AddCriterion(id CriterionID, name CriterionName, maxScore MaxScore, weight Weight) error {
	switch {
	case id.IsZero():
		return fmt.Errorf("критерий: %w", ErrEmptyID)
	case name.IsZero():
		return fmt.Errorf("критерий: %w", ErrEmptyName)
	case maxScore.IsZero():
		return ErrInvalidMaxScore
	case weight.IsZero():
		return ErrInvalidWeight
	}
	for _, c := range r.criteria {
		if c.id == id {
			return fmt.Errorf("%w: %s", ErrDuplicateCriterion, id)
		}
		if c.name.SameAs(name) {
			return fmt.Errorf("%w: «%s»", ErrDuplicateCriterion, name)
		}
	}
	r.criteria = append(r.criteria, &Criterion{id: id, name: name, maxScore: maxScore, weight: weight})
	return nil
}

func (r *Rubric) Criterion(id CriterionID) (Criterion, error) {
	c, err := r.find(id)
	if err != nil {
		return Criterion{}, err
	}
	return *c, nil
}

func (r *Rubric) Criteria() []Criterion {
	out := make([]Criterion, len(r.criteria))
	for i, c := range r.criteria {
		out[i] = *c
	}
	return out
}

func (r *Rubric) IsEmpty() bool { return len(r.criteria) == 0 }

func (r *Rubric) RenameCriterion(id CriterionID, name CriterionName) error {
	if name.IsZero() {
		return fmt.Errorf("критерий: %w", ErrEmptyName)
	}
	target, err := r.find(id)
	if err != nil {
		return err
	}
	for _, c := range r.criteria {
		if c.id != id && c.name.SameAs(name) {
			return fmt.Errorf("%w: «%s»", ErrDuplicateCriterion, name)
		}
	}
	target.name = name
	return nil
}

func (r *Rubric) ChangeWeight(id CriterionID, weight Weight) error {
	if weight.IsZero() {
		return ErrInvalidWeight
	}
	target, err := r.find(id)
	if err != nil {
		return err
	}
	target.weight = weight
	return nil
}

func (r *Rubric) RemoveCriterion(id CriterionID) error {
	for i, c := range r.criteria {
		if c.id == id {
			r.criteria = append(r.criteria[:i], r.criteria[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrCriterionNotInRubric, id)
}

func (r *Rubric) find(id CriterionID) (*Criterion, error) {
	for _, c := range r.criteria {
		if c.id == id {
			return c, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrCriterionNotInRubric, id)
}
