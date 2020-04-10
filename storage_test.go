package checkup

import (
	"strings"
	"testing"
)

func TestUnknownStorageType(t *testing.T) {
	kind, err := storageType("")
	if got, want := kind, ""; got != want {
		t.Errorf("Expected type '%s', got '%s'", want, got)
	}
	want := strings.Replace(errUnknownStorageType, "%T", "string", -1)
	if got := err.Error(); got != want {
		t.Errorf("Expected error '%s', got '%s'", want, got)
	}
}
