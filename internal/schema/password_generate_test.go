package schema_test

import "testing"

// TestPasswordStepsTakePasswordOrGenerate: set_password and create_user need
// exactly one of password / generate, and generate's type is one of the three
// generators.
func TestPasswordStepsTakePasswordOrGenerate(t *testing.T) {
	host := func(step string) string {
		return "host:\n  name: web01\n  os: ubuntu22\n  size: small\n  disk: 40\n  steps:\n    - " + step + "\n"
	}
	for _, ok := range []string{
		`set_password: { name: root, password: "x" }`,
		`set_password: { name: root, generate: { type: passphrase, size: 3 } }`,
		`set_password: { name: root, generate: { type: wordchain, size: 5, label: "team:da" } }`,
		`create_user: { name: ops, generate: { type: random, size: 20 }, groups: [sudo] }`,
	} {
		if errs := validateHost(t, host(ok)); len(errs) != 0 {
			t.Errorf("%s: unexpected errors %+v", ok, errs)
		}
	}
	for _, bad := range []string{
		`set_password: { name: root }`,
		`set_password: { name: root, password: "x", generate: { type: passphrase, size: 3 } }`,
		`set_password: { name: root, generate: { type: dice, size: 3 } }`,
		`set_password: { name: root, generate: { type: passphrase } }`,
		`create_user: { name: ops, generate: { type: random, size: "long" } }`,
	} {
		if errs := validateHost(t, host(bad)); len(errs) == 0 {
			t.Errorf("%s: accepted, want a schema error", bad)
		}
	}
}
