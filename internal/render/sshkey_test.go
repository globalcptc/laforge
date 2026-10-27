package render

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSSHKeysAreStableAndScoped: a team-labelled key is the same on every host
// of the team (one host holds the private key, the other trusts the public
// half, with no contact between them), and the same again when a host is
// rebuilt within the build; it differs across teams and builds; an unlabelled
// key is the host's own.
func TestSSHKeysAreStableAndScoped(t *testing.T) {
	secret := []byte("build-secret")
	win := &Context{Team: 3, As: "vdi-win-01", Secret: secret}
	kali := &Context{Team: 3, As: "vdi-kali-01", Secret: secret}

	a, b := SSHKey(win, "team:vdi-pair-01"), SSHKey(kali, "team:vdi-pair-01")
	if a["private"] != b["private"] || a["public"] != b["public"] {
		t.Fatal("a team-labelled key differs between two hosts of the same team")
	}
	if again := SSHKey(win, "team:vdi-pair-01"); again["private"] != a["private"] {
		t.Fatal("rendering the same key twice gave different keys")
	}
	if SSHKey(win, "team:vdi-pair-02")["public"] == a["public"] {
		t.Fatal("two labels gave the same key")
	}
	if SSHKey(&Context{Team: 4, As: "vdi-win-01", Secret: secret}, "team:vdi-pair-01")["public"] == a["public"] {
		t.Fatal("two teams got the same team-scoped key")
	}
	if SSHKey(&Context{Team: 3, As: "vdi-win-01", Secret: []byte("next-build")}, "team:vdi-pair-01")["public"] == a["public"] {
		t.Fatal("a new build reused the last build's key")
	}
	if SSHKey(win, "")["public"] == SSHKey(kali, "")["public"] {
		t.Fatal("two hosts' own (unlabelled) keys are the same")
	}
	// A key never shares its seed with a password of the same label.
	pw, _ := GeneratePassword(win, GenRandom, 32, "team:vdi-pair-01")
	if strings.Contains(a["private"], pw) {
		t.Fatal("key material overlaps a password")
	}
}

// TestSSHKeyIsWhatOpenSSHReads checks the output with the real ssh-keygen: it
// reads the private key without a passphrase, derives exactly the public key
// LaForge rendered, and reports the same fingerprint.
func TestSSHKeyIsWhatOpenSSHReads(t *testing.T) {
	keygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("ssh-keygen not installed")
	}
	k := SSHKey(&Context{Team: 1, As: "vdi-kali-01", Secret: []byte("s")}, "team:vdi-kali-host-01")
	if !strings.HasPrefix(k["private"], "-----BEGIN OPENSSH PRIVATE KEY-----\n") || !strings.HasSuffix(k["private"], "-----END OPENSSH PRIVATE KEY-----\n") {
		t.Fatalf("private key armor wrong:\n%s", k["private"])
	}
	path := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(path, []byte(k["private"]), 0o600); err != nil {
		t.Fatal(err)
	}
	derived, err := exec.Command(keygen, "-y", "-P", "", "-f", path).Output()
	if err != nil {
		t.Fatalf("ssh-keygen -y rejected the private key: %v", err)
	}
	if fields := strings.Fields(string(derived)); len(fields) < 2 || fields[0]+" "+fields[1] != k["public"] {
		t.Fatalf("ssh-keygen derived %q, LaForge rendered %q", strings.TrimSpace(string(derived)), k["public"])
	}
	pubPath := path + ".pub"
	if err := os.WriteFile(pubPath, []byte(k["public"]+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fp, err := exec.Command(keygen, "-l", "-E", "sha256", "-f", pubPath).Output()
	if err != nil {
		t.Fatalf("ssh-keygen -l: %v", err)
	}
	if fields := strings.Fields(string(fp)); len(fields) < 2 || fields[1] != k["fingerprint"] {
		t.Fatalf("ssh-keygen fingerprint %q, LaForge rendered %q", strings.TrimSpace(string(fp)), k["fingerprint"])
	}
}

// TestSSHKeyTemplate: the template form, as the VDI scripts use it -- a label
// built from the host's own name with printf and slice.
func TestSSHKeyTemplate(t *testing.T) {
	ctx := &Context{Team: 2, As: "vdi-kali-07", Secret: []byte("s")}
	out, err := RenderString("t", `{{ $k := sshkey (printf "team:vdi-pair-%s" (slice "vdi-kali-07" 9)) }}{{ $k.public }}|{{ $k.fingerprint }}`, ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := SSHKey(ctx, "team:vdi-pair-07")
	if out != want["public"]+"|"+want["fingerprint"] {
		t.Fatalf("template rendered %q, want the team:vdi-pair-07 key", out)
	}
	if _, err := RenderString("t", `{{ sshkey "a" "b" }}`, ctx, nil); err == nil {
		t.Fatal("sshkey with two labels rendered, want an error")
	}
}
