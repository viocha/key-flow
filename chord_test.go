package main

import "testing"

func TestCanonicalChordOrdersModifiers(t *testing.T) {
	got, err := canonicalChord([]string{"Shift", "K", "Ctrl"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Ctrl+Shift+K" {
		t.Fatalf("canonicalChord() = %q, want %q", got, "Ctrl+Shift+K")
	}
}

func TestCanonicalChordNormalizesAliases(t *testing.T) {
	got, err := canonicalChord([]string{"controlleft", "left"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Ctrl+ArrowLeft" {
		t.Fatalf("canonicalChord() = %q, want %q", got, "Ctrl+ArrowLeft")
	}
}

func TestCanonicalChordRejectsMultipleMainKeys(t *testing.T) {
	if _, err := canonicalChord([]string{"Ctrl", "K", "L"}); err == nil {
		t.Fatal("expected multiple main keys to be rejected")
	}
}

func TestCanonicalChordRejectsUnsupportedKey(t *testing.T) {
	if _, err := canonicalChord([]string{"Ctrl", "MouseLeft"}); err == nil {
		t.Fatal("expected unsupported key to be rejected")
	}
}
