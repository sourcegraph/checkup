package checkup

import (
	"strings"
	"testing"
)

func TestUnknownNotifierType(t *testing.T) {
	kind, err := notifierType("")
	if got, want := kind, ""; got != want {
		t.Errorf("Expected type '%s', got '%s'", want, got)
	}
	want := strings.Replace(errUnknownNotifierType, "%T", "string", -1)
	if got := err.Error(); got != want {
		t.Errorf("Expected error '%s', got '%s'", want, got)
	}
}
