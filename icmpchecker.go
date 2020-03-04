package checkup

import (
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"github.com/tatsushid/go-fastping"
)

// ICMPChecker implements a Checker for TCP endpoints.
type ICMPChecker struct {
	// Name is the name of the endpoint.
	Name string `json:"endpoint_name"`

	// URL is the URL of the endpoint.
	URL string `json:"endpoint_url"`

	// Timeout is the maximum time to wait for a
	// TCP connection to be established.
	Timeout time.Duration `json:"timeout,omitempty"`

	// ThresholdRTT is the maximum round trip time to
	// allow for a healthy endpoint. If non-zero and a
	// request takes longer than ThresholdRTT, the
	// endpoint will be considered unhealthy. Note that
	// this duration includes any in-between network
	// latency.
	ThresholdRTT time.Duration `json:"threshold_rtt,omitempty"`

	// Attempts is how many requests the client will
	// make to the endpoint in a single check.
	Attempts int `json:"attempts,omitempty"`
}

// Check performs checks using c according to its configuration.
// An error is only returned if there is a configuration error.
func (c ICMPChecker) Check() (Result, error) {
	if c.Attempts < 1 {
		c.Attempts = 1
	}

	result := Result{Title: c.Name, Endpoint: c.URL, Timestamp: Timestamp()}
	result.Times = c.doChecks()

	return c.conclude(result), nil
}

// doChecks executes and returns each attempt.
func (c ICMPChecker) doChecks() Attempts {
	var err error

	var pinger := fastping.NewPinger()

	timeout := c.Timeout
	if timeout == 0 {
		timeout = 1 * time.Second
	}

	checks := make(Attempts, c.Attempts)
	for i := 0; i < c.Attempts; i++ {
		start := time.Now()

		// can it resolve at all?
		var ra, resolveErr := net.ResolveIPAddr("ip4:icmp", c.URL)

		if resolveErr != nil {
			checks[i].Error = resolveErr.Error()
			checks[i].RTT = time.Since(start)
			continue
		}

		pinger.AddIPAddr(ra)
		p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
			// fmt.Printf("IP Addr: %s receive, RTT: %v\n", addr.String(), rtt)
		}
		p.OnIdle = func() {
			// fmt.Println("finish")
		}

		err = p.Run()

		checks[i].RTT = time.Since(start)

		if err != nil {
			checks[i].Error = err.Error()
			continue
		}
	}
	return checks
}

// conclude takes the data in result from the attempts and
// computes remaining values needed to fill out the result.
// It detects degraded (high-latency) responses and makes
// the conclusion about the result's status.
func (c ICMPChecker) conclude(result Result) Result {
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
