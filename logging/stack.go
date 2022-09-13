package logging

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

type entry struct {
	logEntry *logrus.Entry
}

func (e *entry) Level(lvl logrus.Level) {
	e.logEntry.Level = lvl
}

type Level interface {
	Level(level logrus.Level)
}

type Stack struct {
	entries []*entry
	mu      sync.Mutex
}

type ctxKey uint8

const logStack ctxKey = iota

func NewStack(ctx context.Context) (context.Context, *Stack) {
	s := &Stack{}
	return context.WithValue(ctx, logStack, s), s
}

func (s *Stack) Push(e *logrus.Entry) Level {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := &entry{logEntry: e}
	s.entries = append(s.entries, item)
	return item
}

func (s *Stack) Fire() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, item := range s.entries {
		item.logEntry.Log(item.logEntry.Level)
	}
}

func FromContext(ctx context.Context) (*Stack, bool) {
	s, exist := ctx.Value(logStack).(*Stack)
	return s, exist
}
