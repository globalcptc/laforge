package render

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"
)

// GenSSHKey names the SSH key generator in generatorSeed's derivation, so a
// key never shares a seed with a password of the same label.
const GenSSHKey = "ssh-ed25519"

// SSHKey is a generated Ed25519 key pair, as a template sees it:
//
//	{{ $k := sshkey "team:vdi-pair-01" }}
//	{{ $k.private }}      OpenSSH private key file (unencrypted), LF line endings
//	{{ $k.public }}       authorized_keys / known_hosts form: "ssh-ed25519 AAAA..."
//	{{ $k.fingerprint }}  "SHA256:..." as ssh-keygen -l and PuTTY's -hostkey take it
//
// Keys are stable for the build exactly like generated passwords: derived from
// the build's render secret, scoped to the host by default, or to the team
// (`team:` label) or every team (`build:` label). Two hosts that render the same
// team-scoped label get the same key with no contact between them, and a host
// rebuilt within the build gets its key back -- which is what lets one host
// hold a private key and another trust its public half (or pin a host key)
// without a handshake at deploy time.
func SSHKey(ctx *Context, label string) map[string]string {
	seed := generatorSeed(ctx, GenSSHKey, 0, label)
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := priv.Public().(ed25519.PublicKey)

	blob := sshString(nil, []byte("ssh-ed25519"))
	blob = sshString(blob, pub)
	sum := sha256.Sum256(blob)
	return map[string]string{
		"private":     opensshPrivateKey(priv, pub, blob, sshKeyComment(label), seed),
		"public":      "ssh-ed25519 " + base64.StdEncoding.EncodeToString(blob),
		"fingerprint": "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:]),
	}
}

// sshKeyComment labels the key file with what generated it, minus the scope
// prefix ("team:vdi-pair-01" -> "laforge vdi-pair-01").
func sshKeyComment(label string) string {
	name := label
	if i := strings.IndexByte(label, ':'); i >= 0 {
		name = label[i+1:]
	}
	if name == "" {
		return "laforge"
	}
	return "laforge " + name
}

// opensshPrivateKey encodes an unencrypted key in OpenSSH's
// "openssh-key-v1" format (PROTOCOL.key), the form ssh, ssh-keygen and
// WinSCP's key converter read. The two check words are derived from the seed
// so the output is byte-for-byte stable for the build.
func opensshPrivateKey(priv ed25519.PrivateKey, pub ed25519.PublicKey, pubBlob []byte, comment string, seed [32]byte) string {
	checkSum := sha256.Sum256(append([]byte("laforge-sshkey-check\x00"), seed[:]...))
	check := binary.BigEndian.Uint32(checkSum[:4])
	var section []byte
	section = binary.BigEndian.AppendUint32(section, check)
	section = binary.BigEndian.AppendUint32(section, check)
	section = sshString(section, []byte("ssh-ed25519"))
	section = sshString(section, pub)
	section = sshString(section, priv) // 64 bytes: seed then public key
	section = sshString(section, []byte(comment))
	for i := byte(1); len(section)%8 != 0; i++ {
		section = append(section, i)
	}

	out := []byte("openssh-key-v1\x00")
	out = sshString(out, []byte("none")) // cipher
	out = sshString(out, []byte("none")) // kdf
	out = sshString(out, nil)            // kdf options
	out = binary.BigEndian.AppendUint32(out, 1)
	out = sshString(out, pubBlob)
	out = sshString(out, section)

	// ssh-keygen wraps the base64 body at 70 characters; pem.Encode would
	// wrap at 64, which OpenSSH also reads, but match the tool's own output.
	body := base64.StdEncoding.EncodeToString(out)
	var b strings.Builder
	b.WriteString("-----BEGIN OPENSSH PRIVATE KEY-----\n")
	for len(body) > 70 {
		b.WriteString(body[:70] + "\n")
		body = body[70:]
	}
	b.WriteString(body + "\n-----END OPENSSH PRIVATE KEY-----\n")
	return b.String()
}

// sshString appends v as an SSH wire-format string: a uint32 length, then the
// bytes.
func sshString(dst, v []byte) []byte {
	dst = binary.BigEndian.AppendUint32(dst, uint32(len(v)))
	return append(dst, v...)
}

// sshkeyFunc is the template form: {{ sshkey }} for this host's own key, or
// {{ sshkey "label" }} for a labelled (and optionally team- or build-shared)
// one.
func sshkeyFunc(ctx *Context) func(...string) (map[string]string, error) {
	return func(label ...string) (map[string]string, error) {
		if len(label) > 1 {
			return nil, fmt.Errorf("sshkey takes at most one label")
		}
		l := ""
		if len(label) == 1 {
			l = label[0]
		}
		return SSHKey(ctx, l), nil
	}
}
