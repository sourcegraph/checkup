package checkup

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"time"
)

// HTTPSChecker implements a Checker for HTTP endpoints.
type HTTPSChecker struct {
	*HTTPChecker
}

// Check performs checks using c according to its configuration.
// An error is only returned if there is a configuration error.
func (c HTTPSChecker) Check() (Result, error) {
	if c.Attempts < 1 {
		c.Attempts = 1
	}
	if c.Client == nil {
		c.Client = DefaultHTTPSClient
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

// DefaultHTTPSClient is used when no other http.Client
// is specified on a HTTPSChecker.
var DefaultHTTPSClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Proxy:           http.ProxyFromEnvironment,
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
