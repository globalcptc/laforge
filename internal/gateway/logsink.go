package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// LogRecord is one enriched console line as it leaves the gateway for a log
// sink: self-contained, so an external ingester (Splunk, Loki via Vector,
// Elastic, ...) needs nothing from LaForge to index it. The agent sends only
// {ts_ms, stream, line}; the gateway adds the build / team / object identity it
// resolves from the agent's cert. Vendor-neutral on purpose -- it serializes to
// one JSON object per line (NDJSON), which any collector can reshape.
type LogRecord struct {
	TS       string            `json:"ts"` // RFC3339 (millis), from the agent's ts_ms
	BuildID  string            `json:"build_id,omitempty"`
	Team     int32             `json:"team"`
	Object   string            `json:"object,omitempty"`
	ObjectID string            `json:"object_id"`
	Kind     string            `json:"kind,omitempty"`
	Stream   string            `json:"stream"`
	Line     string            `json:"line"`
	Dropped  int               `json:"dropped,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
}

// LogSink is where enriched console lines go. Enqueue must never block the
// agent-facing connection -- a slow or down backend drops lines (counted), it
// does not stall an agent. Close flushes and stops the shipper.
type LogSink interface {
	Enqueue(rec LogRecord)
	Close() error
}

// HTTPSink ships records as batched NDJSON (application/x-ndjson) to one HTTP
// endpoint, on its own background goroutine. Deliberately generic: the URL,
// optional headers (an auth token, a Splunk HEC header, ...) and optional
// static labels are all config, so the actual backend is a deployment choice,
// not code. For a backend that speaks a vendor wire format (Loki's push API,
// Elastic's bulk format), front this with Vector / Promtail / Fluent Bit.
type HTTPSink struct {
	url     string
	headers map[string]string
	labels  map[string]string
	client  *http.Client

	ch      chan LogRecord
	done    chan struct{}
	wg      sync.WaitGroup
	dropped atomic.Int64
}

const (
	logSinkQueueCap  = 20000
	logSinkBatchMax  = 500
	logSinkFlushTick = 2 * time.Second
	logSinkPostTries = 3
)

// NewHTTPSink starts a sink shipping to url. headers and labels may be nil.
func NewHTTPSink(url string, headers, labels map[string]string) *HTTPSink {
	s := &HTTPSink{
		url:     url,
		headers: headers,
		labels:  labels,
		client:  &http.Client{Timeout: 10 * time.Second},
		ch:      make(chan LogRecord, logSinkQueueCap),
		done:    make(chan struct{}),
	}
	s.wg.Add(1)
	go s.run()
	return s
}

// Enqueue hands a record to the shipper without blocking: if the buffer is full
// (a backend that can't keep up), the record is dropped and counted rather than
// back-pressuring the agent connection that produced it.
func (s *HTTPSink) Enqueue(rec LogRecord) {
	if len(s.labels) > 0 {
		rec.Labels = s.labels
	}
	select {
	case s.ch <- rec:
	default:
		if n := s.dropped.Add(1); n%1000 == 1 {
			log.Printf("gateway: log sink buffer full, dropping records (total dropped: %d)", n)
		}
	}
}

func (s *HTTPSink) Close() error {
	close(s.done)
	s.wg.Wait()
	return nil
}

// run accumulates records and flushes them as one NDJSON POST on whichever
// comes first: a full batch or the flush tick. Drains what's buffered on close.
func (s *HTTPSink) run() {
	defer s.wg.Done()
	ticker := time.NewTicker(logSinkFlushTick)
	defer ticker.Stop()
	batch := make([]LogRecord, 0, logSinkBatchMax)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		s.post(batch)
		batch = batch[:0]
	}
	for {
		select {
		case rec := <-s.ch:
			batch = append(batch, rec)
			if len(batch) >= logSinkBatchMax {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-s.done:
			// Drain anything already queued, then a final flush.
			for {
				select {
				case rec := <-s.ch:
					batch = append(batch, rec)
					if len(batch) >= logSinkBatchMax {
						flush()
					}
					continue
				default:
				}
				break
			}
			flush()
			return
		}
	}
}

// post sends one batch as newline-delimited JSON, retrying a bounded number of
// times with linear backoff. A batch that still won't send is dropped (and
// logged), never retried forever -- the shipper must not fall behind live
// agents because one delivery is wedged.
func (s *HTTPSink) post(batch []LogRecord) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf) // Encoder writes a trailing newline per value -> NDJSON
	for i := range batch {
		if err := enc.Encode(&batch[i]); err != nil {
			log.Printf("gateway: log sink: encoding record: %v", err)
		}
	}
	body := buf.Bytes()
	for attempt := 1; attempt <= logSinkPostTries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(body))
		if err != nil {
			cancel()
			log.Printf("gateway: log sink: building request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/x-ndjson")
		for k, v := range s.headers {
			req.Header.Set(k, v)
		}
		resp, err := s.client.Do(req)
		cancel()
		if err == nil {
			ok := resp.StatusCode >= 200 && resp.StatusCode < 300
			resp.Body.Close()
			if ok {
				return
			}
			err = &httpStatusError{code: resp.StatusCode}
		}
		if attempt == logSinkPostTries {
			log.Printf("gateway: log sink: dropping %d records after %d tries: %v", len(batch), attempt, err)
			return
		}
		select {
		case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
		case <-s.done:
			return
		}
	}
}

type httpStatusError struct{ code int }

func (e *httpStatusError) Error() string { return http.StatusText(e.code) }
