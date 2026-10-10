package main

import (
	"encoding/base64"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMainBootsAndServes starts the real server on a scratch port with a
// throwaway database and probes the routes whose handlers live in main
// itself: the health check, robots.txt, the body-limit guard in both of
// its rejection modes, the compression skip predicate and both CSRF token
// extractors. The Listen error branch stays uncovered on purpose: the
// port is free, so Listen never returns during the test.
func TestMainBootsAndServes(t *testing.T) {
	t.Setenv("DATABASE_URL", filepath.Join(t.TempDir(), "boot.db"))
	t.Setenv("PORT", "18199")
	t.Setenv("ADMIN_USER", "admin")
	t.Setenv("ADMIN_PASSWORD", "secret")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.1, 10.0.0.2")
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("SPOTIFY_CLIENT_ID", "")
	t.Setenv("SPOTIFY_CLIENT_SECRET", "")
	t.Setenv("SPOTIFY_REFRESH_TOKEN", "")

	go main()

	base := "http://127.0.0.1:18199"
	client := &http.Client{Timeout: 5 * time.Second}

	deadline := time.Now().Add(30 * time.Second)
	for {
		resp, err := client.Get(base + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("server did not come up on the scratch port")
		}
		time.Sleep(50 * time.Millisecond)
	}

	get := func(path string) (int, string) {
		t.Helper()
		resp, err := client.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode, string(body)
	}

	if status, body := get("/robots.txt"); status != 200 || !strings.Contains(body, "User-agent: *") {
		t.Errorf("robots.txt: status %d body %q", status, body)
	}
	// unconfigured site: the homepage renders the setup guard
	if status, _ := get("/"); status != 200 {
		t.Errorf("home: status %d", status)
	}
	// media ids are rejected before any database lookup happens, and the
	// path exercises the compression skip predicate
	if status, _ := get("/media/1"); status != 404 {
		t.Errorf("media: status %d, want 404", status)
	}
	// no route matches a multi-segment path: the fiber 404 goes through
	// the configured error handler
	if status, _ := get("/no/such/path"); status != 404 {
		t.Errorf("unknown path: status %d, want 404", status)
	}

	post := func(path string, body io.Reader, contentType string, headers map[string]string) int {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, base+path, body)
		if err != nil {
			t.Fatal(err)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode
	}

	// CSRF extractor, header branch: a bogus header token is rejected
	if status := post("/gift/claim", strings.NewReader(`{"amount":1}`), "application/json",
		map[string]string{"X-Csrf-Token": "bogus"}); status != 403 {
		t.Errorf("claim with header token: status %d, want 403", status)
	}
	// CSRF extractor, form branch: a bogus form token is rejected too
	if status := post("/gift/claim", strings.NewReader("_csrf=bogus"), "application/x-www-form-urlencoded",
		nil); status != 403 {
		t.Errorf("claim with form token: status %d, want 403", status)
	}
	// body limit: an oversized declared length is rejected before CSRF
	if status := post("/gift/claim", strings.NewReader(strings.Repeat("x", 300*1024)), "application/json",
		nil); status != 413 {
		t.Errorf("oversized claim: status %d, want 413", status)
	}
	// body limit: an unknown (chunked) length is rejected conservatively;
	// an io.Pipe body makes the client use chunked transfer encoding
	pipeReader, pipeWriter := io.Pipe()
	go func() {
		_, _ = pipeWriter.Write([]byte("chunk"))
		_ = pipeWriter.Close()
	}()
	if status := post("/gift/claim", pipeReader, "application/json", nil); status != 413 {
		t.Errorf("chunked claim: status %d, want 413", status)
	}

	// the dashboard group: basic auth first, then the admin CSRF
	// extractor in both branches
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret"))
	if status := post("/dashboard/guests", strings.NewReader("first_name=X"), "application/x-www-form-urlencoded",
		map[string]string{"Authorization": auth, "X-Csrf-Token": "bogus"}); status != 403 {
		t.Errorf("dashboard with header token: status %d, want 403", status)
	}
	if status := post("/dashboard/guests", strings.NewReader("_csrf=bogus&first_name=X"), "application/x-www-form-urlencoded",
		map[string]string{"Authorization": auth}); status != 403 {
		t.Errorf("dashboard with form token: status %d, want 403", status)
	}
	// without credentials the group answers 401 before anything else
	if status := post("/dashboard/guests", strings.NewReader("first_name=X"), "application/x-www-form-urlencoded",
		nil); status != 401 {
		t.Errorf("dashboard without auth: status %d, want 401", status)
	}
}
