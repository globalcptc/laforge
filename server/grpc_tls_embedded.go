//go:build !production

package main

import (
	"crypto/tls"
	"fmt"

	grpcserver "github.com/gen0cide/laforge/grpc/server"
	"github.com/gen0cide/laforge/grpc/server/static"
)

func loadEmbeddedGRPCCertificate() (*tls.Certificate, error) {
	certPEM, err := static.ReadFile(grpcserver.CertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded gRPC certificate: %w", err)
	}
	keyPEM, err := static.ReadFile(grpcserver.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded gRPC key: %w", err)
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse embedded gRPC certificate: %w", err)
	}
	return &cert, nil
}
