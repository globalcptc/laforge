package ghclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func testClient(t *testing.T, apiHandler, authHandler http.Handler) *Client {
	c := New()
	c.sleep = func(time.Duration) {} // no real waiting in tests
	if apiHandler != nil {
		s := httptest.NewServer(apiHandler)
		t.Cleanup(s.Close)
		c.APIBaseURL = s.URL
	}
	if authHandler != nil {
		s := httptest.NewServer(authHandler)
		t.Cleanup(s.Close)
		c.AuthBaseURL = s.URL
	}
	return c
}

func TestGetRepo(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/repos/globalcptc/laforge"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Authorization"), "Bearer test-token"; got != want {
			t.Errorf("Authorization = %q, want %q", got, want)
		}
		json.NewEncoder(w).Encode(Repo{
			FullName: "globalcptc/laforge", DefaultBranch: "main",
			Permissions: RepoPermissions{Push: true, Pull: true},
		})
	}), nil)

	repo, err := c.GetRepo(context.Background(), "test-token", "globalcptc", "laforge")
	if err != nil {
		t.Fatalf("GetRepo: %v", err)
	}
	if repo.FullName != "globalcptc/laforge" || !repo.Permissions.Push {
		t.Fatalf("GetRepo = %+v, want push permission on globalcptc/laforge", repo)
	}
}

func TestGetRepoNotFoundIsAPIError(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found"}`))
	}), nil)

	_, err := c.GetRepo(context.Background(), "test-token", "nobody", "nothing")
	if err == nil {
		t.Fatal("expected an error for a 404")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestGetCombinedStatusAndCheckRuns(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r/commits/abc123/status":
			json.NewEncoder(w).Encode(CombinedStatus{State: "success", TotalCount: 1})
		case "/repos/o/r/commits/abc123/check-runs":
			json.NewEncoder(w).Encode(checkRunsResponse{
				TotalCount: 2,
				CheckRuns: []CheckRun{
					{Name: "build", Status: "completed", Conclusion: "success"},
					{Name: "test", Status: "completed", Conclusion: "failure"},
				},
			})
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}), nil)

	st, err := c.GetCombinedStatus(context.Background(), "tok", "o", "r", "abc123")
	if err != nil {
		t.Fatalf("GetCombinedStatus: %v", err)
	}
	if st.State != "success" {
		t.Fatalf("State = %q, want success", st.State)
	}

	runs, err := c.ListCheckRuns(context.Background(), "tok", "o", "r", "abc123")
	if err != nil {
		t.Fatalf("ListCheckRuns: %v", err)
	}
	if len(runs) != 2 || runs[1].Conclusion != "failure" {
		t.Fatalf("ListCheckRuns = %+v, want 2 runs with the second failing", runs)
	}
}

func TestCreateStatus(t *testing.T) {
	var gotBody map[string]string
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if got, want := r.URL.Path, "/repos/o/r/statuses/deadbeef"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}), nil)

	err := c.CreateStatus(context.Background(), "tok", "o", "r", "deadbeef", "success", "check passed", "laforge/validate")
	if err != nil {
		t.Fatalf("CreateStatus: %v", err)
	}
	if gotBody["state"] != "success" || gotBody["context"] != "laforge/validate" {
		t.Fatalf("posted body = %+v, want state=success context=laforge/validate", gotBody)
	}
}

func TestDeviceFlowHappyPath(t *testing.T) {
	pollCount := 0
	c := testClient(t, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/device/code":
			json.NewEncoder(w).Encode(DeviceCodeResponse{
				DeviceCode: "devcode123", UserCode: "ABCD-1234",
				VerificationURI: "https://github.com/login/device",
				ExpiresIn:       900, Interval: 1,
			})
		case "/login/oauth/access_token":
			pollCount++
			if pollCount < 3 {
				json.NewEncoder(w).Encode(deviceTokenResponse{Error: "authorization_pending"})
				return
			}
			json.NewEncoder(w).Encode(deviceTokenResponse{AccessToken: "gho_realtoken"})
		default:
			t.Errorf("unexpected auth path %q", r.URL.Path)
		}
	}))

	dc, err := c.RequestDeviceCode(context.Background(), "client-id", []string{"repo"})
	if err != nil {
		t.Fatalf("RequestDeviceCode: %v", err)
	}
	if dc.DeviceCode != "devcode123" || dc.UserCode != "ABCD-1234" {
		t.Fatalf("RequestDeviceCode = %+v", dc)
	}

	tok, err := c.PollForToken(context.Background(), "client-id", dc.DeviceCode, dc.Interval, dc.ExpiresIn)
	if err != nil {
		t.Fatalf("PollForToken: %v", err)
	}
	if tok.AccessToken != "gho_realtoken" {
		t.Fatalf("token = %+v, want access gho_realtoken", tok)
	}
	if pollCount != 3 {
		t.Fatalf("polled %d times, want exactly 3 (2 pending + 1 success)", pollCount)
	}
}

func TestDeviceFlowSlowDownThenSuccess(t *testing.T) {
	pollCount := 0
	c := testClient(t, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login/oauth/access_token" {
			return
		}
		pollCount++
		switch pollCount {
		case 1:
			json.NewEncoder(w).Encode(deviceTokenResponse{Error: "slow_down"})
		default:
			json.NewEncoder(w).Encode(deviceTokenResponse{AccessToken: "gho_final"})
		}
	}))

	tok, err := c.PollForToken(context.Background(), "client-id", "dev123", 1, 900)
	if err != nil {
		t.Fatalf("PollForToken: %v", err)
	}
	if tok.AccessToken != "gho_final" {
		t.Fatalf("token = %+v, want access gho_final", tok)
	}
}

func TestDeviceFlowAccessDenied(t *testing.T) {
	c := testClient(t, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(deviceTokenResponse{Error: "access_denied"})
	}))

	_, err := c.PollForToken(context.Background(), "client-id", "dev123", 1, 900)
	if err != ErrAuthorizationDenied {
		t.Fatalf("err = %v, want ErrAuthorizationDenied", err)
	}
}

func TestDeviceFlowExpired(t *testing.T) {
	c := testClient(t, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(deviceTokenResponse{Error: "authorization_pending"})
	}))

	// expiresIn=0 means the deadline (now + 0s) is already passed by the
	// time the first poll would fire, so this exercises the expiry branch
	// deterministically instead of racing a real clock.
	_, err := c.PollForToken(context.Background(), "client-id", "dev123", 1, 0)
	if err != ErrDeviceCodeExpired {
		t.Fatalf("err = %v, want ErrDeviceCodeExpired", err)
	}
}

func TestDeviceFlowContextCancellation(t *testing.T) {
	c := testClient(t, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(deviceTokenResponse{Error: "authorization_pending"})
	}))
	// A real (non-stubbed) sleep would block forever waiting for the
	// interval; cancel the context immediately so this proves the poll
	// loop actually respects it rather than looping until expiry.
	c.sleep = func(time.Duration) {}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.PollForToken(ctx, "client-id", "dev123", 1, 900)
	if err != context.Canceled {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestAuthorizeURL(t *testing.T) {
	c := New()
	got := c.AuthorizeURL("client-id", "https://laforge.example/callback", "state-xyz", []string{"repo", "read:user"})
	want := "https://github.com/login/oauth/authorize?client_id=client-id&redirect_uri=https%3A%2F%2Flaforge.example%2Fcallback&scope=repo+read%3Auser&state=state-xyz"
	if got != want {
		t.Fatalf("AuthorizeURL = %q, want %q", got, want)
	}
}

func TestExchangeCodeHappyPath(t *testing.T) {
	var gotForm url.Values
	c := testClient(t, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login/oauth/access_token" {
			t.Errorf("unexpected auth path %q", r.URL.Path)
		}
		r.ParseForm()
		gotForm = r.PostForm
		json.NewEncoder(w).Encode(deviceTokenResponse{AccessToken: "gho_webtoken", RefreshToken: "ghr_refresh", ExpiresIn: 28800, RefreshTokenExpiresIn: 15897600})
	}))

	tok, err := c.ExchangeCode(context.Background(), "client-id", "client-secret", "the-code", "https://laforge.example/callback")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if tok.AccessToken != "gho_webtoken" || tok.RefreshToken != "ghr_refresh" || tok.ExpiresIn != 28800 {
		t.Fatalf("token = %+v, want access gho_webtoken + refresh ghr_refresh + expires 28800", tok)
	}
	if gotForm.Get("client_id") != "client-id" || gotForm.Get("client_secret") != "client-secret" ||
		gotForm.Get("code") != "the-code" || gotForm.Get("redirect_uri") != "https://laforge.example/callback" {
		t.Fatalf("posted form = %+v, missing/wrong fields", gotForm)
	}
}

func TestExchangeCodeRejected(t *testing.T) {
	c := testClient(t, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(deviceTokenResponse{Error: "bad_verification_code", ErrorDesc: "The code passed is incorrect or expired."})
	}))

	_, err := c.ExchangeCode(context.Background(), "client-id", "client-secret", "wrong-code", "https://laforge.example/callback")
	if err == nil {
		t.Fatal("ExchangeCode: expected an error for a rejected code, got nil")
	}
}

func TestRefreshUserToken(t *testing.T) {
	var gotForm url.Values
	c := testClient(t, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login/oauth/access_token" {
			t.Errorf("unexpected auth path %q", r.URL.Path)
		}
		r.ParseForm()
		gotForm = r.PostForm
		json.NewEncoder(w).Encode(deviceTokenResponse{AccessToken: "gho_new", RefreshToken: "ghr_rotated", ExpiresIn: 28800, RefreshTokenExpiresIn: 15897600})
	}))

	tok, err := c.RefreshUserToken(context.Background(), "client-id", "client-secret", "ghr_old")
	if err != nil {
		t.Fatalf("RefreshUserToken: %v", err)
	}
	if tok.AccessToken != "gho_new" || tok.RefreshToken != "ghr_rotated" {
		t.Fatalf("token = %+v, want a rotated gho_new/ghr_rotated", tok)
	}
	if gotForm.Get("grant_type") != "refresh_token" || gotForm.Get("refresh_token") != "ghr_old" {
		t.Fatalf("posted form = %+v, want grant_type=refresh_token & refresh_token=ghr_old", gotForm)
	}
}

func TestListBranchesPaginates(t *testing.T) {
	pages := 0
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		page := r.URL.Query().Get("page")
		if r.URL.Query().Get("per_page") != "100" {
			t.Errorf("per_page = %q, want 100", r.URL.Query().Get("per_page"))
		}
		var out []Branch
		if page == "1" {
			for i := 0; i < 100; i++ {
				out = append(out, Branch{Name: "branch-1-" + string(rune('a'+i%26))})
			}
		} else {
			out = []Branch{{Name: "main"}}
		}
		json.NewEncoder(w).Encode(out)
	}), nil)

	branches, err := c.ListBranches(context.Background(), "test-token", "globalcptc", "laforge")
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	if len(branches) != 101 {
		t.Fatalf("len(branches) = %d, want 101 (100 on page 1, 1 more on page 2)", len(branches))
	}
	if pages != 2 {
		t.Fatalf("pages fetched = %d, want 2 (stopped once a page came back short)", pages)
	}
}

func TestGetBranch(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/repos/globalcptc/laforge/branches/main"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		var b Branch
		b.Name = "main"
		b.Commit.SHA = "deadbeef"
		json.NewEncoder(w).Encode(b)
	}), nil)

	b, err := c.GetBranch(context.Background(), "test-token", "globalcptc", "laforge", "main")
	if err != nil {
		t.Fatalf("GetBranch: %v", err)
	}
	if b.Name != "main" || b.Commit.SHA != "deadbeef" {
		t.Fatalf("GetBranch = %+v, want name=main commit.sha=deadbeef", b)
	}
}
