// Package ratelimit provides a small in-memory, per-key fixed-window
// limiter for guarding unauthenticated endpoints (login, setup) against
// brute-force attempts.
package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	mu       sync.Mutex
	max      int
	window   time.Duration
	attempts map[string][]time.Time
}

// New returns a Limiter allowing at most max calls per key within window.
func New(max int, window time.Duration) *Limiter {
	return &Limiter{max: max, window: window, attempts: make(map[string][]time.Time)}
}

// Allow reports whether a call under key is permitted right now, and
// records it if so.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)
	kept := l.attempts[key][:0]
	for _, t := range l.attempts[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}

	if len(kept) >= l.max {
		l.attempts[key] = kept
		return false
	}

	l.attempts[key] = append(kept, now)
	return true
}
