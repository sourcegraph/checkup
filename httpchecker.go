package checkup

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

// HTTPChecker implements a Checker for HTTP endpoints.
type HTTPChecker struct {
	// Name is the name of the endpoint.
	Name string `json:"endpoint_name"`

	// URL is the URL of the endpoint.
	URL string `json:"endpoint_url"`

	// UpStatus is the HTTP status code expected by
	// a healthy endpoint. Default is http.StatusOK.
	UpStatus int `json:"up_status,omitempty"`

	// ThresholdRTT is the maximum round trip time to
	// allow for a healthy endpoint. If non-zero and a
	// request takes longer than ThresholdRTT, the
	// endpoint will be considered unhealthy. Note that
	// this duration includes any in-between network
	// latency.
	ThresholdRTT time.Duration `json:"threshold_rtt,omitempty"`

	// MustContain is a string that the response body
	// must contain in order to be considered up.
	// NOTE: If set, the entire response body will
	// be consumed, which has the potential of using
	// lots of memory and slowing down checks if the
	// response body is large.
	MustContain string `json:"must_contain,omitempty"`

	// MustNotContain is a string that the response
	// body must NOT contain in order to be considered
	// up. If both MustContain and MustNotContain are
	// set, they are and-ed together. NOTE: If set,
	// the entire response body will be consumed, which
	// has the potential of using lots of memory and
	// slowing down checks if the response body is large.
	MustNotContain string `json:"must_not_contain,omitempty"`

	// Attempts is how many requests the client will
	// make to the endpoint in a single check.
	Attempts int `json:"attempts,omitempty"`

	// AttemptSpacing spaces out each attempt in a check
	// by this duration to avoid hitting a remote too
	// quickly in succession. By default, no waiting
	// occurs between attempts.
	AttemptSpacing time.Duration `json:"attempt_spacing,omitempty"`

	// Client is the http.Client with which to make
	// requests. If not set, DefaultHTTPClient is
	// used.
	Client *http.Client `json:"-"`

	// Headers contains headers to added to the request
	// that is sent for the check
	Headers http.Header `json:"headers,omitempty"`
}

// Check performs checks using c according to its configuration.
// An error is only returned if there is a configuration error.
func (c HTTPChecker) Check() (Result, error) {
	if c.Attempts < 1 {
		c.Attempts = 1
	}
	if c.Client == nil {
		c.Client = DefaultHTTPClient
	}
	if c.UpStatus == 0 {
		c.UpStatus = http.StatusOK
	}

	result := Result{Title: c.Name, Endpoint: c.URL, Timestamp: Timestamp()}
	req, err := http.NewRequest("GET", c.URL, nil)
	if err != nil {
		return result, err
	}

	if c.Headers != nil {
		for key, header := range c.Headers {
			if strings.EqualFold(key, "host") {
				req.Host = header[0]
			} else {
				req.Header.Add(key, strings.Join(header, ", "))
			}
		}
	}

	result.Times = c.doChecks(req)

	return c.conclude(result), nil
}

// doChecks executes req using c.Client and returns each attempt.
func (c HTTPChecker) doChecks(req *http.Request) Attempts {
	checks := make(Attempts, c.Attempts)
	for i := 0; i < c.Attempts; i++ {
		start := time.Now()
		resp, err := c.Client.Do(req)
		checks[i].RTT = time.Since(start)
		if err != nil {
			checks[i].Error = err.Error()
			continue
		}
		err = c.checkDown(resp)
		if err != nil {
			checks[i].Error = err.Error()
		}
		resp.Body.Close()
		if c.AttemptSpacing > 0 {
			time.Sleep(c.AttemptSpacing)
		}
	}
	return checks
}

// conclude takes the data in result from the attempts and
// computes remaining values needed to fill out the result.
// It detects degraded (high-latency) responses and makes
// the conclusion about the result's status.
func (c HTTPChecker) conclude(result Result) Result {
	result.ThresholdRTT = c.ThresholdRTT

	// Check errors (down)
	for i := range result.Times {
		if result.Times[i].Error != "" {
			result.Down = true
			return result
		}
	}

	// Check round trip time (degraded)
	if c.ThresholdRTT > 0 {
		stats := result.ComputeStats()
		if stats.Median > c.ThresholdRTT {
			result.Notice = fmt.Sprintf("median round trip time exceeded threshold (%s)", c.ThresholdRTT)
			result.Degraded = true
			return result
		}
	}

	result.Healthy = true
	return result
}

// checkDown checks whether the endpoint is down based on resp and
// the configuration of c. It returns a non-nil error if down.
// Note that it does not check for degraded response.
func (c HTTPChecker) checkDown(resp *http.Response) error {
	// Check status code
	if resp.StatusCode != c.UpStatus {
		return fmt.Errorf("response status %s", resp.Status)
	}

	// Check response body
	if c.MustContain == "" && c.MustNotContain == "" {
		return nil
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}
	body := string(bodyBytes)
	if c.MustContain != "" && !strings.Contains(body, c.MustContain) {
		return fmt.Errorf("response does not contain '%s'", c.MustContain)
	}
	if c.MustNotContain != "" && strings.Contains(body, c.MustNotContain) {
		return fmt.Errorf("response contains '%s'", c.MustNotContain)
	}

	return nil
}

// DefaultHTTPClient is used when no other http.Client
// is specified on a HTTPChecker.
var DefaultHTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 0,
		}).Dial,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   1,
		DisableCompression:    true,
		DisableKeepAlives:     true,
		ResponseHeaderTimeout: 5 * time.Second,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Timeout: 10 * time.Second,
}
