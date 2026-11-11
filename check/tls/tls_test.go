package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

var errUnknownPrivateKeyType = errors.New("unknown private key type")

func TestChecker(t *testing.T) {
	selfSigned, err := makeSelfSignedCert("localhost", "", time.Hour*24*30)
	if err != nil {
		t.Fatal(err)
	}

	endpt := "localhost:4043"
	config := &tls.Config{Certificates: []tls.Certificate{selfSigned}}

	ln, err := tls.Listen("tcp", endpt, config)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				break
			}
			_, _ = conn.Read(nil) // necessary, otherwise client-side Dial hangs or returns EOF
			_ = conn.Close()
		}
	}()

	tc := Checker{
		Name:      "Test",
		URL:       endpt,
		Attempts:  2,
		tlsConfig: &tls.Config{RootCAs: x509.NewCertPool()},
	}

	// allow our self-signed certificate to be considered valid
	tc.tlsConfig.RootCAs.AddCert(selfSigned.Leaf)

	// Try an up server
	result, err := tc.Check()
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
	if got, want := len(result.Times), tc.Attempts; got != want {
		t.Errorf("Expected %d attempts, got %d", want, got)
	}
	ts := time.Unix(0, result.Timestamp)
	if time.Since(ts) > 5*time.Second {
		t.Errorf("Expected timestamp to be recent, got %s", ts)
	}

	// check Degraded by long connection time
	tc.ThresholdRTT = 1 * time.Nanosecond
	result, err = tc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Degraded, true; got != want {
		t.Errorf("Expected result.Degraded=%v, got %v", want, got)
	}
	tc.ThresholdRTT = 0

	// check Degraded by certificate expiring "soon"
	tc.CertExpiryThreshold = 24 * time.Hour * 90
	result, err = tc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Degraded, true; got != want {
		t.Errorf("Expected result.Degraded=%v, got %v", want, got)
	}
	tc.CertExpiryThreshold = 0

	// check Down by handshake timeout
	tc.Timeout = 1 * time.Nanosecond
	result, err = tc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := result.Down, true; got != want {
		t.Errorf("Expected result.Down=%v, got %v", want, got)
	}
	tc.Timeout = 0

	// check Down when server is not even online
	ln.Close()
	result, err = tc.Check()
	if err != nil {
		t.Errorf("Didn't expect an error: %v", err)
	}
	if got, want := len(result.Times), tc.Attempts; got != want {
		t.Errorf("Expected %d attempts, got %d", want, got)
	}
	if got, want := result.Down, true; got != want {
		t.Errorf("Expected result.Down=%v, got %v", want, got)
	}
}

func makeCertWithValidity(hostname, keyType string, notBefore, notAfter time.Time) (tls.Certificate, error) {
	// start by generating private key
	var privKey interface{}
	var err error
	switch keyType {
	case "", "ec256":
		privKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "ec384":
		privKey, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case "rsa2048":
		privKey, err = rsa.GenerateKey(rand.Reader, 2048)
	case "rsa4096":
		privKey, err = rsa.GenerateKey(rand.Reader, 4096)
	case "rsa8192":
		privKey, err = rsa.GenerateKey(rand.Reader, 8192)
	default:
		return tls.Certificate{}, fmt.Errorf("cannot generate private key: %w", errUnknownPrivateKeyType)
	}
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate private key: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate serial number: %w", err)
	}
	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{Organization: []string{"Checkup Test"}},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	cert.DNSNames = append(cert.DNSNames, hostname)

	publicKey := func(privKey interface{}) interface{} {
		switch k := privKey.(type) {
		case *rsa.PrivateKey:
			return &k.PublicKey
		case *ecdsa.PrivateKey:
			return &k.PublicKey
		}
		return errUnknownPrivateKeyType
	}

	// TODO: I don't know a way to get a proper x509.Certificate without getting
	// its ASN1 encoding then decoding it. Either way, we need both representations.
	derBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, publicKey(privKey), privKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("could not create certificate: %w", err)
	}
	finalCert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  privKey,
		Leaf:        finalCert,
	}, nil
}

func makeSelfSignedCert(hostname, keyType string, validity time.Duration) (tls.Certificate, error) {
	notBefore := time.Now()
	notAfter := notBefore.Add(validity)
	return makeCertWithValidity(hostname, keyType, notBefore, notAfter)
}

func makeExpiredCert(hostname, keyType string, expiredAgo time.Duration) (tls.Certificate, error) {
	notBefore := time.Now().Add(-2 * expiredAgo)
	notAfter := time.Now().Add(-expiredAgo)
	return makeCertWithValidity(hostname, keyType, notBefore, notAfter)
}

func TestCheckerExpiredCertificate(t *testing.T) {
	expiredAgo := 24 * time.Hour
	expiredCert, err := makeExpiredCert("localhost", "", expiredAgo)
	if err != nil {
		t.Fatal(err)
	}

	config := &tls.Config{Certificates: []tls.Certificate{expiredCert}}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", config)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				break
			}
			_, _ = conn.Read(nil)
			_ = conn.Close()
		}
	}()

	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	tc := Checker{
		Name:      "TestExpiredCert",
		URL:       net.JoinHostPort("localhost", port),
		Attempts:  2,
		tlsConfig: &tls.Config{RootCAs: x509.NewCertPool()},
	}
	tc.tlsConfig.RootCAs.AddCert(expiredCert.Leaf)

	result, err := tc.Check()
	if err != nil {
		t.Fatalf("unexpected Check error: %v", err)
	}

	if got, want := result.Down, true; got != want {
		t.Errorf("expected result.Down=%v, got %v", want, got)
	}
	if got, want := result.Healthy, false; got != want {
		t.Errorf("expected result.Healthy=%v, got %v", want, got)
	}
	if len(result.Times) != tc.Attempts {
		t.Fatalf("expected %d attempts, got %d", tc.Attempts, len(result.Times))
	}
	for i, attempt := range result.Times {
		if attempt.Error == "" {
			t.Errorf("expected attempt %d to have an error", i)
			continue
		}
		if !strings.Contains(attempt.Error, "certificate expired") {
			t.Errorf("expected attempt %d error to contain 'certificate expired', got: %s", i, attempt.Error)
		}
		if !strings.Contains(attempt.Error, "ago") {
			t.Errorf("expected attempt %d error to contain 'ago', got: %s", i, attempt.Error)
		}
	}
}

func TestCheckerExpiredCertificateInsecureSkipVerify(t *testing.T) {
	expiredAgo := 12 * time.Hour
	expiredCert, err := makeExpiredCert("localhost", "", expiredAgo)
	if err != nil {
		t.Fatal(err)
	}

	config := &tls.Config{Certificates: []tls.Certificate{expiredCert}}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", config)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				break
			}
			_, _ = conn.Read(nil)
			_ = conn.Close()
		}
	}()

	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	tc := Checker{
		Name:     "TestExpiredCertInsecure",
		URL:      net.JoinHostPort("localhost", port),
		Attempts: 1,
		tlsConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	result, err := tc.Check()
	if err != nil {
		t.Fatalf("unexpected Check error: %v", err)
	}

	if got, want := result.Down, true; got != want {
		t.Errorf("expected result.Down=%v, got %v", want, got)
	}
	if got, want := result.Healthy, false; got != want {
		t.Errorf("expected result.Healthy=%v, got %v", want, got)
	}
	if len(result.Times) != 1 {
		t.Fatalf("expected 1 attempt, got %d", len(result.Times))
	}
	if !strings.Contains(result.Times[0].Error, "certificate expired") {
		t.Errorf("expected error to contain 'certificate expired', got: %s", result.Times[0].Error)
	}
	if !strings.Contains(result.Times[0].Error, "ago") {
		t.Errorf("expected error to contain 'ago', got: %s", result.Times[0].Error)
	}
}
