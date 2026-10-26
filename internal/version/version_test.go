package version

import (
	"regexp"
	"strings"
	"testing"
)

func TestStringIsVPlusNumber(t *testing.T) {
	if String() != "v"+Number {
		t.Fatalf("String() = %q, want %q", String(), "v"+Number)
	}
	if !strings.HasPrefix(String(), "v") {
		t.Fatalf("String() must carry the v prefix, got %q", String())
	}
}

// TestShortIsBareNumber pins the --short output: exactly the release number,
// no "v", no spaces, no decoration — scripts paste it into URLs and compares.
func TestShortIsBareNumber(t *testing.T) {
	s := Short()
	if s != Number {
		t.Fatalf("Short() = %q, want %q", s, Number)
	}
	if strings.Contains(s, "v") || strings.ContainsAny(s, " \t") {
		t.Fatalf("Short() must be the bare number, got %q", s)
	}
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(s) {
		t.Fatalf("Short() must be major.minor.patch, got %q", s)
	}
}
