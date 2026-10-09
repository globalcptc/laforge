package microcloud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"
)

// Keep retries local to MicroCloud. Cluster DB and Ceph recovery need time;
// retrying immediately just adds load. The caller's cancellation always wins.
var instanceRetryDelays = [...]time.Duration{15 * time.Second, 30 * time.Second, time.Minute, 2 * time.Minute, 2 * time.Minute}

func transientInstanceError(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}
	var pending *pendingOperationError
	if errors.As(err, &pending) && errors.Is(pending.Err, errOperationRunning) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		code := apiErr.HTTPStatus
		if apiErr.ErrorCode >= 400 {
			code = apiErr.ErrorCode
		}
		if code == http.StatusUnauthorized || code == http.StatusForbidden || code == http.StatusNotFound {
			return false
		}
		msg := strings.ToLower(apiErr.Message)
		if apiErr.Busy() || code == http.StatusRequestTimeout || code == http.StatusTooManyRequests || code == http.StatusBadGateway || code == http.StatusServiceUnavailable || code == http.StatusGatewayTimeout {
			return true
		}
		// Some operation failures are HTTP 200 with an inner 400/500. Only
		// recognized transient failures should retry, not every server error.
		for _, fragment := range []string{"database is locked", "database is busy", "failed to begin transaction", "context deadline exceeded", "connection refused", "connection reset", "no available leader", "is down", "resource temporarily unavailable", "device or resource busy"} {
			if strings.Contains(msg, fragment) {
				return true
			}
		}
		return false
	}
	var networkErr net.Error
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) || (errors.As(err, &networkErr) && (networkErr.Timeout() || networkErr.Temporary()))
}

func (b *Builder) retryInstanceStep(ctx context.Context, name, step string, fn func() error) error {
	started := time.Now()
	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := fn()
		if err == nil {
			slog.InfoContext(ctx, "MicroCloud instance operation completed", "project", b.Client.Project, "instance", name, "step", step, "attempts", attempt+1, "elapsed", time.Since(started))
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if attempt == len(instanceRetryDelays) || !transientInstanceError(err) {
			slog.ErrorContext(ctx, "MicroCloud instance operation failed", "project", b.Client.Project, "instance", name, "step", step, "attempts", attempt+1, "elapsed", time.Since(started), "error", err)
			return fmt.Errorf("%s instance %s (after %d attempt(s)): %w", step, name, attempt+1, err)
		}
		delay := instanceRetryDelays[attempt]
		slog.WarnContext(ctx, "MicroCloud instance operation will retry", "instance", name, "step", step, "attempt", attempt+1, "delay", delay, "error", err)
		wait := b.retryWait
		if wait == nil {
			wait = waitInstanceRetry
		}
		if err := wait(ctx, delay); err != nil {
			return err
		}
	}
}

func waitInstanceRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// An unsuccessful wait is not a failed operation. Preserve its ID so retries
// poll the original operation instead of submitting another create/start.
var errOperationRunning = errors.New("operation has not finished")

type pendingOperationError struct {
	Path string
	Err  error
}

func (e *pendingOperationError) Error() string { return e.Err.Error() }
func (e *pendingOperationError) Unwrap() error { return e.Err }

func (b *Builder) createInstance(ctx context.Context, name string, body interface{}) error {
	var pending *pendingOperationError
	checkExisting := false
	return b.retryInstanceStep(ctx, name, "creating", func() error {
		if pending != nil {
			slog.InfoContext(ctx, "MicroCloud waiting on existing creation operation", "instance", name, "operation", pending.Path)
			_, err := b.Client.waitForOperation(ctx, pending.Path)
			pending = nil
			errors.As(err, &pending)
			return err
		}
		if checkExisting {
			// Exact reads only: cluster list endpoints can lag or return empty
			// while the instance exists. A failed read never permits a POST.
			_, err := b.Client.get(ctx, "/1.0/instances/"+name)
			if err == nil {
				slog.InfoContext(ctx, "MicroCloud instance found after uncertain create; adopting", "project", b.Client.Project, "instance", name)
				return nil
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusNotFound {
				return err
			}
		}
		_, err := b.Client.post(ctx, "/1.0/instances", body)
		checkExisting = true
		errors.As(err, &pending)
		var apiErr *APIError
		if pending == nil && errors.As(err, &apiErr) && apiErr.AlreadyExists() {
			_, err = b.Client.get(ctx, "/1.0/instances/"+name)
		}
		return err
	})
}

func (b *Builder) startInstance(ctx context.Context, name string) error {
	var pending *pendingOperationError
	return b.retryInstanceStep(ctx, name, "starting", func() error {
		var err error
		if pending != nil {
			_, err = b.Client.waitForOperation(ctx, pending.Path)
			pending = nil
		} else {
			_, err = b.Client.put(ctx, "/1.0/instances/"+name+"/state", map[string]interface{}{
				"action": "start", "timeout": 60, "force": false,
			})
		}
		errors.As(err, &pending)
		var apiErr *APIError
		if pending == nil && errors.As(err, &apiErr) && apiErr.AlreadyRunning() {
			return nil
		}
		return err
	})
}
