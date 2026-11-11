package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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

func TestCheckerAllowedFailures(t *testing.T) {
	var requestCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	}))
	defer srv.Close()

	// 1. Default behavior (AllowedFailures: 0) marks down on single failure.
	hcDefault := Checker{
		Name:            "Default Failures",
		URL:             srv.URL,
		Attempts:        3,
		AllowedFailures: 0,
	}
	atomic.StoreInt32(&requestCount, 0)
	result, err := hcDefault.Check()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Down {
		t.Errorf("expected result.Down=true with default AllowedFailures=0 and 1 failure, got %v", result.Down)
	}
	if result.Healthy {
		t.Errorf("expected result.Healthy=false, got %v", result.Healthy)
	}

	// 2. AllowedFailures: 1 permits 1 failed attempt out of N attempts without marking down.
	hcAllowed := Checker{
		Name:            "Allowed 1 Failure",
		URL:             srv.URL,
		Attempts:        3,
		AllowedFailures: 1,
	}
	atomic.StoreInt32(&requestCount, 0)
	result, err = hcAllowed.Check()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Down {
		t.Errorf("expected result.Down=false with AllowedFailures=1 and 1 failure, got %v", result.Down)
	}
	if !result.Healthy {
		t.Errorf("expected result.Healthy=true with AllowedFailures=1 and 1 failure, got %v", result.Healthy)
	}

	// 3. Errors exceeding AllowedFailures properly mark down.
	var multiFailCount int32
	srvMultiFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&multiFailCount, 1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	}))
	defer srvMultiFail.Close()

	hcExceeded := Checker{
		Name:            "Exceeded Failures",
		URL:             srvMultiFail.URL,
		Attempts:        3,
		AllowedFailures: 1,
	}
	result, err = hcExceeded.Check()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Down {
		t.Errorf("expected result.Down=true when errors exceed AllowedFailures, got %v", result.Down)
	}
	if result.Healthy {
		t.Errorf("expected result.Healthy=false when errors exceed AllowedFailures, got %v", result.Healthy)
	}
}

func TestCheckerUnmarshalAllowedFailures(t *testing.T) {
	raw := []byte(`{"endpoint_name":"test","endpoint_url":"http://localhost","allowed_failures":2,"attempts":5}`)
	c, err := New(raw)
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if c.AllowedFailures != 2 {
		t.Errorf("expected AllowedFailures=2, got %d", c.AllowedFailures)
	}
}
