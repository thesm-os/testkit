// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package weird

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"iter"
	"sync"
	"time"
)

// --- Codec impl ---

// JSONCodec implements [Codec] using JSON encoding.
type JSONCodec struct{}

func NewJSONCodec() *JSONCodec { return &JSONCodec{} }

func (*JSONCodec) Encode(w io.Writer, v any) error {
	if w == nil {
		return ErrInvalidInput
	}
	return json.NewEncoder(w).Encode(v)
}

func (*JSONCodec) Decode(r io.Reader, v any) error {
	if r == nil {
		return ErrInvalidInput
	}
	return json.NewDecoder(r).Decode(v)
}

func (*JSONCodec) MarshalBinary(v any) ([]byte, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(v)
	return buf.Bytes(), err
}

func (*JSONCodec) ContentType() string {
	return "application/json"
}

func (*JSONCodec) Handles(mime string) bool {
	return mime == "application/json"
}

// --- Scheduler impl ---

// InMemoryScheduler implements [Scheduler] for spec testing.
type InMemoryScheduler struct {
	mu    sync.Mutex
	tasks map[string]Task
	name  string
}

func NewInMemoryScheduler(name string) *InMemoryScheduler {
	return &InMemoryScheduler{
		tasks: make(map[string]Task),
		name:  name,
	}
}

func (s *InMemoryScheduler) Schedule(ctx context.Context, id string, interval time.Duration, fn func(context.Context) error) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[id] = Task{ID: id, Interval: interval, Fn: fn}
	return nil
}

func (s *InMemoryScheduler) Cancel(ctx context.Context, id string) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[id]; !ok {
		return ErrNotScheduled
	}
	delete(s.tasks, id)
	return nil
}

func (s *InMemoryScheduler) Status(ctx context.Context, id string) (TaskStatus, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return TaskStatus{}, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return TaskStatus{}, ErrNotScheduled
	}
	return TaskStatus{ID: t.ID, Running: true}, nil
}

func (s *InMemoryScheduler) Running(ctx context.Context) (int, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.tasks), nil
}

func (s *InMemoryScheduler) Flush(ctx context.Context) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks = make(map[string]Task)
	return nil
}

func (s *InMemoryScheduler) Tasks(ctx context.Context) iter.Seq2[TaskStatus, error] {
	return func(yield func(TaskStatus, error) bool) {
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, t := range s.tasks {
			if ctx != nil && ctx.Err() != nil {
				yield(TaskStatus{}, ctx.Err())
				return
			}
			if !yield(TaskStatus{ID: t.ID, Running: true}, nil) {
				return
			}
		}
	}
}

func (s *InMemoryScheduler) Name() string {
	return s.name
}
