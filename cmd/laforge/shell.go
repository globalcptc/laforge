package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/coder/websocket"
	"golang.org/x/term"
)

// runShell opens a fully interactive root/admin shell on one host (or
// container) through its agent, relayed over an encrypted WebSocket to the
// laforge-api. The local terminal is put in raw mode so keys, colors, Ctrl-C,
// and full-screen programs (vim, top) all work; the remote PTY is resized to
// match. Needs manage access on the repo (same as `laforge run`). Only one or
// two shells may run across the instance at once.
func runShell(args []string) error {
	fs := flag.NewFlagSet("shell", flag.ExitOnError)
	build := fs.String("build", "", "build id (required)")
	id := fs.String("id", "", "deployed object id to open a shell on (exact)")
	team := fs.Int("team", -1, "team number (with --host, to resolve a single object)")
	host := fs.String("host", "", "host/container name -- its `as` name (with --team)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *build == "" {
		return fmt.Errorf("usage: laforge shell --build <id> (--id <object-id> | --team <n> --host <as>)")
	}

	objectID := *id
	if objectID == "" {
		if *team < 0 || *host == "" {
			return fmt.Errorf("give either --id <object-id>, or both --team <n> and --host <as>")
		}
		resolved, err := resolveShellObject(*build, *team, *host)
		if err != nil {
			return err
		}
		objectID = resolved
	}

	token, _ := readStoredToken()
	if token == "" {
		return fmt.Errorf("not signed in -- run `laforge login` first")
	}
	wsURL, err := terminalWebSocketURL(*build, objectID)
	if err != nil {
		return err
	}

	// Raw mode so every keystroke (Ctrl-C, arrows, tab) goes to the remote
	// shell, not the local one. Restored on any exit path.
	stdinFd := int(os.Stdin.Fd())
	if !term.IsTerminal(stdinFd) {
		return fmt.Errorf("laforge shell needs an interactive terminal on stdin")
	}
	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		return fmt.Errorf("putting terminal in raw mode: %w", err)
	}
	defer term.Restore(stdinFd, oldState)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: map[string][]string{"Authorization": {"Bearer " + token}},
	})
	if err != nil {
		term.Restore(stdinFd, oldState)
		return fmt.Errorf("opening shell: %w", err)
	}
	defer c.Close(websocket.StatusNormalClosure, "bye")
	c.SetReadLimit(1 << 20)

	// Initial size + live resizes (SIGWINCH on unix; a no-op elsewhere).
	sendResize := func() {
		if cols, rows, err := term.GetSize(stdinFd); err == nil {
			msg, _ := json.Marshal(map[string]any{"type": "resize", "cols": cols, "rows": rows})
			_ = c.Write(ctx, websocket.MessageText, msg)
		}
	}
	sendResize()
	stopResize := watchResize(sendResize)
	defer stopResize()

	// stdin -> server (binary). A read error (EOF, or the terminal closing)
	// ends the session.
	go func() {
		defer cancel()
		buf := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				if werr := c.Write(ctx, websocket.MessageBinary, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// server -> stdout. Ends when the shell exits or the connection drops.
	for {
		typ, data, err := c.Read(ctx)
		if err != nil {
			break
		}
		if typ == websocket.MessageBinary {
			if _, werr := os.Stdout.Write(data); werr != nil {
				break
			}
		}
	}
	term.Restore(stdinFd, oldState)
	fmt.Fprintln(os.Stderr, "\nlaforge: shell closed")
	return nil
}

// resolveShellObject finds the single deployed object matching team+host in a
// build, erroring if none or more than one match (host `as` names repeat across
// teams, so team is required to disambiguate).
func resolveShellObject(build string, team int, host string) (string, error) {
	var objs []struct {
		ID         string `json:"id"`
		ObjectName string `json:"object_name"`
		AsName     string `json:"as_name"`
		Kind       string `json:"kind"`
		TeamNumber int    `json:"team_number"`
	}
	if err := newAPIClient().do("GET", "/builds/"+url.PathEscape(build)+"/objects", nil, &objs); err != nil {
		return "", err
	}
	var matches []string
	for _, o := range objs {
		if o.TeamNumber != team || (o.Kind != "host" && o.Kind != "container") {
			continue
		}
		if o.AsName == host || o.ObjectName == host {
			matches = append(matches, o.ID)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("no host or container named %q in team %d", host, team)
	default:
		return "", fmt.Errorf("%q is ambiguous in team %d (%d matches) -- use --id", host, team, len(matches))
	}
}

// terminalWebSocketURL turns the configured api base URL into the ws(s):// URL
// for a host's terminal endpoint.
func terminalWebSocketURL(build, objectID string) (string, error) {
	base := os.Getenv("LAFORGE_API_URL")
	if base == "" {
		base = "http://localhost:8080"
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid LAFORGE_API_URL %q: %w", base, err)
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/builds/" + url.PathEscape(build) + "/objects/" + url.PathEscape(objectID) + "/terminal"
	return u.String(), nil
}
