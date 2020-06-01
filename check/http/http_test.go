package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestChecker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Checkup", r.Header.Get("X-Checkup"))
		fmt.Fprintln(w, "I'm up", "@"+r.Host)
	}))
	endpt := "http://" + srv.Listener.Addr().String()
	hc := Checker{Name: "Test", URL: endpt, Attempts: 2}

	// Try an up server
	result, err := hc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Title, "Test"; got != want {
		t.Errorf("Expected result.Title='%s', got '%s'", want, got)
	}
	if got, want := result.Endpoint, endpt; got != want {
		t.Errorf("Expected result.Endpoint='%s', got '%s'", want, got)
	}
	if got, want := result.Down, false; got != want {
		t.Errorf("Expected result.Down=%v, got %v", want, got)
	}
	if got, want := result.Degraded, false; got != want {
		t.Errorf("Expected result.Degraded=%v, got %v", want, got)
	}
	if got, want := result.Healthy, true; got != want {
		t.Errorf("Expected result.Healthy=%v, got %v", want, got)
	}
	if got, want := len(result.Times), hc.Attempts; got != want {
		t.Errorf("Expected %d attempts, got %d", want, got)
	}
	ts := time.Unix(0, result.Timestamp)
	if time.Since(ts) > 5*time.Second {
		t.Errorf("Expected timestamp to be recent, got %s", ts)
	}

	// Try various different down criteria

	hc.UpStatus = 201
	result, err = hc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Down, true; got != want {
		t.Errorf("Expected result.Down=%v, got %v", want, got)
	}

	hc.UpStatus = 200
	hc.ThresholdRTT = 1 * time.Nanosecond
	result, err = hc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Degraded, true; got != want {
		t.Errorf("Expected result.Degraded=%v, got %v", want, got)
	}

	hc.ThresholdRTT = 0
	hc.MustContain = "up"
	result, err = hc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Down, false; got != want {
		t.Errorf("Expected result.Down=%v, got %v", want, got)
	}

	hc.MustContain = "online"
	result, err = hc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Down, true; got != want {
		t.Errorf("Expected result.Down=%v, got %v", want, got)
	}

	hc.MustContain = ""
	hc.MustNotContain = "down"
	result, err = hc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Down, false; got != want {
		t.Errorf("Expected result.Down=%v, got %v", want, got)
	}

	hc.MustNotContain = "I"
	result, err = hc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Down, true; got != want {
		t.Errorf("Expected result.Down=%v, got %v", want, got)
	}

	// Test with a Header
	hc.Headers = http.Header{
		"X-Checkup": []string{"Echo"},
	}
	hc.MustNotContain = ""
	hc.MustContain = "Echo"
	hc.ThresholdRTT = 0
	result, err = hc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Down, true; got != want {
		t.Errorf("Expected result.Down=%v, got %v", want, got)
	}

	// Test with a Host header
	hc.Headers = http.Header{
		"Host": []string{"http.check.local"},
	}
	hc.MustContain = "@http.check.local"
	hc.MustNotContain = ""
	hc.ThresholdRTT = 0
	result, err = hc.Check()

	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Down, false; got != want {
		t.Errorf("Expected result.Down=%v, got %v", want, got)
	}

	// Try when the server is not even online
	srv.Listener.Close()
	result, err = hc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := len(result.Times), hc.Attempts; got != want {
		t.Errorf("Expected %d attempts, got %d", want, got)
	}
	if got, want := result.Down, true; got != want {
		t.Errorf("Expected result.Down=%v, got %v", want, got)
	}
}
