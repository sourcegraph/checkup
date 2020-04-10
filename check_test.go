package checkup

import (
	"strings"
	"testing"
)

func TestUnknownCheckerType(t *testing.T) {
	kind, err := checkerType("")
	if got, want := kind, ""; got != want {
		t.Errorf("Expected type '%s', got '%s'", want, got)
	}
	want := strings.Replace(errUnknownCheckerType, "%T", "string", -1)
	if got := err.Error(); got != want {
		t.Errorf("Expected error '%s', got '%s'", want, got)
	}
}
