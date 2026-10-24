//go:build !unix

package main

// watchResize is a no-op where SIGWINCH doesn't exist (Windows): the remote PTY
// keeps the size sent at connect. Live resizing is a platform follow-up.
func watchResize(onResize func()) func() {
	return func() {}
}
