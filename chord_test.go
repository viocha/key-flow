package main

import "testing"

func TestCanonicalChordOrdersModifiers(t *testing.T) {
	got, err := canonicalChord([]string{"Shift", "K", "Ctrl", "Win", "Alt"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Win+Ctrl+Alt+Shift+K" {
		t.Fatalf("canonicalChord() = %q, want %q", got, "Win+Ctrl+Alt+Shift+K")
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

func TestMouseShortcutKeyOrdersModifiers(t *testing.T) {
	got, err := mouseShortcutKey([]string{"Alt", "Shift", "controlleft", "Win"}, "left")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Win+Ctrl+Alt+Shift+left" {
		t.Fatalf("mouseShortcutKey() = %q, want %q", got, "Win+Ctrl+Alt+Shift+left")
	}
}

func TestMouseShortcutKeyRejectsNonModifier(t *testing.T) {
	if _, err := mouseShortcutKey([]string{"Shift", "K"}, "left"); err == nil {
		t.Fatal("expected non-modifier key to be rejected")
	}
}
