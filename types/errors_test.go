package types

import (
	"errors"
	"testing"
)

func TestErrors(t *testing.T) {
	errs := []error{
		errors.New("Err 1"),
		errors.New("Err 2"),
	}
	errsT := Errors(errs)

	want := "Err 1; Err 2"
	if got := errsT.Error(); want != got {
		t.Errorf("Errors, wanted '%s', got '%s'", want, got)
	}
}
