//go:build unix

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// watchResize calls onResize whenever the local terminal is resized (SIGWINCH),
// so the remote PTY follows the window. Returns a stop function.
func watchResize(onResize func()) func() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ch:
				onResize()
			case <-done:
				return
			}
		}
	}()
	return func() {
		signal.Stop(ch)
		close(done)
	}
}
