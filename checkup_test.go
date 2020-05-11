package checkup

import (
	"bytes"
	"errors"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/sourcegraph/checkup/types"
)

func TestCheckAndStore(t *testing.T) {
	f := new(fake)
	c := Checkup{
		Storage:          f,
		Checkers:         []Checker{f, f},
		ConcurrentChecks: 1,
		Timestamp:        time.Now(),
		Notifiers:        []Notifier{f, f},
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
	if got, want := f.notified, 2; got != want {
		t.Errorf("Expected Notify() to be called %d time, called %d times", want, got)
	}
	if got, want := f.maintained, 1; got != want {
		t.Errorf("Expected Maintain() to be called %d time, called %d times", want, got)
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

	c.ConcurrentChecks = -1
	_, err = c.Check()
	if err == nil {
		t.Error("Expected an error with ConcurrentChecks < 0, didn't get one")
	}
	c.ConcurrentChecks = 0
	c.Storage = nil
	err = c.CheckAndStore()
	if err == nil {
		t.Error("Expected an error with no storage, didn't get one")
	}
}

func TestCheckAndStoreEvery(t *testing.T) {
	f := new(fake)
	c := Checkup{Storage: f, Checkers: []Checker{f}}

	ticker := c.CheckAndStoreEvery(50 * time.Millisecond)
	time.Sleep(170 * time.Millisecond)
	ticker.Stop()

	f.Lock()
	defer f.Unlock()
	if got, want := f.checked, 3; got != want {
		t.Errorf("Expected %d checks while sleeping, had: %d", want, got)
	}
}

func TestComputeStats(t *testing.T) {
	s := types.Result{Times: []types.Attempt{
		{RTT: 7 * time.Second},
		{RTT: 4 * time.Second},
		{RTT: 4 * time.Second},
		{RTT: 6 * time.Second},
		{RTT: 6 * time.Second},
		{RTT: 3 * time.Second},
	}}.ComputeStats()

	if got, want := s.Total, 30*time.Second; got != want {
		t.Errorf("Expected Total=%v, got %v", want, got)
	}
	if got, want := s.Mean, 5*time.Second; got != want {
		t.Errorf("Expected Mean=%v, got %v", want, got)
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

func TestResultStatus(t *testing.T) {
	r := types.Result{Healthy: true}
	if got, want := r.Status(), types.StatusHealthy; got != want {
		t.Errorf("Expected status '%s' but got: '%s'", want, got)
	}

	r = types.Result{Degraded: true}
	if got, want := r.Status(), types.StatusDegraded; got != want {
		t.Errorf("Expected status '%s' but got: '%s'", want, got)
	}

	r = types.Result{Down: true}
	if got, want := r.Status(), types.StatusDown; got != want {
		t.Errorf("Expected status '%s' but got: '%s'", want, got)
	}

	r = types.Result{}
	if got, want := r.Status(), types.StatusUnknown; got != want {
		t.Errorf("Expected status '%s' but got: '%s'", want, got)
	}

	// These are invalid states, but we need to test anyway in case a
	// checker is buggy. We expect the worst of the enabled fields.
	r = types.Result{Down: true, Degraded: true}
	if got, want := r.Status(), types.StatusDown; got != want {
		t.Errorf("(INVALID RESULT CASE) Expected status '%s' but got: '%s'", want, got)
	}
	r = types.Result{Degraded: true, Healthy: true}
	if got, want := r.Status(), types.StatusDegraded; got != want {
		t.Errorf("(INVALID RESULT CASE) Expected status '%s' but got: '%s'", want, got)
	}
	r = types.Result{Down: true, Healthy: true}
	if got, want := r.Status(), types.StatusDown; got != want {
		t.Errorf("(INVALID RESULT CASE) Expected status '%s' but got: '%s'", want, got)
	}
}

func TestPriorityOver(t *testing.T) {
	for i, test := range []struct {
		status   types.StatusText
		another  types.StatusText
		expected bool
	}{
		{types.StatusDown, types.StatusDown, false},
		{types.StatusDown, types.StatusDegraded, true},
		{types.StatusDown, types.StatusHealthy, true},
		{types.StatusDown, types.StatusUnknown, true},
		{types.StatusDegraded, types.StatusDown, false},
		{types.StatusDegraded, types.StatusDegraded, false},
		{types.StatusDegraded, types.StatusHealthy, true},
		{types.StatusDegraded, types.StatusUnknown, true},
		{types.StatusHealthy, types.StatusDown, false},
		{types.StatusHealthy, types.StatusDegraded, false},
		{types.StatusHealthy, types.StatusHealthy, false},
		{types.StatusHealthy, types.StatusUnknown, true},
		{types.StatusUnknown, types.StatusDown, false},
		{types.StatusUnknown, types.StatusDegraded, false},
		{types.StatusUnknown, types.StatusHealthy, false},
		{types.StatusUnknown, types.StatusUnknown, false},
	} {
		actual := test.status.PriorityOver(test.another)
		if actual != test.expected {
			t.Errorf("Test %d: Expected %s.PriorityOver(%s)=%v, but got %v",
				i, test.status, test.another, test.expected, actual)
		}
	}
}

func TestJSON(t *testing.T) {
	var (
		checkup    = new(Checkup)
		testConfig = "testdata/config.json"
	)

	jsonBytes, err := ioutil.ReadFile(testConfig)
	if err != nil {
		t.Fatalf("Error reading config file: %s", testConfig)
	}

	err = checkup.UnmarshalJSON(jsonBytes)
	if err != nil {
		t.Fatalf("Error unmarshaling: %v", err)
	}

	result, err := checkup.MarshalJSON()
	if err != nil {
		t.Fatalf("Error marshaling: %v", err)
	}

	if !bytes.Equal(result, jsonBytes) {
		t.Errorf("\nGot:  %s\nWant: %s", string(result), string(jsonBytes))
	}
}

var errTest = errors.New("i'm an error")

type fake struct {
	sync.Mutex

	returnErr  bool
	checked    int
	stored     []types.Result
	maintained int
	notified   int
}

func (f *fake) Type() string {
	return "fake"
}

func (f *fake) Check() (types.Result, error) {
	f.Lock()
	defer f.Unlock()

	f.checked++
	r := types.Result{Timestamp: time.Now().UTC().UnixNano()}
	if f.returnErr {
		return r, errTest
	}
	return r, nil
}

func (f *fake) Store(results []types.Result) error {
	f.Lock()
	defer f.Unlock()

	f.stored = results
	if f.returnErr {
		return errTest
	}
	return nil
}

func (f *fake) Maintain() error {
	f.Lock()
	defer f.Unlock()

	f.maintained++
	return nil
}

func (f *fake) Notify(results []types.Result) error {
	f.Lock()
	defer f.Unlock()

	f.notified++
	return nil
}
