package tls

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"github.com/sourcegraph/checkup/types"
)

// Type should match the package name
const Type = "tls"

// Checker implements a Checker for TLS endpoints.
//
// TODO: Implement more checks on the certificate and TLS configuration.
//  - Cipher suites
//  - Protocol versions
//  - OCSP stapling
//  - Multiple SNIs
//  - Other things that you might see at SSL Labs or other TLS health checks
type Checker struct {
	// Name is the name of the endpoint.
	Name string `json:"endpoint_name"`

	// URL is the host:port of the remote endpoint to check.
	URL string `json:"endpoint_url"`

	// Timeout is the maximum time to wait for a
	// TLS connection to be established.
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

	// CertExpiryThreshold is how close to expiration
	// the TLS certificate must be before declaring
	// a degraded status. Default is 14 days.
	CertExpiryThreshold time.Duration `json:"cert_expiry_threshold,omitempty"`

	// TrustedRoots is a list of PEM files to load as
	// trusted root CAs when connecting to TLS remotes.
	TrustedRoots []string `json:"trusted_roots,omitempty"`

	// tlsConfig is the config to use when making a TLS
	// connection. Values in this struct take precedence
	// over values described from the JSON (exported)
	// fields, where necessary.
	tlsConfig *tls.Config
}

// New creates a new Checker instance based on json config
func New(config json.RawMessage) (Checker, error) {
	var checker Checker
	err := json.Unmarshal(config, &checker)
	return checker, err
}

// Type returns the checker package name
func (Checker) Type() string {
	return Type
}

// Check performs checks using c according to its configuration.
// An error is only returned if there is a configuration error.
func (c Checker) Check() (types.Result, error) {
	if c.Attempts < 1 {
		c.Attempts = 1
	}
	if c.CertExpiryThreshold == 0 {
		c.CertExpiryThreshold = 24 * time.Hour * 14
	}

	if len(c.TrustedRoots) > 0 {
		if c.tlsConfig == nil {
			c.tlsConfig = new(tls.Config)
		}
		if c.tlsConfig.RootCAs == nil {
			c.tlsConfig.RootCAs = x509.NewCertPool()
		}
		for _, fname := range c.TrustedRoots {
			pemData, err := ioutil.ReadFile(fname)
			if err != nil {
				return types.Result{}, fmt.Errorf("error loading file: %w", err)
			}
			if !c.tlsConfig.RootCAs.AppendCertsFromPEM(pemData) {
				return types.Result{}, fmt.Errorf("error appending certs from PEM %s: %w", fname, err)
			}
		}
	}

	attempts, conns := c.doChecks()

	result := types.NewResult()
	result.Title = c.Name
	result.Endpoint = c.URL
	result.Times = attempts
	result.ThresholdRTT = c.ThresholdRTT

	return c.conclude(conns, result), nil
}

// doChecks executes the checks and returns each attempt
// along with its associated TLS connection. These connections
// will be open, so it's vital that conclude() is called,
// passing in the connections, so that they will be inspected
// and closed properly.
func (c Checker) doChecks() (types.Attempts, []*tls.Conn) {
	checks := make(types.Attempts, c.Attempts)
	conns := make([]*tls.Conn, c.Attempts)
	for i := 0; i < c.Attempts; i++ {
		dialer := &net.Dialer{Timeout: c.Timeout}
		start := time.Now()
		conn, err := tls.DialWithDialer(dialer, "tcp", c.URL, c.tlsConfig)
		checks[i].RTT = time.Since(start)
		conns[i] = conn
		if err != nil {
			checks[i].Error = err.Error()
			continue
		}
	}
	return checks, conns
}

// conclude takes the data in result from the attempts and
// computes remaining values needed to fill out the result.
// It detects less-than-ideal (degraded) connections and
// marks them as such. It closes the connections that are
// passed in.
func (c Checker) conclude(conns []*tls.Conn, result types.Result) types.Result {
	// close all connections when done
	defer func() {
		for _, conn := range conns {
			if conn != nil {
				conn.Close()
			}
		}
	}()

	// check errors (down)
	for i := range result.Times {
		if result.Times[i].Error != "" {
			result.Down = true
			return result
		}
	}

	// check if certificates expired (down)
	for i, conn := range conns {
		if conn == nil {
			continue
		}
		serverCerts := conn.ConnectionState().PeerCertificates
		if len(serverCerts) == 0 {
			result.Times[i].Error = "no certificates presented"
			result.Down = true
			return result
		}
		leaf := serverCerts[0]
		if leaf.NotAfter.Before(time.Now()) {
			result.Times[i].Error = fmt.Sprintf("certificate expired %s ago", time.Since(leaf.NotAfter))
			result.Down = true
			return result
		}
	}

	// check certificates expiring soon (degraded)
	for _, conn := range conns {
		if conn == nil {
			continue
		}
		serverCerts := conn.ConnectionState().PeerCertificates
		leaf := serverCerts[0]
		if until := time.Until(leaf.NotAfter); until < c.CertExpiryThreshold {
			result.Notice = fmt.Sprintf("certificate expiring soon (%s)", until)
			result.Degraded = true
			return result
		}
	}

	// check round trip time (degraded)
	if c.ThresholdRTT > 0 {
		result.Stats = result.ComputeStats()
		if result.Stats.Median > c.ThresholdRTT {
			result.Notice = fmt.Sprintf("median round trip time exceeded threshold (%s)", c.ThresholdRTT)
			result.Degraded = true
			return result
		}
	}

	result.Healthy = true
	return result
}
