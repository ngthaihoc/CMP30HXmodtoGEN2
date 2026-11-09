package main

import (
	"strings"
	"testing"
)

func TestCMP30HXAdviceNeverSuggestsHardReset(t *testing.T) {
	for _, state := range []struct {
		name            string
		ss0OK, unlocked bool
		speed, tls      uint32
	}{
		{"TLS Gen1", false, false, 1, 1},
		{"TLS Gen2", false, false, 1, 2},
	} {
		t.Run(state.name, func(t *testing.T) {
			advice := linkAdvice(true, state.ss0OK, state.unlocked, state.speed, state.tls)
			if !strings.Contains(advice, "Không reset thiết bị") {
				t.Fatalf("CMP 30HX needs actionable Gen1/Gen2 guidance: %q", advice)
			}
			text := knownIssuesBlock(true) + advice
			if strings.Contains(text, "-hard") || strings.Contains(text, "Stage2") || strings.Contains(text, "Root Link Disable") || strings.Contains(text, "EFI không tìm thấy card") || strings.Contains(text, "Gen3") {
				t.Fatalf("CMP 30HX diagnostics suggest a dangerous reset: %s", text)
			}
		})
	}
}

func TestCMP30HXTaskAdviceDoesNotRecommendCardReset(t *testing.T) {
	text := missingTaskAdvice(true, true, 2, 2)
	if strings.Contains(text, "reset card") || strings.Contains(text, "EFI") || !strings.Contains(text, "CMP 30HX") {
		t.Fatalf("incorrect CMP 30HX task advice: %q", text)
	}
}

func TestKnownIssuesBlockKeepsCMP30HXAwayFromHardReset(t *testing.T) {
	text := knownIssuesBlock(true)
	if strings.Contains(text, "-hard") || strings.Contains(text, "Stage2") || strings.Contains(text, "Root Link Disable") || strings.Contains(text, "EFI không tìm thấy card") || strings.Contains(text, "Gen3") {
		t.Fatalf("CMP 30HX diagnostics suggest a dangerous reset: %s", text)
	}
	if !strings.Contains(text, "TLS") || !strings.Contains(text, "Gen2") {
		t.Fatalf("CMP 30HX diagnostics lost link evidence: %s", text)
	}
}
