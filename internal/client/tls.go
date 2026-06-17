package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"
)

// TLSOptions configures the transport for talking to FerrisKey instances that
// use a private CA or (in development) an untrusted certificate.
type TLSOptions struct {
	// CACertPEM is an optional PEM-encoded CA certificate (or bundle) added to
	// the trust store, for instances served by a private CA.
	CACertPEM string
	// InsecureSkipVerify disables TLS certificate verification. Development
	// only — never use against production.
	InsecureSkipVerify bool
}

// IsZero reports whether no TLS customization was requested, in which case the
// caller can keep the default HTTP client.
func (o TLSOptions) IsZero() bool {
	return o.CACertPEM == "" && !o.InsecureSkipVerify
}

// HTTPClientWithTLS builds an *http.Client honoring the TLS options. A custom
// CA is appended to the system roots (falling back to an empty pool if the
// system pool is unavailable).
func HTTPClientWithTLS(o TLSOptions) (*http.Client, error) {
	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: o.InsecureSkipVerify, //nolint:gosec // opt-in, dev-only, documented
	}

	if o.CACertPEM != "" {
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM([]byte(o.CACertPEM)) {
			return nil, fmt.Errorf("ca_cert does not contain any valid PEM certificate")
		}
		tlsCfg.RootCAs = pool
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsCfg

	return &http.Client{
		Timeout:   60 * time.Second,
		Transport: transport,
	}, nil
}
