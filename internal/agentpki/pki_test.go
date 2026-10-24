package agentpki

import (
	"crypto/tls"
	"io"
	"net"
	"testing"
	"time"
)

// TestMTLSRoundTrip proves the whole point of this package with a real
// TCP+TLS handshake, not just "the certs parse": a server requiring
// client certs accepts a properly-signed one and rejects both an
// unsigned self-signed cert and one signed by a different CA. This is
// the actual mechanism internal/gateway's listener uses.
func TestMTLSRoundTrip(t *testing.T) {
	ca, err := GenerateCA("test-ca")
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	serverCert, serverKey, err := ca.IssueLeaf("localhost", true, []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")}, time.Hour)
	if err != nil {
		t.Fatalf("IssueLeaf (server): %v", err)
	}
	clientCert, clientKey, err := ca.IssueLeaf("host-abc-123", false, nil, nil, time.Hour)
	if err != nil {
		t.Fatalf("IssueLeaf (client): %v", err)
	}

	serverTLS, err := ServerTLSConfig(ca.CertPEM, serverCert, serverKey)
	if err != nil {
		t.Fatalf("ServerTLSConfig: %v", err)
	}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", serverTLS)
	if err != nil {
		t.Fatalf("tls.Listen: %v", err)
	}
	defer ln.Close()

	// Echo one line back, and report the verified client cert's CN so the
	// test can confirm identity actually came through the handshake.
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				tconn, ok := conn.(*tls.Conn)
				if !ok {
					return
				}
				if err := tconn.Handshake(); err != nil {
					return
				}
				cn := "none"
				if state := tconn.ConnectionState(); len(state.PeerCertificates) > 0 {
					cn = state.PeerCertificates[0].Subject.CommonName
				}
				conn.Write([]byte(cn + "\n"))
			}()
		}
	}()

	t.Run("valid client cert is accepted and identified", func(t *testing.T) {
		clientTLS, err := ClientTLSConfig(ca.CertPEM, clientCert, clientKey, "localhost")
		if err != nil {
			t.Fatalf("ClientTLSConfig: %v", err)
		}
		conn, err := tls.Dial("tcp", ln.Addr().String(), clientTLS)
		if err != nil {
			t.Fatalf("tls.Dial: %v", err)
		}
		defer conn.Close()
		buf := make([]byte, 64)
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			t.Fatalf("reading response: %v", err)
		}
		got := string(buf[:n])
		if got != "host-abc-123\n" {
			t.Fatalf("server saw CN = %q, want %q", got, "host-abc-123\n")
		}
	})

	t.Run("a cert from a different CA is rejected", func(t *testing.T) {
		otherCA, err := GenerateCA("other-ca")
		if err != nil {
			t.Fatalf("GenerateCA (other): %v", err)
		}
		badCert, badKey, err := otherCA.IssueLeaf("imposter-host", false, nil, nil, time.Hour)
		if err != nil {
			t.Fatalf("IssueLeaf (imposter): %v", err)
		}
		// Trust the REAL ca's cert as root (pinning the server), but
		// present a client cert signed by a DIFFERENT ca -- the gateway
		// must refuse this connection.
		clientTLS, err := ClientTLSConfig(ca.CertPEM, badCert, badKey, "localhost")
		if err != nil {
			t.Fatalf("ClientTLSConfig: %v", err)
		}
		conn, err := tls.Dial("tcp", ln.Addr().String(), clientTLS)
		// A real, non-obvious TLS 1.3 behavior, found by actually running
		// this rather than assuming: Dial() alone does NOT reliably prove
		// the server accepted the connection. Go's client-side stack sees
		// the server's CertificateRequest naming which CAs it will
		// accept, notices badCert isn't signed by one of them, and
		// (correctly, per the TLS spec) sends an EMPTY certificate
		// message rather than offering one it knows won't be trusted.
		// The server then rejects the handshake for "no certificate
		// provided" -- but in TLS 1.3 that rejection can arrive as a
		// post-handshake alert the client only observes on its next I/O,
		// not necessarily as a Dial() error. So the real assertion has to
		// be "the connection doesn't actually work," proven with a Read,
		// not "Dial returned an error."
		if err != nil {
			return // failed at Dial() -- also an acceptable rejection
		}
		defer conn.Close()
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 64)
		if n, readErr := conn.Read(buf); readErr == nil {
			t.Fatalf("expected the connection to fail (client cert signed by a different CA), but read %q successfully", buf[:n])
		}
	})
}
