package dns

import (
	"net"
	"testing"
	"time"
)

func TestChecker(t *testing.T) {
	// Listen on localhost, random port
	srv, err := net.Listen("tcp", "localhost:8382")
	if err != nil {
		t.Errorf("Couldn't start TCP test server with error: %v", err)
	}
	defer srv.Close()

	// Accept a future connection
	go func() {
		for {
			conn, err := srv.Accept()
			if err != nil {
				break
			}
			conn.Close()
		}
	}()

	// Should know the host:port by now
	endpt := srv.Addr().String()
	testName := "TestDNS"
	hc := Checker{Name: testName, URL: endpt, Attempts: 2}

	// Try an up server
	result, err := hc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Title, testName; got != want {
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
	result, err = hc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Healthy, true; got != want {
		t.Errorf("Expected result.Healthy=%v, got %v", want, got)
	}

	hc.ThresholdRTT = 1 * time.Nanosecond
	result, err = hc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Degraded, true; got != want {
		t.Errorf("Expected result.Degraded=%v, got %v", want, got)
	}

	hc.ThresholdRTT = 0
	result, err = hc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Down, false; got != want {
		t.Errorf("Expected result.Down=%v, got %v", want, got)
	}

	// Try when the server is not even online
	srv.Close()
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
	if got, want := result.Healthy, false; got != want {
		t.Errorf("Expected result.Healthy=%v, got %v", want, got)
	}
}

func TestCheckerWithAgressiveTimeout(t *testing.T) {
	// Listen on localhost, random port
	srv, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Errorf("Couldn't start TCP test server with error: %v", err)
	}
	defer srv.Close()

	// Accept a future connection
	go func() {
		for {
			conn, err := srv.Accept()
			if err != nil {
				break
			}
			conn.Close()
		}
	}()

	// Should know the host:port by now
	endpt := srv.Addr().String()
	testName := "TestTCP"
	hc := Checker{Name: testName, URL: endpt, Attempts: 2, Timeout: 1 * time.Nanosecond}

	result, err := hc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := len(result.Times), hc.Attempts; got != want {
		t.Errorf("Expected %d attempts, got %d", want, got)
	}
	if got, want := result.Down, true; got != want {
		t.Errorf("Expected result.Down=%v, got %v", want, got)
	}
	if got, want := result.Healthy, false; got != want {
		t.Errorf("Expected result.Healthy=%v, got %v", want, got)
	}
}
