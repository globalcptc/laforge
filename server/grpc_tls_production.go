//go:build production

package main

import "crypto/tls"

func loadEmbeddedGRPCCertificate() (*tls.Certificate, error) {
	return nil, nil
}
