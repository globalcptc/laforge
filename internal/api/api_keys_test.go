package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
)

// apiKeyClient signs a fresh person in and removes their keys afterwards.
func apiKeyClient(t *testing.T, env *apiTestEnv, who string) (*http.Client, string) {
	t.Helper()
	code := fmt.Sprintf("%s-%d", who, os.Getpid())
	client := signedInClient(t, env, code)
	login := "exchanged-" + code
	t.Cleanup(func() {
		env.server.Pool.Exec(context.Background(),
			`DELETE FROM api_key WHERE account_id = (SELECT id FROM account WHERE github_login = $1)`, login)
	})
	return client, login
}

func doJSON(t *testing.T, client *http.Client, method, url string, body any, out any) int {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, url, r)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("%s %s: decode %q: %v", method, url, raw, err)
		}
	}
	return resp.StatusCode
}

func TestAPIKeyLifecycle(t *testing.T) {
	env := setupAPITest(t)
	client, _ := apiKeyClient(t, env, "key-owner")

	var created struct {
		Key   map[string]any `json:"key"`
		Token string         `json:"token"`
	}
	status := doJSON(t, client, http.MethodPost, env.httpURL+"/auth/api-keys",
		map[string]any{"name": "  Claude on my laptop ", "expires_in_days": 30}, &created)
	if status != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201", status)
	}
	if !strings.HasPrefix(created.Token, "lfk_") || len(created.Token) != 4+43 {
		t.Fatalf("token = %q, want lfk_ + 43 base62 characters", created.Token)
	}
	if created.Key["name"] != "Claude on my laptop" || created.Key["status"] != "active" || created.Key["scope"] != "read" {
		t.Fatalf("created key = %v", created.Key)
	}
	if created.Key["prefix"] != created.Token[:12] {
		t.Fatalf("prefix = %v, want %q", created.Key["prefix"], created.Token[:12])
	}
	if _, ok := created.Key["token_hash"]; ok {
		t.Fatal("created key exposes token_hash")
	}

	// The list never carries the key itself or its hash.
	req, _ := http.NewRequest(http.MethodGet, env.httpURL+"/auth/api-keys", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: status = %d", resp.StatusCode)
	}
	if strings.Contains(string(raw), created.Token[12:]) || strings.Contains(string(raw), "token_hash") || strings.Contains(string(raw), hashToken(created.Token)) {
		t.Fatalf("list leaks the key: %s", raw)
	}
	var listed []map[string]any
	json.Unmarshal(raw, &listed)
	if len(listed) != 1 || listed[0]["id"] != created.Key["id"] {
		t.Fatalf("list = %s, want the one created key", raw)
	}

	var revoked map[string]any
	if status := doJSON(t, client, http.MethodDelete, env.httpURL+"/auth/api-keys/"+created.Key["id"].(string), nil, &revoked); status != http.StatusOK {
		t.Fatalf("revoke: status = %d, want 200", status)
	}
	if revoked["status"] != "revoked" {
		t.Fatalf("revoked status = %v", revoked["status"])
	}
	if status := doJSON(t, client, http.MethodDelete, env.httpURL+"/auth/api-keys/"+created.Key["id"].(string), nil, nil); status != http.StatusNotFound {
		t.Fatalf("revoke twice: status = %d, want 404", status)
	}
}

func TestAPIKeyCreateValidation(t *testing.T) {
	env := setupAPITest(t)
	client, login := apiKeyClient(t, env, "key-validate")
	url := env.httpURL + "/auth/api-keys"

	for name, body := range map[string]map[string]any{
		"no name":         {"name": "  ", "expires_in_days": 30},
		"long name":       {"name": strings.Repeat("x", 101), "expires_in_days": 30},
		"no expiry":       {"name": "k"},
		"odd expiry":      {"name": "k", "expires_in_days": 365},
		"bad repo id":     {"name": "k", "expires_in_days": 7, "repository_ids": []string{"nope"}},
		"unknown repo id": {"name": "k", "expires_in_days": 7, "repository_ids": []string{"00000000-0000-0000-0000-000000000001"}},
	} {
		if status := doJSON(t, client, http.MethodPost, url, body, nil); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, status)
		}
	}

	// A repository the person can read is accepted and kept on the key.
	var created struct {
		Key struct {
			RepositoryIDs []string `json:"repository_ids"`
		} `json:"key"`
	}
	body := map[string]any{"name": "k", "expires_in_days": 7, "repository_ids": []string{env.repo.ID.String(), env.repo.ID.String()}}
	if status := doJSON(t, client, http.MethodPost, url, body, &created); status != http.StatusCreated {
		t.Fatalf("readable repo: status = %d, want 201", status)
	}
	if len(created.Key.RepositoryIDs) != 1 || created.Key.RepositoryIDs[0] != env.repo.ID.String() {
		t.Fatalf("repository_ids = %v, want just %s", created.Key.RepositoryIDs, env.repo.ID)
	}

	// One the person can't read is refused: an admin's 'none' for them there.
	if _, err := env.server.Pool.Exec(context.Background(),
		`INSERT INTO repository_access (repository_id, account_id, level)
		 SELECT $1, id, 'none' FROM account WHERE github_login = $2`, env.repo.ID, login); err != nil {
		t.Fatal(err)
	}
	env.server.auth.forgetAccess()
	if status := doJSON(t, client, http.MethodPost, url, body, nil); status != http.StatusBadRequest {
		t.Fatalf("unreadable repo: status = %d, want 400", status)
	}
}

func TestAPIKeyActiveLimit(t *testing.T) {
	env := setupAPITest(t)
	client, _ := apiKeyClient(t, env, "key-limit")
	for i := 0; i < maxActiveAPIKeys; i++ {
		if status := doJSON(t, client, http.MethodPost, env.httpURL+"/auth/api-keys", map[string]any{"name": fmt.Sprint(i), "expires_in_days": 7}, nil); status != http.StatusCreated {
			t.Fatalf("key %d: status = %d", i, status)
		}
	}
	if status := doJSON(t, client, http.MethodPost, env.httpURL+"/auth/api-keys", map[string]any{"name": "one too many", "expires_in_days": 7}, nil); status != http.StatusConflict {
		t.Fatalf("over the limit: status = %d, want 409", status)
	}
}

// An API key is for the MCP server only: every REST route refuses it, and it
// is never forwarded to GitHub as if it were a GitHub token.
func TestAPIKeyRefusedOnREST(t *testing.T) {
	env := setupAPITest(t)
	client, _ := apiKeyClient(t, env, "key-rest")
	var created struct {
		Token string `json:"token"`
	}
	doJSON(t, client, http.MethodPost, env.httpURL+"/auth/api-keys", map[string]any{"name": "k", "expires_in_days": 7}, &created)
	env.gh.pushTokens[created.Token] = true // even a key GitHub would accept stays refused

	for _, path := range []string{"/repos/" + env.repo.ID.String() + "/builds", "/auth/api-keys"} {
		req, _ := http.NewRequest(http.MethodGet, env.httpURL+path, nil)
		req.Header.Set("Authorization", "Bearer "+created.Token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("GET %s with an API key: status = %d, want 401", path, resp.StatusCode)
		}
		if path != "/auth/api-keys" && !strings.Contains(string(raw), "MCP") {
			t.Fatalf("GET %s: error %s doesn't say keys are for MCP", path, raw)
		}
	}
	req, _ := http.NewRequest(http.MethodPost, env.httpURL+"/repos", strings.NewReader(`{"owner":"o","repo":"r"}`))
	req.Header.Set("Authorization", "Bearer "+created.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("POST /repos with an API key: status = %d, want 401", resp.StatusCode)
	}
}

// Key management takes the browser session only, not a bearer GitHub token.
func TestAPIKeyEndpointsNeedASession(t *testing.T) {
	env := setupAPITest(t)
	req, _ := http.NewRequest(http.MethodPost, env.httpURL+"/auth/api-keys", strings.NewReader(`{"name":"k","expires_in_days":7}`))
	req.Header.Set("Authorization", "Bearer good-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("create with a bearer GitHub token: status = %d, want 401", resp.StatusCode)
	}
}

func TestAPIKeyOwnershipAndAdmin(t *testing.T) {
	env := setupAPITest(t)
	owner, _ := apiKeyClient(t, env, "key-owner2")
	other, _ := apiKeyClient(t, env, "key-other")
	admin, adminLogin := apiKeyClient(t, env, "key-admin")

	var created struct {
		Key struct {
			ID string `json:"id"`
		} `json:"key"`
	}
	doJSON(t, owner, http.MethodPost, env.httpURL+"/auth/api-keys", map[string]any{"name": "mine", "expires_in_days": 7}, &created)

	// Someone else can neither see nor revoke it.
	var otherList []map[string]any
	doJSON(t, other, http.MethodGet, env.httpURL+"/auth/api-keys", nil, &otherList)
	if len(otherList) != 0 {
		t.Fatalf("another person's list = %v, want empty", otherList)
	}
	if status := doJSON(t, other, http.MethodDelete, env.httpURL+"/auth/api-keys/"+created.Key.ID, nil, nil); status != http.StatusNotFound {
		t.Fatalf("revoke someone else's key: status = %d, want 404", status)
	}
	if status := doJSON(t, other, http.MethodGet, env.httpURL+"/admin/api-keys", nil, nil); status != http.StatusForbidden {
		t.Fatalf("non-admin admin list: status = %d, want 403", status)
	}

	// An instance admin sees it with its owner, and can revoke it.
	env.server.AdminLogins = []string{adminLogin}
	var all []map[string]any
	if status := doJSON(t, admin, http.MethodGet, env.httpURL+"/admin/api-keys", nil, &all); status != http.StatusOK {
		t.Fatalf("admin list: status = %d", status)
	}
	found := false
	for _, k := range all {
		if k["id"] == created.Key.ID {
			found = true
			if !strings.HasPrefix(k["owner_login"].(string), "exchanged-key-owner2") {
				t.Fatalf("owner_login = %v", k["owner_login"])
			}
		}
		if _, ok := k["token_hash"]; ok {
			t.Fatal("admin list exposes token_hash")
		}
	}
	if !found {
		t.Fatal("admin list is missing the key")
	}
	var revoked map[string]any
	if status := doJSON(t, admin, http.MethodDelete, env.httpURL+"/admin/api-keys/"+created.Key.ID, nil, &revoked); status != http.StatusOK || revoked["status"] != "revoked" {
		t.Fatalf("admin revoke: status = %d, key = %v", status, revoked)
	}
}

func TestAPIKeyActivity(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()
	owner, _ := apiKeyClient(t, env, "activity-owner")
	other, _ := apiKeyClient(t, env, "activity-other")
	admin, adminLogin := apiKeyClient(t, env, "activity-admin")

	// One key each for owner and other, with a call recorded against each.
	keyFor := func(client *http.Client) string {
		var created struct {
			Key struct {
				ID string `json:"id"`
			} `json:"key"`
		}
		doJSON(t, client, http.MethodPost, env.httpURL+"/auth/api-keys", map[string]any{"name": "agent", "expires_in_days": 7}, &created)
		return created.Key.ID
	}
	// The owner has three calls, a second apart; other has one.
	ownerKey := keyFor(owner)
	for i, tool := range []string{"whoami", "get_build", "power"} {
		if _, err := env.server.Pool.Exec(ctx, `INSERT INTO mcp_audit (api_key_id, account_id, tool, args, status, duration_ms, detail, created_at)
			SELECT id, account_id, $2, '{"action":"reboot"}', 'ok', 12, '{"trace":[{"kind":"sql","name":"McpGetBuild"}]}', now() - make_interval(secs => $3)
			FROM api_key WHERE id = $1`, ownerKey, tool, 10-i); err != nil {
			t.Fatal(err)
		}
	}
	otherKey := keyFor(other)
	if _, err := env.server.Pool.Exec(ctx, `INSERT INTO mcp_audit (api_key_id, account_id, tool, args, status)
		SELECT id, account_id, 'list_steps', '{}', 'ok' FROM api_key WHERE id = $1`, otherKey); err != nil {
		t.Fatal(err)
	}

	type page struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
		Next  *string          `json:"next"`
	}
	var first page
	if status := doJSON(t, owner, http.MethodGet, env.httpURL+"/auth/api-keys/activity?limit=2", nil, &first); status != http.StatusOK {
		t.Fatalf("own activity: %d", status)
	}
	if first.Total != 3 || len(first.Items) != 2 || first.Items[0]["tool"] != "power" || first.Items[1]["tool"] != "get_build" || first.Next == nil {
		t.Fatalf("first page = %+v, want power then get_build, 3 in all, and a next cursor", first)
	}
	if first.Items[0]["key_name"] != "agent" || first.Items[0]["args"].(map[string]any)["action"] != "reboot" {
		t.Fatalf("row = %v", first.Items[0])
	}
	if _, ok := first.Items[0]["detail"]; ok {
		t.Fatal("the list carries detail; it should only come with one call")
	}
	var second page
	doJSON(t, owner, http.MethodGet, env.httpURL+"/auth/api-keys/activity?limit=2&before="+url.QueryEscape(*first.Next), nil, &second)
	if len(second.Items) != 1 || second.Items[0]["tool"] != "whoami" || second.Next != nil {
		t.Fatalf("second page = %+v, want just whoami and no next cursor", second)
	}
	if status := doJSON(t, owner, http.MethodGet, env.httpURL+"/auth/api-keys/activity?before=nonsense", nil, nil); status != http.StatusBadRequest {
		t.Fatalf("bad cursor: %d, want 400", status)
	}

	// Opening a row returns its detail -- for its owner only.
	callID := first.Items[0]["id"].(string)
	var one map[string]any
	if status := doJSON(t, owner, http.MethodGet, env.httpURL+"/auth/api-keys/activity/"+callID, nil, &one); status != http.StatusOK {
		t.Fatalf("own call detail: %d", status)
	}
	if one["detail"].(map[string]any)["trace"].([]any)[0].(map[string]any)["name"] != "McpGetBuild" {
		t.Fatalf("detail = %v", one["detail"])
	}
	if status := doJSON(t, other, http.MethodGet, env.httpURL+"/auth/api-keys/activity/"+callID, nil, nil); status != http.StatusNotFound {
		t.Fatalf("someone else's call: %d, want 404", status)
	}
	if status := doJSON(t, owner, http.MethodGet, env.httpURL+"/admin/api-keys/activity", nil, nil); status != http.StatusForbidden {
		t.Fatalf("non-admin admin activity: %d, want 403", status)
	}
	if status := doJSON(t, owner, http.MethodGet, env.httpURL+"/admin/api-keys/activity/"+callID, nil, nil); status != http.StatusForbidden {
		t.Fatalf("non-admin admin call detail: %d, want 403", status)
	}

	env.server.AdminLogins = []string{adminLogin}
	var all page
	if status := doJSON(t, admin, http.MethodGet, env.httpURL+"/admin/api-keys/activity?limit=200", nil, &all); status != http.StatusOK {
		t.Fatalf("admin activity: %d", status)
	}
	tools := map[string]bool{}
	for _, row := range all.Items {
		if strings.HasPrefix(row["owner_login"].(string), "exchanged-activity-") {
			tools[row["tool"].(string)] = true
		}
	}
	if !tools["power"] || !tools["list_steps"] || all.Total < 4 {
		t.Fatalf("admin activity is missing someone's calls: %v (total %d)", tools, all.Total)
	}
	if status := doJSON(t, admin, http.MethodGet, env.httpURL+"/admin/api-keys/activity/"+callID, nil, nil); status != http.StatusOK {
		t.Fatalf("admin call detail: %d", status)
	}
}
