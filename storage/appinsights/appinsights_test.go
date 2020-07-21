package appinsights

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/sourcegraph/checkup/types"
)

const instrumentationKey = "11111111-1111-1111-1111-111111111111"

var results = []types.Result{{
	Title:        "Testing",
	Endpoint:     "http://www.example.com",
	ThresholdRTT: 100000000,
	Timestamp:    time.Now().UnixNano(),
	Healthy:      true,
	Times: []types.Attempt{{
		RTT: 40 * time.Millisecond,
	}},
}}

func TestNew(t *testing.T) {
	type test struct {
		retries  int
		interval int
		timeout  int
		wantErr  bool
	}
	tests := []test{
		{retries: -1, interval: 0, timeout: 0, wantErr: true},
		{retries: 0, interval: -1, timeout: 0, wantErr: true},
		{retries: 0, interval: 0, timeout: -1, wantErr: true},
		{retries: 0, interval: 0, timeout: 0, wantErr: false},
		{retries: 1, interval: 0, timeout: 0, wantErr: false},
		{retries: 0, interval: 1, timeout: 0, wantErr: false},
		{retries: 0, interval: 0, timeout: 1, wantErr: false},
	}

	for _, tc := range tests {
		config, _ := setup(0, tc.retries, tc.interval, tc.timeout, results)
		c, _ := json.Marshal(config)
		msg := fmt.Sprintf("Expected error from New() to be %v, got: %v", tc.wantErr, !tc.wantErr)

		_, err := New(c)
		if tc.wantErr {
			if err == nil {
				t.Fatalf(msg)
			}
		} else {
			if err != nil {
				t.Fatalf(msg)
			}
		}
	}
}
func TestStoreNoRetry(t *testing.T) {
	type test struct {
		retries  int
		interval int
		timeout  int
	}
	tests := []test{
		{retries: 0, interval: 0, timeout: 0},
		{retries: 1, interval: 0, timeout: 0},
		{retries: 0, interval: 1, timeout: 0},
		{retries: 0, interval: 0, timeout: 1},
	}

	for _, tc := range tests {
		config, server := setup(0, tc.retries, tc.interval, tc.timeout, results)
		defer server.Close()
		c, _ := json.Marshal(config)

		specimen, err := New(c)
		if err != nil {
			t.Fatalf("Expected no error from New(), got: %v", err)
		}

		if err := specimen.Store(results); err != nil {
			t.Fatalf("Expected no error from Store(), got: %v", err)
		}
	}
}

func TestStoreWithRetry(t *testing.T) {
	config, server := setup(2, 1, 1, 0, results)
	defer server.Close()
	c, _ := json.Marshal(config)

	specimen, err := New(c)
	if err != nil {
		t.Fatalf("Expected no error from New(), got: %v", err)
	}

	if err := specimen.Store(results); err != nil {
		t.Fatalf("Expected no error from Store() with retry, got: %v", err)
	}
}

func setup(delay time.Duration, retries int, interval int, timeout int, results []types.Result) (Storage, *httptest.Server) {
	forceRetry := false
	if interval > 0 && retries > 0 {
		forceRetry = true
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if forceRetry {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(""))
			forceRetry = false
			return
		}

		var req *gzip.Reader
		req, err := gzip.NewReader(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("gzip NewReader: %v", err)))
		}
		b, _ := ioutil.ReadAll(req)
		parsed, err := parsePayload(b)
		for i, j := range parsed {
			data := j["data"].(map[string]interface{})
			baseData := data["baseData"].(map[string]interface{})

			got, ok := baseData["name"].(string)
			if !ok || got != results[i].Title {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("Expected test result name to be '%s', but got '%s'", results[i].Title, got)))
			}
		}
	}))

	telemetryConfig := appinsights.NewTelemetryConfiguration(instrumentationKey)
	telemetryConfig.EndpointUrl = server.URL

	config := Storage{
		InstrumentationKey: instrumentationKey,
		TestLocation:       "test location",
		Tags: map[string]string{
			"tag1": "test tag",
		},
		telemetryConfig: telemetryConfig,
		Timeout:         timeout,
	}
	if interval > -1 || retries > -1 {
		config.MaxRetries = retries
		config.RetryInterval = interval
	}

	return config, server
}

// Ref: https://github.com/microsoft/ApplicationInsights-Go/blob/master/appinsights/jsonserializer_test.go
func parsePayload(payload []byte) (result []map[string]interface{}, err error) {
	for _, item := range bytes.Split(payload, []byte("\n")) {
		if len(item) == 0 {
			continue
		}

		decoder := json.NewDecoder(bytes.NewReader(item))
		msg := make(map[string]interface{})
		if err := decoder.Decode(&msg); err == nil {
			result = append(result, msg)
		} else {
			return result, err
		}
	}

	return result, nil
}
