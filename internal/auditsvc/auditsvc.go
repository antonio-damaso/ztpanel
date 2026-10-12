// Package auditsvc keeps a simple, append-only activity log: who did
// what, and when. It exists so a shared admin tool with several users
// has a paper trail, the way ztnet's "Activity Logs" panel does.
package auditsvc

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"
)

const maxEntries = 500

type Entry struct {
	Time   time.Time `json:"time"`
	User   string    `json:"user"`
	Action string    `json:"action"`
	// Target is a short human-readable reference to what was
	// changed, e.g. a network name or "user colega".
	Target string `json:"target"`
	Detail string `json:"detail,omitempty"`
}

// Log is an append-only JSON-lines file, trimmed to the most recent
// maxEntries on write so it never grows unbounded.
type Log struct {
	path string
	mu   sync.Mutex
}

func NewLog(path string) (*Log, error) {
	l := &Log{path: path}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, err
		}
		f.Close()
	}
	return l, nil
}

func (l *Log) Record(user, action, target, detail string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entries, err := l.readAll()
	if err != nil {
		entries = nil
	}
	entries = append(entries, Entry{Time: time.Now(), User: user, Action: action, Target: target, Detail: detail})
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}
	_ = l.writeAll(entries)
}

// Recent returns up to `limit` entries, most recent first.
func (l *Log) Recent(limit int) ([]Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entries, err := l.readAll()
	if err != nil {
		return nil, err
	}
	// Reverse into newest-first.
	out := make([]Entry, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		out = append(out, entries[i])
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (l *Log) readAll() ([]Entry, error) {
	f, err := os.Open(l.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}

func (l *Log) writeAll(entries []Entry) error {
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			return err
		}
	}
	return nil
}
