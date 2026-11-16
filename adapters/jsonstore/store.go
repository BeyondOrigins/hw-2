package jsonstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

type document struct {
	Students    []studentDTO    `json:"students"`
	Criteria    []criterionDTO  `json:"criteria"`
	Evaluations []evaluationDTO `json:"evaluations"`
}

type studentDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type criterionDTO struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	MaxScore float64 `json:"max_score"`
	Weight   float64 `json:"weight"`
}

type evaluationDTO struct {
	StudentID   string    `json:"student_id"`
	CriterionID string    `json:"criterion_id"`
	Score       float64   `json:"score"`
	Comment     string    `json:"comment,omitempty"`
	GradedAt    time.Time `json:"graded_at"`
}

func (d document) clone() document {
	return document{
		Students:    slices.Clone(d.Students),
		Criteria:    slices.Clone(d.Criteria),
		Evaluations: slices.Clone(d.Evaluations),
	}
}

type Store struct {
	path string
	mu   sync.Mutex
	doc  document
}

func Open(path string) (*Store, error) {
	s := &Store{path: path}
	if path == "" {
		return s, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("чтение %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &s.doc); err != nil {
		return nil, fmt.Errorf("разбор %s: %w", path, err)
	}
	return s, nil
}

func (s *Store) read(fn func(doc *document) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(&s.doc)
}

func (s *Store) write(fn func(doc *document) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	backup := s.doc.clone()
	if err := fn(&s.doc); err != nil {
		s.doc = backup
		return err
	}
	if err := s.flush(); err != nil {
		s.doc = backup
		return err
	}
	return nil
}

func (s *Store) flush() error {
	if s.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(s.doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("запись %s: %w", s.path, err)
	}
	return os.Rename(tmp, s.path)
}
