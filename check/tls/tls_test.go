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

func makeSelfSignedCert(hostname, keyType string, validity time.Duration) (tls.Certificate, error) {
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

	// create certificate structure with proper values
	notBefore := time.Now()
	notAfter := notBefore.Add(validity)
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
