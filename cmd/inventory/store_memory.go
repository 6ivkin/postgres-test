package main

import (
	"cmp"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
)

var errPartNotFound = errors.New("деталь не найдена")

type memoryStore struct {
	mu    sync.RWMutex
	parts map[string]Part
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		parts: make(map[string]Part),
	}
}

func (s *memoryStore) Create(req CreatePartRequest) Part {
	s.mu.Lock()
	defer s.mu.Unlock()

	part := Part{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Quantity:  req.Quantity,
		Price:     req.Price,
		Reserved:  0,
		CreatedAt: time.Now().UTC(),
	}

	s.parts[part.ID] = part

	return part
}

func (s *memoryStore) GetByID(id string) (Part, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	part, ok := s.parts[id]
	if !ok {
		return Part{}, errPartNotFound
	}

	return part, nil
}

func (s *memoryStore) List() []Part {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Part, 0, len(s.parts))
	for _, part := range s.parts {
		result = append(result, part)
	}

	slices.SortFunc(result, func(a, b Part) int {
		return cmp.Compare(a.ID, b.ID)
	})

	return result
}
