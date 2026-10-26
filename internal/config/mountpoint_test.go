package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIsDriveLetter pins the lexical split between Windows drive-letter
// targets and macOS/Linux mountpoint paths stored in Mount.Letter.
func TestIsDriveLetter(t *testing.T) {
	cases := map[string]bool{
		"X:":       true,
		"Z:":       true,
		"z:":       false, // gd only ever stores uppercase letters
		"X":        false,
		"X:\\":     false,
		"ABC":      false,
		"/tmp/mnt": false,
		"C:/":      false,
		"":         false,
	}
	for in, want := range cases {
		if got := IsDriveLetter(in); got != want {
			t.Errorf("IsDriveLetter(%q) = %v, want %v", in, got, want)
		}
	}
}

// TestValidateMountTarget pins the readable errors for the ways a mount
// target goes bad — empty, relative, occupied — plus the healthy shapes
// (a fresh absolute dir, an existing dir, a drive letter in its real state).
func TestValidateMountTarget(t *testing.T) {
	if err := ValidateMountTarget(""); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty target must error with a hint, got %v", err)
	}
	if err := ValidateMountTarget(filepath.Join("relative", "mnt")); err == nil || !strings.Contains(err.Error(), "not an absolute path") {
		t.Fatalf("relative target must error with a hint, got %v", err)
	}

	file := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateMountTarget(file); err == nil || !strings.Contains(err.Error(), "already in use or inaccessible") {
		t.Fatalf("target occupied by a file must error as in use, got %v", err)
	}

	if err := ValidateMountTarget(filepath.Join(t.TempDir(), "acc1")); err != nil {
		t.Fatalf("fresh absolute dir must be a valid target, got %v", err)
	}
	if err := ValidateMountTarget(t.TempDir()); err != nil {
		t.Fatalf("existing empty dir must be a valid target, got %v", err)
	}

	// A drive letter's state depends on the machine; pin both sides.
	const letter = "X:"
	if _, statErr := os.Stat(letter + `\`); statErr == nil {
		err := ValidateMountTarget(letter)
		if err == nil || !strings.Contains(err.Error(), "already in use or inaccessible") {
			t.Fatalf("taken letter %s must error as in use, got %v", letter, err)
		}
	} else if err := ValidateMountTarget(letter); err != nil {
		t.Fatalf("free letter %s must be valid, got %v", letter, err)
	}
}
