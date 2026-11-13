package main

import (
	"strings"
	"testing"
	"time"
)

func TestValidatePassword(t *testing.T) {
	cases := map[string]bool{"1234567": false, "12345678": true, "รหัสผ่านไทย": true, strings.Repeat("a", 72): true, strings.Repeat("a", 73): false}
	for p, ok := range cases {
		if got := validatePassword(p) == ""; got != ok {
			t.Errorf("validatePassword(%q) ok=%v, want %v", p, got, ok)
		}
	}
}

func TestLoginLockoutAfterFiveEmailFailures(t *testing.T) {
	keys := []string{"email:lock@test.com", "ip:203.0.113.9"}
	now := time.Now()
	for i := 0; i < maxEmailFailures-1; i++ {
		recordLoginFailure(keys, now)
	}
	if lockedFor(keys, now) != 0 {
		t.Fatal("locked too early")
	}
	recordLoginFailure(keys, now)
	if wait := lockedFor(keys, now); wait <= 0 || wait > lockDuration {
		t.Fatalf("expected lock, got %v", wait)
	}
	if lockedFor(keys, now.Add(lockDuration+time.Second)) != 0 {
		t.Fatal("lock should expire")
	}
	recordLoginFailure([]string{"email:other@test.com"}, now)
	if lockedFor([]string{"email:other@test.com"}, now) != 0 {
		t.Fatal("other accounts must not be locked")
	}
}

func TestOldFailuresOutsideWindowDontCount(t *testing.T) {
	keys := []string{"email:window@test.com"}
	start := time.Now()
	for i := 0; i < maxEmailFailures-1; i++ {
		recordLoginFailure(keys, start)
	}
	later := start.Add(lockWindow + time.Minute)
	recordLoginFailure(keys, later)
	if lockedFor(keys, later) != 0 {
		t.Fatal("failures older than the window should be forgotten")
	}
}

func TestClearLoginFailures(t *testing.T) {
	keys := []string{"email:clear@test.com"}
	now := time.Now()
	for i := 0; i < maxEmailFailures-1; i++ {
		recordLoginFailure(keys, now)
	}
	clearLoginFailures(" clear@test.com ")
	recordLoginFailure([]string{"email:clear@test.com"}, now)
	if lockedFor([]string{"email:clear@test.com"}, now) != 0 {
		t.Fatal("success should reset the failure count")
	}
}

func TestTempPassword(t *testing.T) {
	p, err := tempPassword(10)
	if err != nil || len(p) != 10 || validatePassword(p) != "" {
		t.Fatalf("bad temp password %q %v", p, err)
	}
	for _, ch := range p {
		if !strings.ContainsRune(tempAlphabet, ch) {
			t.Fatalf("unexpected char %q", ch)
		}
	}
}
