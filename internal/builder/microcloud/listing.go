package microcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"sync"
)

// listNames reads a collection without recursion -- just its members' names.
// The server answers that cheaply however large the collection is, whereas
// recursion=1 makes it work out every member's full record first (for a
// network, everything that uses it), which on a cluster with hundreds of
// networks or instances can outlast the request.
func (c *Client) listNames(ctx context.Context, collection string) ([]string, error) {
	raw, err := c.get(ctx, collection)
	if err != nil {
		return nil, err
	}
	var urls []string
	if err := json.Unmarshal(raw, &urls); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(urls))
	for _, u := range urls {
		p, _, _ := strings.Cut(u, "?") // "/1.0/networks/UPLINK?project=x"
		name, err := url.PathUnescape(path.Base(p))
		if err != nil || name == "" || name == "." || name == "/" {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}

// forEachConcurrently calls fn for every item, at most limit at a time.
func forEachConcurrently(items []string, limit int, fn func(i int, item string)) {
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	for i, item := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, item string) {
			defer wg.Done()
			defer func() { <-sem }()
			fn(i, item)
		}(i, item)
	}
	wg.Wait()
}

// detailConcurrency bounds the per-member requests a listing makes in
// parallel -- kept low because the cluster may be shared.
const detailConcurrency = 8

// maxNetworkDetails bounds how many networks ListNetworks reads in full.
const maxNetworkDetails = 40

// laforgeInstances reads LaForge's own instances (named lf-…) in the client's
// project. In default it reads them one request each, at most
// detailConcurrency at a time -- never every instance in one recursive
// listing. On a shared cluster that listing grows
// with everyone else's instances, and it's polled throughout an event. An
// instance deleted between the two steps is skipped; any other failure fails
// the whole read, since callers (drift, power state) would otherwise treat a
// missing instance as gone.
//
// In a project of LaForge's own (anything but default) the project holds only
// LaForge's instances, so one recursive listing is the cheaper way to read
// them all -- this is polled every 30 seconds during an event.
func (c *Client) laforgeInstances(ctx context.Context) ([]json.RawMessage, error) {
	if c.Project != "" && c.Project != "default" {
		raw, err := c.get(ctx, "/1.0/instances?recursion=1")
		if err != nil {
			return nil, err
		}
		var all []json.RawMessage
		if err := json.Unmarshal(raw, &all); err != nil {
			return nil, err
		}
		var ours []json.RawMessage
		for _, r := range all {
			var named struct {
				Name string `json:"name"`
			}
			if json.Unmarshal(r, &named) == nil && strings.HasPrefix(named.Name, "lf-") {
				ours = append(ours, r)
			}
		}
		return ours, nil
	}
	names, err := c.listNames(ctx, "/1.0/instances")
	if err != nil {
		return nil, err
	}
	var ours []string
	for _, n := range names {
		if strings.HasPrefix(n, "lf-") {
			ours = append(ours, n)
		}
	}
	raws := make([]json.RawMessage, len(ours))
	errs := make([]error, len(ours))
	forEachConcurrently(ours, detailConcurrency, func(i int, name string) {
		raw, err := c.get(ctx, "/1.0/instances/"+url.PathEscape(name))
		var apiErr *APIError
		if err != nil && !(errors.As(err, &apiErr) && apiErr.NotFound()) {
			errs[i] = fmt.Errorf("reading instance %s: %w", name, err)
			return
		}
		raws[i] = raw
	})
	out := make([]json.RawMessage, 0, len(raws))
	for i, r := range raws {
		if errs[i] != nil {
			return nil, errs[i]
		}
		if r != nil {
			out = append(out, r)
		}
	}
	return out, nil
}

// fetchDetails reads each named item at path(name), at most
// detailConcurrency at a time, decoding into T. Items that fail are left out;
// the first failure is returned alongside whatever succeeded, so a caller can
// show what it could read and say what it couldn't.
func fetchDetails[T any](ctx context.Context, c *Client, names []string, path func(string) string) ([]T, error) {
	results := make([]*T, len(names))
	var mu sync.Mutex
	var firstErr error
	forEachConcurrently(names, detailConcurrency, func(i int, name string) {
		raw, err := c.get(ctx, path(name))
		if err == nil {
			var v T
			if err = json.Unmarshal(raw, &v); err == nil {
				results[i] = &v
				return
			}
		}
		mu.Lock()
		if firstErr == nil {
			firstErr = fmt.Errorf("%s: %w", name, err)
		}
		mu.Unlock()
	})
	out := make([]T, 0, len(results))
	for _, r := range results {
		if r != nil {
			out = append(out, *r)
		}
	}
	return out, firstErr
}
