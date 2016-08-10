package checkup

import (
	"crypto/tls"
	"net"
	"testing"
	"time"
)

func TestTCPChecker(t *testing.T) {
	// Listen on localhost, random port
	srv, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Errorf("Couldn't start TCP test server with error: %v", err)
	}
	defer srv.Close()

	// Accept a future connection
	go func() {
		for {
			srv.Accept()
		}
	}()

	// Should know the host:port by now
	endpt := srv.Addr().String()
	testName := "TestTCP"
	hc := TCPChecker{Name: testName, URL: endpt, Attempts: 2}

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

func TestTCPCheckerWithAgressiveTimeout(t *testing.T) {
	// Listen on localhost, random port
	srv, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Errorf("Couldn't start TCP test server with error: %v", err)
	}
	defer srv.Close()

	// Accept a future connection
	go func() {
		for {
			srv.Accept()
		}
	}()

	// Should know the host:port by now
	endpt := srv.Addr().String()
	testName := "TestTCP"
	hc := TCPChecker{Name: testName, URL: endpt, Attempts: 2, Timeout: 1 * time.Nanosecond}

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

func TestTCPCheckerWithTLSNoVerify(t *testing.T) {
	// Listen on localhost, random port
	certPair, err := tls.LoadX509KeyPair("certs/server.pem", "certs/key.pem")
	if err != nil {
		t.Error("Failed to load certificate.", err)
	}
	config := tls.Config{
		Certificates: []tls.Certificate{certPair},
	}
	srv, err := tls.Listen("tcp", "localhost:0", &config)
	if err != nil {
		t.Errorf("There was an error while starting TLS: %v", err)
	}
	defer srv.Close()

	// Accept a future connection
	go func() {
		for {
			conn, err := srv.Accept()
			if err != nil {
				return
			}
			// Keep connection open for enough time to complete test
			conn.SetDeadline(time.Now().Add(100 * time.Millisecond))
			tmp := make([]byte, 1)
			conn.Read(tmp)
		}
	}()

	// Should know the host:port by now
	endpt := srv.Addr().String()
	testName := "TestWithTLSNoVerify"
	hc := TCPChecker{Name: testName, URL: endpt, TLSEnabled: true, TLSSkipVerify: true, Attempts: 2}

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
}

func TestTCPCheckerWithTLSVerifySuccess(t *testing.T) {
	// Listen on localhost, random port
	certPair, err := tls.LoadX509KeyPair("certs/server.pem", "certs/key.pem")
	if err != nil {
		t.Error("Failed to load certificate.", err)
	}
	config := tls.Config{
		Certificates: []tls.Certificate{certPair},
	}
	srv, err := tls.Listen("tcp", "localhost:0", &config)
	if err != nil {
		t.Errorf("There was an error while starting TLS: %v", err)
	}
	defer srv.Close()

	// Accept a future connection
	go func() {
		for {
			conn, err := srv.Accept()
			if err != nil {
				return
			}
			// Keep connection open for enough time to complete test
			conn.SetDeadline(time.Now().Add(100 * time.Millisecond))
			tmp := make([]byte, 1)
			conn.Read(tmp)
		}
	}()

	// Should know the host:port by now
	endpt := srv.Addr().String()
	testName := "TestWithTLSNoVerify"
	hc := TCPChecker{Name: testName, URL: endpt, TLSEnabled: true, TLSCAfile: "certs/ca.pem", Attempts: 2}

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
}

func TestTCPCheckerWithTLSVerifyError(t *testing.T) {
	// Listen on localhost, random port
	certPair, err := tls.LoadX509KeyPair("certs/server.pem", "certs/key.pem")
	if err != nil {
		t.Error("Failed to load certificate.", err)
	}
	config := tls.Config{
		Certificates: []tls.Certificate{certPair},
	}
	srv, err := tls.Listen("tcp", "localhost:0", &config)
	if err != nil {
		t.Errorf("There was an error while starting TLS: %v", err)
	}
	defer srv.Close()

	// Accept a future connection
	go func(t *testing.T) {
		for {
			conn, err := srv.Accept()
			if err != nil {
				return
			}
			// Keep connection open for enough time to complete test
			conn.SetDeadline(time.Now().Add(100 * time.Millisecond))
			tmp := make([]byte, 1)
			conn.Read(tmp)
		}
	}(t)

	// Should know the host:port by now
	endpt := srv.Addr().String()
	testName := "TestWithTLSVerifyError"
	hc := TCPChecker{Name: testName, URL: endpt, TLSEnabled: true, Attempts: 2}

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
	if got, want := result.Down, true; got != want {
		t.Errorf("Expected result.Down=%v, got %v", want, got)
	}
	if got, want := result.Healthy, false; got != want {
		t.Errorf("Expected result.Healthy=%v, got %v", want, got)
	}
	if got, want := len(result.Times), hc.Attempts; got != want {
		t.Errorf("Expected %d attempts, got %d", want, got)
	}
	ts := time.Unix(0, result.Timestamp)
	if time.Since(ts) > 5*time.Second {
		t.Errorf("Expected timestamp to be recent, got %s", ts)
	}
}
