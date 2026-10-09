package incus

import (
	"context"
	"encoding/json"
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
