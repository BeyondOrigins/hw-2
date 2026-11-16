package domain

import "fmt"

type Student struct {
	id   StudentID
	name FullName
}

func NewStudent(id StudentID, name FullName) (*Student, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("студент: %w", ErrEmptyID)
	}
	if name.IsZero() {
		return nil, fmt.Errorf("студент: ФИО: %w", ErrEmptyName)
	}
	return &Student{id: id, name: name}, nil
}

func (s *Student) ID() StudentID  { return s.id }
func (s *Student) Name() FullName { return s.name }

func (s *Student) Rename(name FullName) error {
	if name.IsZero() {
		return fmt.Errorf("студент: ФИО: %w", ErrEmptyName)
	}
	s.name = name
	return nil
}
