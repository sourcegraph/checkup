package checkup

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPChecker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "I'm up")
	}))
	endpt := "http://" + srv.Listener.Addr().String()
	hc := HTTPChecker{Name: "Test", URL: endpt, Attempts: 2}

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
	hc.MaxRTT = 1 * time.Nanosecond
	result, err = hc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Down, true; got != want {
		t.Errorf("Expected result.Down=%v, got %v", want, got)
	}

	hc.MaxRTT = 0
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
