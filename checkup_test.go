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
		Timestamp:        time.Now(),
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
	for i := range f.stored {
		if i > 0 && f.stored[i].Timestamp != f.stored[i-1].Timestamp {
			t.Error("Expected timestamps to be the same, but they weren't")
		}
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

func TestComputeStats(t *testing.T) {
	s := Result{Times: []Attempt{
		{RTT: 3 * time.Second},
		{RTT: 4 * time.Second},
		{RTT: 4 * time.Second},
		{RTT: 6 * time.Second},
		{RTT: 6 * time.Second},
		{RTT: 7 * time.Second},
	}}.ComputeStats()

	if got, want := s.Total, 30*time.Second; got != want {
		t.Errorf("Expected Total=%v, got %v", want, got)
	}
	if got, want := s.Average, 5*time.Second; got != want {
		t.Errorf("Expected Average=%v, got %v", want, got)
	}
	if got, want := s.Median, 5*time.Second; got != want {
		t.Errorf("Expected Median=%v, got %v", want, got)
	}
	if got, want := s.Min, 3*time.Second; got != want {
		t.Errorf("Expected Min=%v, got %v", want, got)
	}
	if got, want := s.Max, 7*time.Second; got != want {
		t.Errorf("Expected Max=%v, got %v", want, got)
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
	r := Result{Timestamp: time.Now().UTC().UnixNano()}
	if f.returnErr {
		return r, errTest
	}
	return r, nil
}

func (f *fake) Store(results []Result) error {
	f.stored = results
	if f.returnErr {
		return errTest
	}
	return nil
}
