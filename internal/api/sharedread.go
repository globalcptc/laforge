package api

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

// sharedReadMaxAge is how old a shared response may be when it's served.
// Pages poll these every 5s, so a response at most this old is as current as
// the page would have shown anyway.
const sharedReadMaxAge = 2 * time.Second

// sharedReads lets every viewer of a build share one computation of the
// build-wide reads each open page polls (the build, its hosts, its
// dashboard). Without it, each open tab re-ran the same queries every few
// seconds, so the database load grew with the number of people watching.
// With it, a build's response is computed at most once per sharedReadMaxAge
// however many tabs ask, and requests arriving while it's being computed
// wait for that one rather than starting their own.
//
// Only responses that are the same for everyone may be shared: callers
// check the requester's access first, then call serve.
type sharedReads struct {
	mu      sync.Mutex
	entries map[string]*sharedEntry
}

type sharedEntry struct {
	ready  chan struct{} // closed once computed
	at     time.Time
	status int
	header http.Header
	body   []byte
}

// serve answers r from the shared response for key, computing it with fill
// when there's none recent enough. fill runs detached from any one
// requester, so a viewer closing their tab doesn't fail the others waiting
// on it. Only 200s are kept; anything else is answered to the requests that
// waited on it and then forgotten.
func (s *sharedReads) serve(w http.ResponseWriter, r *http.Request, key string, fill func(http.ResponseWriter, *http.Request)) {
	key = key + "?" + r.URL.RawQuery
	s.mu.Lock()
	if s.entries == nil {
		s.entries = map[string]*sharedEntry{}
	}
	e := s.entries[key]
	fresh := e != nil && (e.at.IsZero() || time.Since(e.at) < sharedReadMaxAge)
	if !fresh {
		e = &sharedEntry{ready: make(chan struct{})}
		s.entries[key] = e
		s.sweepLocked()
		s.mu.Unlock()
		s.fill(e, key, r, fill)
	} else {
		s.mu.Unlock()
	}
	select {
	case <-e.ready:
	case <-r.Context().Done():
		return
	}
	for k, v := range e.header {
		w.Header()[k] = v
	}
	w.WriteHeader(e.status)
	w.Write(e.body)
}

func (s *sharedReads) fill(e *sharedEntry, key string, r *http.Request, fill func(http.ResponseWriter, *http.Request)) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 60*time.Second)
	defer cancel()
	rec := &bufferedResponse{header: http.Header{}, status: http.StatusOK}
	func() {
		defer func() {
			if p := recover(); p != nil {
				rec.status, rec.body = http.StatusInternalServerError, *bytes.NewBufferString(`{"error":"internal error"}`)
			}
		}()
		fill(rec, r.WithContext(ctx))
	}()
	// The response is set before ready closes (waiters read it only after);
	// at is read under the lock by requests deciding whether it's fresh.
	e.status, e.header, e.body = rec.status, rec.header, rec.body.Bytes()
	s.mu.Lock()
	e.at = time.Now()
	if e.status != http.StatusOK && s.entries[key] == e {
		delete(s.entries, key)
	}
	s.mu.Unlock()
	close(e.ready)
}

// sweepLocked drops entries that are well past serving, so builds nobody is
// watching any more don't keep their last response in memory.
func (s *sharedReads) sweepLocked() {
	for k, e := range s.entries {
		if !e.at.IsZero() && time.Since(e.at) > time.Minute {
			delete(s.entries, k)
		}
	}
}

// bufferedResponse captures a response so it can be served to many requests.
type bufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (b *bufferedResponse) Header() http.Header         { return b.header }
func (b *bufferedResponse) WriteHeader(code int)        { b.status = code }
func (b *bufferedResponse) Write(p []byte) (int, error) { return b.body.Write(p) }

// forgetBuild drops a build's shared responses, so the next request after a
// change (new events, an operator's action) computes them fresh instead of
// waiting out sharedReadMaxAge.
func (s *sharedReads) forgetBuild(buildID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k := range s.entries {
		if strings.Contains(k, ":"+buildID+"?") {
			delete(s.entries, k)
		}
	}
}
