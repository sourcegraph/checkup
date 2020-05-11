package types

import (
	"errors"
	"testing"
)

var (
	err1 = errors.New("err 1")
	err2 = errors.New("err 2")
)

func TestErrors(t *testing.T) {
	errs := []error{
		err1,
		err2,
	}
	errsT := Errors(errs)

	want := "err 1; err 2"
	if got := errsT.Error(); want != got {
		t.Errorf("Errors, wanted '%s', got '%s'", want, got)
	}
}
