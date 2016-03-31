package checkup

import (
	"errors"
	"testing"
	"time"
)

func TestCheckAndStore(t *testing.T) {
	f := new(fake)
	c := Checkup{
		Storage:          f,
		Checkers:         []Checker{f, f},
		ConcurrentChecks: 1,
	}

	err := c.CheckAndStore()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := f.checked, 2; got != want {
		t.Errorf("Expected %d checks to be executed, but had: %d", want, got)
	}
	if got, want := len(f.stored), 2; got != want {
		t.Errorf("Expected %d checks to be stored, but had: %d", want, got)
	}

	// Check error handling
	f.returnErr = true
	err = c.CheckAndStore()
	if err == nil {
		t.Error("Expected an error, didn't get one")
	}
	if got, want := err.Error(), "i'm an error; i'm an error"; got != want {
		t.Errorf(`Expected error string "%s" but got: "%s"`, want, got)
	}
}

func TestCheckAndStoreEvery(t *testing.T) {
	f := new(fake)
	c := Checkup{Storage: f, Checkers: []Checker{f}}

	ticker := c.CheckAndStoreEvery(50 * time.Millisecond)
	time.Sleep(170 * time.Millisecond)
	ticker.Stop()

	if got, want := f.checked, 3; got != want {
		t.Errorf("Expected %d checks while sleeping, had: %d", want, got)
	}
}

var errTest = errors.New("i'm an error")

type fake struct {
	returnErr bool
	checked   int
	stored    []Result
}

func (f *fake) Check() (Result, error) {
	f.checked++
	if f.returnErr {
		return Result{}, errTest
	}
	return Result{}, nil
}

func (f *fake) Store(results []Result) error {
	f.stored = results
	if f.returnErr {
		return errTest
	}
	return nil
}
