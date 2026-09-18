//go:build production

package main

import (
	"crypto/tls"
	"errors"
)

func loadEmbeddedGRPCCertificate() (*tls.Certificate, error) {
	return nil, errors.New("production builds require agent.grpc_tls_cert_path and agent.grpc_tls_key_path")
}
