package auditsvc

import (
	"fmt"
	"path/filepath"
	"testing"
)

func newTestLog(t *testing.T) *Log {
	t.Helper()
	l, err := NewLog(filepath.Join(t.TempDir(), "activity.log"))
	if err != nil {
		t.Fatalf("NewLog: %v", err)
	}
	return l
}

func TestRecordAndRecent(t *testing.T) {
	l := newTestLog(t)

	l.Record("alice", "created network", "office (abc123)", "")
	l.Record("bob", "deleted network", "lab (def456)", "")

	entries, err := l.Recent(10)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	// Newest first.
	if entries[0].User != "bob" || entries[0].Action != "deleted network" {
		t.Errorf("entries[0] = %+v, want bob/deleted network", entries[0])
	}
	if entries[1].User != "alice" {
		t.Errorf("entries[1] = %+v, want alice", entries[1])
	}
}

func TestRecentRespectsLimit(t *testing.T) {
	l := newTestLog(t)
	for i := 0; i < 5; i++ {
		l.Record("alice", fmt.Sprintf("action-%d", i), "target", "")
	}
	entries, err := l.Recent(2)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if entries[0].Action != "action-4" || entries[1].Action != "action-3" {
		t.Errorf("entries = %+v, want action-4 then action-3", entries)
	}
}

func TestRecentOnEmptyLog(t *testing.T) {
	l := newTestLog(t)
	entries, err := l.Recent(10)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("len(entries) = %d, want 0", len(entries))
	}
}

func TestRecordTrimsToMaxEntries(t *testing.T) {
	l := newTestLog(t)
	total := maxEntries + 20
	for i := 0; i < total; i++ {
		l.Record("alice", fmt.Sprintf("action-%d", i), "target", "")
	}

	entries, err := l.Recent(maxEntries + 100)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(entries) != maxEntries {
		t.Fatalf("len(entries) = %d, want %d", len(entries), maxEntries)
	}
	// The oldest 20 entries (action-0 .. action-19) must have been
	// dropped; the newest, action-(total-1), must still be entries[0].
	if entries[0].Action != fmt.Sprintf("action-%d", total-1) {
		t.Errorf("entries[0] = %+v, want action-%d", entries[0], total-1)
	}
	oldest := entries[len(entries)-1]
	if oldest.Action != "action-20" {
		t.Errorf("oldest kept entry = %+v, want action-20", oldest)
	}
}
