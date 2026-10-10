package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/models"
	"github.com/streambinder/foedus/internal/spotify"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func swapChatClient(t *testing.T, transport http.RoundTripper) {
	t.Helper()
	original := chatHTTPClient
	chatHTTPClient = &http.Client{Transport: transport}
	t.Cleanup(func() { chatHTTPClient = original })
}

func resetChatLimiter() {
	chatRateLimiter.Range(func(key, _ any) bool {
		chatRateLimiter.Delete(key)
		return true
	})
}

func resetSoundtrackLimiter() {
	soundtrackRateLimiter.Range(func(key, _ any) bool {
		soundtrackRateLimiter.Delete(key)
		return true
	})
}

func sseResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func seedChatWorld(t *testing.T) {
	t.Helper()
	configureSettings(t, nil)
	if err := database.ReplaceImpersonations(database.DB, []models.Impersonation{
		{Codename: "nonna Ada", Profile: "Grandmother"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := database.ReplacePlaces(database.DB, models.PlaceKindStory, []models.Place{
		{Label: "2019", Name: "Bar Centrale", Address: "Via X"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := database.ReplaceAccommodations(database.DB, []models.Accommodation{
		{Name: "Hotel Uno", Description: "Near", URL: "https://example.com"},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestChatStreamGuards(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	app := newTestApp()
	defer app.Shutdown()
	resetChatLimiter()

	// no API key configured
	InitChat("", "")
	status, _, _ := postJSON(t, app, "/chat", map[string]any{"message": "ciao"})
	if status != 404 {
		t.Errorf("chat without key: status %d, want 404", status)
	}

	InitChat("test-key", "test-model")
	// malformed JSON, empty and over-long messages, over-long history
	status, _, _ = postRaw(t, app, "/chat", fiber.MIMEApplicationJSON, "{broken")
	if status != 400 {
		t.Errorf("chat bad json: status %d", status)
	}
	status, _, _ = postJSON(t, app, "/chat", map[string]any{"message": "   "})
	if status != 400 {
		t.Errorf("chat blank message: status %d", status)
	}
	status, _, _ = postJSON(t, app, "/chat", map[string]any{"message": strings.Repeat("a", chatMaxMessageLen+1)})
	if status != 400 {
		t.Errorf("chat long message: status %d", status)
	}
	history := make([]map[string]string, chatMaxHistoryInReq+1)
	for i := range history {
		history[i] = map[string]string{"role": "user", "content": "x"}
	}
	status, _, _ = postJSON(t, app, "/chat", map[string]any{"message": "ciao", "history": history})
	if status != 400 {
		t.Errorf("chat long history: status %d", status)
	}

	// settings in place but no impersonations exist
	status, _, _ = postJSON(t, app, "/chat", map[string]any{"message": "ciao"})
	if status != 404 {
		t.Errorf("chat without personas: status %d, want 404", status)
	}

	// after the event the chat closes
	configureSettings(t, func(s *models.Settings) { s.CeremonyDatetime = "2020-01-01T10:00" })
	status, _, _ = postJSON(t, app, "/chat", map[string]any{"message": "ciao"})
	if status != 403 {
		t.Errorf("chat after event: status %d, want 403", status)
	}
}

func TestChatStreamRateLimit(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	app := newTestApp()
	defer app.Shutdown()
	InitChat("test-key", "test-model")
	resetChatLimiter()

	for i := 0; i < chatRateLimit; i++ {
		status, _, _ := postJSON(t, app, "/chat", map[string]any{"message": "ciao"})
		if status == 429 {
			t.Fatalf("request %d rate limited too early", i)
		}
	}
	status, _, _ := postJSON(t, app, "/chat", map[string]any{"message": "ciao"})
	if status != 429 {
		t.Errorf("over-limit request: status %d, want 429", status)
	}
}

func TestChatStreamSuccess(t *testing.T) {
	setupDB(t)
	seedChatWorld(t)
	app := newTestApp()
	defer app.Shutdown()
	InitChat("test-key", "test-model")
	resetChatLimiter()

	var upstreamBody string
	swapChatClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read upstream body: %v", err)
		}
		upstreamBody = string(raw)
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("upstream auth header = %q", got)
		}
		return sseResponse(200, ": keepalive\n"+
			"data: {\"choices\":[{\"delta\":{\"content\":\"Ciao\"}}]}\n\n"+
			"data: {not-json}\n\n"+
			"data: {\"choices\":[{\"delta\":{\"content\":\" a tutti\"}}]}\n\n"+
			"data: {\"choices\":[{\"delta\":{}}]}\n\n"+
			"data: [DONE]\n\n"), nil
	}))

	payload := map[string]any{
		"message": "  come stai? ",
		"history": []map[string]string{
			{"role": "user", "content": "prima"},
			{"role": "assistant", "content": ""},
			{"role": "system", "content": "ignored role"},
		},
	}
	status, headers, body := postJSON(t, app, "/chat", payload)
	if status != 200 {
		t.Fatalf("chat stream: status %d body %q", status, body)
	}
	if ct := headers.Get(fiber.HeaderContentType); !strings.Contains(ct, "text/event-stream") {
		t.Errorf("content type = %q", ct)
	}
	// data lines are forwarded verbatim, including the malformed one
	if !strings.Contains(body, `"content":"Ciao"`) || !strings.Contains(body, `"content":" a tutti"`) {
		t.Errorf("streamed body = %q", body)
	}
	if !strings.Contains(body, "[DONE]") || strings.Contains(body, `"error"`) {
		t.Errorf("stream ending = %q", body)
	}
	// the trimmed message and the history reach the upstream request
	if !strings.Contains(upstreamBody, "come stai?") || !strings.Contains(upstreamBody, "Nonna Ada") {
		t.Errorf("upstream body = %q", upstreamBody)
	}
	if !strings.Contains(upstreamBody, "Bar Centrale") || !strings.Contains(upstreamBody, "Hotel Uno") {
		t.Errorf("upstream context incomplete: %q", upstreamBody)
	}
}

type errAfterReader struct {
	data []byte
	done bool
}

func (r *errAfterReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, errors.New("stream broke")
	}
	r.done = true
	return copy(p, r.data), nil
}

func (r *errAfterReader) Close() error { return nil }

func TestChatStreamUpstreamFailures(t *testing.T) {
	setupDB(t)
	seedChatWorld(t)
	app := newTestApp()
	defer app.Shutdown()
	InitChat("test-key", "test-model")

	cases := []struct {
		name      string
		transport http.RoundTripper
	}{
		{"transport error", roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		})},
		{"upstream 500", roundTripFunc(func(*http.Request) (*http.Response, error) {
			return sseResponse(500, "server error"), nil
		})},
		{"panicking transport", roundTripFunc(func(*http.Request) (*http.Response, error) {
			panic("boom")
		})},
		{"broken stream", roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Header:     make(http.Header),
				Body:       &errAfterReader{data: []byte("data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")},
			}, nil
		})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetChatLimiter()
			swapChatClient(t, tc.transport)
			status, _, body := postJSON(t, app, "/chat", map[string]any{"message": "ciao"})
			if status != 200 {
				t.Fatalf("status %d body %q", status, body)
			}
			if !strings.Contains(body, `"error":`) {
				t.Errorf("body = %q, want an error frame", body)
			}
		})
	}
}

func TestChatStreamDataErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		drop string
	}{
		{"impersonations", "impersonations"},
		{"places", "places"},
		{"accommodations", "accommodations"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupDB(t)
			seedChatWorld(t)
			app := newTestApp()
			defer app.Shutdown()
			InitChat("test-key", "test-model")
			resetChatLimiter()
			if _, err := database.DB.Exec(`DROP TABLE ` + tc.drop); err != nil {
				t.Fatal(err)
			}
			status, _, _ := postJSON(t, app, "/chat", map[string]any{"message": "ciao"})
			if status != 500 {
				t.Errorf("status %d, want 500", status)
			}
		})
	}

	t.Run("settings", func(t *testing.T) {
		setupDB(t)
		app := newTestApp()
		defer app.Shutdown()
		InitChat("test-key", "test-model")
		resetChatLimiter()
		database.DB.Close()
		status, _, _ := postJSON(t, app, "/chat", map[string]any{"message": "ciao"})
		if status != 500 {
			t.Errorf("status %d, want 500", status)
		}
	})
}

func TestChatHelpers(t *testing.T) {
	resetChatLimiter()
	ip := "203.0.113.7"
	for i := 0; i < chatRateLimit; i++ {
		if !checkRateLimit(ip) {
			t.Fatalf("request %d rejected", i)
		}
	}
	if checkRateLimit(ip) {
		t.Error("request over the limit accepted")
	}
	if !checkRateLimit("198.51.100.9") {
		t.Error("a different ip must have its own quota")
	}
	// expired entries no longer count
	old := make([]time.Time, chatRateLimit)
	for i := range old {
		old[i] = time.Now().Add(-2 * time.Minute)
	}
	chatRateLimiter.Store("192.0.2.5", old)
	if !checkRateLimit("192.0.2.5") {
		t.Error("expired window still blocks the ip")
	}

	// persona names: first rune upper-cased, the rest untouched
	if got := capitalizedPersonaName(""); got != "" {
		t.Errorf("empty persona = %q", got)
	}
	if got := capitalizedPersonaName("nonna"); got != "Nonna" {
		t.Errorf("persona = %q", got)
	}
	if got := capitalizedPersonaName("Élise"); got != "Élise" {
		t.Errorf("unicode persona = %q", got)
	}
	if got := capitalizedPersonaName("a"); got != "A" {
		t.Errorf("single rune persona = %q", got)
	}

	// accept-language mapping: the first two letters of the first long
	// enough tag win; anything else falls back to english
	for header, want := range map[string]string{
		"it": "it", "it-IT,it;q=0.9": "it", "en-US": "en", "fr": "fr", "": "en", "x": "en",
	} {
		if got := i18nLangFromAccept(header); got != want {
			t.Errorf("i18nLangFromAccept(%q) = %q, want %q", header, got, want)
		}
	}

	// conversation formatting: known roles with content only, numbered in
	// their original positions
	got := formatConversationContext([]chatMessage{
		{Role: "user", Content: "ciao"},
		{Role: "assistant", Content: "risposta"},
		{Role: "system", Content: "skip"},
		{Role: "user", Content: ""},
	})
	if !strings.Contains(got, "1. [user][already handled] ciao") || !strings.Contains(got, "2. [assistant][already handled] risposta") || strings.Contains(got, "skip") {
		t.Errorf("conversation context = %q", got)
	}
	if got := formatConversationContext(nil); got != "(none)" {
		t.Errorf("empty conversation = %q", got)
	}
	if got := formatConversationContext([]chatMessage{{Role: "system", Content: "x"}}); got != "(none)" {
		t.Errorf("all-skipped conversation = %q", got)
	}
}

func TestSoundtrackGuards(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	app := newTestApp()
	defer app.Shutdown()
	resetSoundtrackLimiter()

	// disabled without credentials
	spotify.Init("", "", "")
	status, _, _ := get(t, app, "/soundtrack/search?q=ab")
	if status != 404 {
		t.Errorf("search while disabled: status %d, want 404", status)
	}
	status, _, _ = postJSON(t, app, "/soundtrack/add", map[string]any{"uri": "spotify:track:x", "track_name": "Song"})
	if status != 404 {
		t.Errorf("add while disabled: status %d, want 404", status)
	}

	spotify.Init("fake-id", "fake-secret", "fake-refresh")
	// query validation happens before any upstream call
	status, _, _ = get(t, app, "/soundtrack/search")
	if status != 400 {
		t.Errorf("search without query: status %d", status)
	}
	status, _, _ = get(t, app, "/soundtrack/search?q="+strings.Repeat("a", soundtrackMaxQueryLen+1))
	if status != 400 {
		t.Errorf("search with long query: status %d", status)
	}
	// add validation: bad JSON, missing uri, missing names
	status, _, _ = postRaw(t, app, "/soundtrack/add", fiber.MIMEApplicationJSON, "{broken")
	if status != 400 {
		t.Errorf("add bad json: status %d", status)
	}
	status, _, _ = postJSON(t, app, "/soundtrack/add", map[string]any{"track_name": "Song"})
	if status != 400 {
		t.Errorf("add without uri: status %d", status)
	}
	status, _, _ = postJSON(t, app, "/soundtrack/add", map[string]any{"uri": "spotify:track:x", "track_name": "Song"})
	if status != 400 {
		t.Errorf("add without artist: status %d", status)
	}
	// a complete request stops at the missing playlist configuration
	status, _, body := postJSON(t, app, "/soundtrack/add", map[string]any{"uri": "spotify:track:x", "track_name": "Song", "artist_name": "Artist"})
	if status != 400 || !strings.Contains(body, "playlist") {
		t.Errorf("add without playlist: status %d body %q", status, body)
	}

	// after the event both endpoints close
	configureSettings(t, func(s *models.Settings) { s.CeremonyDatetime = "2020-01-01T10:00" })
	resetSoundtrackLimiter()
	status, _, _ = get(t, app, "/soundtrack/search?q=ab")
	if status != 403 {
		t.Errorf("search after event: status %d, want 403", status)
	}
	status, _, _ = postJSON(t, app, "/soundtrack/add", map[string]any{"uri": "spotify:track:x", "track_name": "Song", "artist_name": "Artist"})
	if status != 403 {
		t.Errorf("add after event: status %d, want 403", status)
	}
}

func TestSoundtrackUpstreamErrors(t *testing.T) {
	setupDB(t)
	configureSettings(t, func(s *models.Settings) {
		s.SpotifyPlaylist = "https://open.spotify.com/playlist/abc123XYZ"
	})
	app := newTestApp()
	defer app.Shutdown()
	spotify.Init("fake-id", "fake-secret", "fake-refresh")
	resetSoundtrackLimiter()

	// the fake credentials make the upstream call fail: the handler maps
	// the spotify error to a 500 for both endpoints
	status, _, _ := get(t, app, "/soundtrack/search?q=ab")
	if status != 500 {
		t.Errorf("search with failing upstream: status %d, want 500", status)
	}
	status, _, _ = postJSON(t, app, "/soundtrack/add", map[string]any{
		"uri": "spotify:track:x", "track_name": "Song", "artist_name": "Artist", "invite_code": "whatever",
	})
	if status != 500 {
		t.Errorf("add with failing upstream: status %d, want 500", status)
	}
}

func TestSoundtrackRateLimits(t *testing.T) {
	setupDB(t)
	configureSettings(t, nil)
	app := newTestApp()
	defer app.Shutdown()
	spotify.Init("fake-id", "fake-secret", "fake-refresh")
	resetSoundtrackLimiter()

	// the settings check runs after the limiter, so a closed DB still
	// consumes quota; once the quota is gone the limiter answers first
	database.DB.Close()
	for i := 0; i < soundtrackRateLimit; i++ {
		status, _, _ := get(t, app, "/soundtrack/search?q=ab")
		if status != 500 {
			t.Fatalf("search %d: status %d, want 500", i, status)
		}
	}
	status, _, _ := get(t, app, "/soundtrack/search?q=ab")
	if status != 429 {
		t.Errorf("over-limit search: status %d, want 429", status)
	}
	// search and add share one limiter: refill the budget for the add half
	resetSoundtrackLimiter()
	for i := 0; i < soundtrackRateLimit; i++ {
		status, _, _ := postJSON(t, app, "/soundtrack/add", map[string]any{"uri": "u", "track_name": "t", "artist_name": "a"})
		if status != 500 {
			t.Fatalf("add %d: status %d, want 500", i, status)
		}
	}
	status, _, _ = postJSON(t, app, "/soundtrack/add", map[string]any{"uri": "u", "track_name": "t", "artist_name": "a"})
	if status != 429 {
		t.Errorf("over-limit add: status %d, want 429", status)
	}
}

func TestSoundtrackRateLimitHelper(t *testing.T) {
	resetSoundtrackLimiter()
	ip := "203.0.113.99"
	for i := 0; i < soundtrackRateLimit; i++ {
		if !checkSoundtrackRateLimit(ip) {
			t.Fatalf("request %d rejected", i)
		}
	}
	if checkSoundtrackRateLimit(ip) {
		t.Error("request over the limit accepted")
	}
	old := make([]time.Time, soundtrackRateLimit)
	for i := range old {
		old[i] = time.Now().Add(-2 * time.Minute)
	}
	soundtrackRateLimiter.Store("192.0.2.99", old)
	if !checkSoundtrackRateLimit("192.0.2.99") {
		t.Error("expired window still blocks the ip")
	}
}
