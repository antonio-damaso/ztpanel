package ratelimit

import (
	"testing"
	"time"
)

func TestAllowsUpToMaxThenBlocks(t *testing.T) {
	l := New(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !l.Allow("1.2.3.4") {
			t.Fatalf("attempt %d: want allowed", i)
		}
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("4th attempt within the window: want blocked")
	}
}

func TestKeysAreIndependent(t *testing.T) {
	l := New(1, time.Minute)

	if !l.Allow("a") {
		t.Fatal("first attempt for key a: want allowed")
	}
	if !l.Allow("b") {
		t.Fatal("first attempt for key b: want allowed")
	}
	if l.Allow("a") {
		t.Fatal("second attempt for key a: want blocked")
	}
}

func TestOldAttemptsExpireOutOfTheWindow(t *testing.T) {
	l := New(1, 10*time.Millisecond)

	if !l.Allow("x") {
		t.Fatal("first attempt: want allowed")
	}
	if l.Allow("x") {
		t.Fatal("second attempt within the window: want blocked")
	}

	time.Sleep(20 * time.Millisecond)

	if !l.Allow("x") {
		t.Fatal("attempt after the window expired: want allowed")
	}
}
