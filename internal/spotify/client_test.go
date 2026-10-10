package spotify

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// roundTripFunc adapts a function to http.RoundTripper so tests can script
// the Spotify API without any network access.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// installTransport swaps the package HTTP client for one backed by fn and
// resets the credential and token state, restoring everything afterwards.
func installTransport(t *testing.T, fn roundTripFunc) {
	t.Helper()
	oldClient := httpClient
	oldID, oldSecret, oldRefresh := clientID, clientSecret, refreshToken
	oldToken, oldExpiry := accessToken, tokenExpiry
	t.Cleanup(func() {
		httpClient = oldClient
		clientID, clientSecret, refreshToken = oldID, oldSecret, oldRefresh
		accessToken, tokenExpiry = oldToken, oldExpiry
	})
	httpClient = &http.Client{Transport: fn}
	accessToken = ""
	tokenExpiry = time.Time{}
}

func tokenHandler(tokenBody string, tokenStatus int) roundTripFunc {
	return func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Host, "accounts.spotify.com") {
			return jsonResponse(tokenStatus, tokenBody), nil
		}
		return jsonResponse(http.StatusNotFound, `{}`), nil
	}
}

func TestInitAndEnabled(t *testing.T) {
	oldID, oldSecret, oldRefresh := clientID, clientSecret, refreshToken
	t.Cleanup(func() { clientID, clientSecret, refreshToken = oldID, oldSecret, oldRefresh })

	Init("", "", "")
	if Enabled() {
		t.Error("client with no credentials must be disabled")
	}
	Init("id", "", "refresh")
	if Enabled() {
		t.Error("client with partial credentials must be disabled")
	}
	Init("id", "secret", "refresh")
	if !Enabled() {
		t.Error("client with full credentials must be enabled")
	}
}

func TestGetAccessTokenCaches(t *testing.T) {
	installTransport(t, nil)
	Init("id", "secret", "refresh")
	var tokenCalls atomic.Int32
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		tokenCalls.Add(1)
		return jsonResponse(200, `{"access_token":"tok-1","expires_in":3600}`), nil
	})}

	token, err := getAccessToken()
	if err != nil || token != "tok-1" {
		t.Fatalf("getAccessToken = %q, %v", token, err)
	}
	token, err = getAccessToken()
	if err != nil || token != "tok-1" {
		t.Fatalf("cached getAccessToken = %q, %v", token, err)
	}
	if got := tokenCalls.Load(); got != 1 {
		t.Errorf("token endpoint called %d times, want 1 (cache hit)", got)
	}
}

func TestGetAccessTokenErrors(t *testing.T) {
	Init("id", "secret", "refresh")

	// upstream rejects the refresh
	installTransport(t, tokenHandler(`{"error":"invalid_grant"}`, 400))
	if _, err := getAccessToken(); err == nil {
		t.Error("expected an error for a rejected token refresh")
	}

	// transport failure
	installTransport(t, func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})
	if _, err := getAccessToken(); err == nil {
		t.Error("expected an error for a failed token request")
	}

	// malformed token payload
	installTransport(t, tokenHandler(`not-json`, 200))
	if _, err := getAccessToken(); err == nil {
		t.Error("expected an error for a malformed token payload")
	}
}

const searchBody = `{"tracks":{"items":[
	{"id":"t1","name":"First Song","uri":"spotify:track:t1",
	 "artists":[{"name":"Artist A"},{"name":"Artist B"}],
	 "album":{"name":"Album One","images":[{"url":"big.png","height":640},{"url":"small.png","height":64}]}},
	{"id":"t2","name":"Second Song","uri":"spotify:track:t2",
	 "artists":[{"name":"Solo"}],
	 "album":{"name":"Album Two","images":[]}}
]}}`

func scriptedAPI(t *testing.T, apiStatus int, apiBody string) {
	t.Helper()
	Init("id", "secret", "refresh")
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Host, "accounts.spotify.com") {
			return jsonResponse(200, `{"access_token":"tok","expires_in":3600}`), nil
		}
		return jsonResponse(apiStatus, apiBody), nil
	})}
}

func TestSearch(t *testing.T) {
	installTransport(t, nil)
	scriptedAPI(t, 200, searchBody)

	tracks, err := Search("love", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 2 {
		t.Fatalf("got %d tracks, want 2", len(tracks))
	}
	first := tracks[0]
	if first.ID != "t1" || first.Name != "First Song" || first.URI != "spotify:track:t1" {
		t.Errorf("first track = %+v", first)
	}
	if first.Artist != "Artist A, Artist B" {
		t.Errorf("first track artist = %q, want joined names", first.Artist)
	}
	if first.Album != "Album One" || first.ImageURL != "small.png" {
		t.Errorf("first track album/image = %q/%q, want smallest image", first.Album, first.ImageURL)
	}
	if tracks[1].ImageURL != "" {
		t.Errorf("second track image = %q, want empty for no images", tracks[1].ImageURL)
	}
}

func TestSearchErrors(t *testing.T) {
	installTransport(t, nil)

	// API error status
	scriptedAPI(t, 500, `{"error":"boom"}`)
	if _, err := Search("x", 5); err == nil {
		t.Error("expected an error for a failed search")
	}

	// malformed payload
	scriptedAPI(t, 200, `{"tracks":`)
	if _, err := Search("x", 5); err == nil {
		t.Error("expected an error for a malformed search payload")
	}

	// transport failure on the API call (token endpoint still works)
	Init("id", "secret", "refresh")
	accessToken = ""
	tokenExpiry = time.Time{}
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Host, "accounts.spotify.com") {
			return jsonResponse(200, `{"access_token":"tok","expires_in":3600}`), nil
		}
		return nil, errors.New("timeout")
	})}
	if _, err := Search("x", 5); err == nil {
		t.Error("expected an error for a failed search request")
	}

	// token failure propagates before any search happens
	installTransport(t, tokenHandler(`{}`, 401))
	Init("id", "secret", "refresh")
	if _, err := Search("x", 5); err == nil {
		t.Error("expected the token error to propagate from Search")
	}
}

func TestAddToPlaylist(t *testing.T) {
	installTransport(t, nil)
	scriptedAPI(t, 201, `{}`)
	if err := AddToPlaylist("playlist1", "spotify:track:t1"); err != nil {
		t.Fatalf("AddToPlaylist with 201: %v", err)
	}

	scriptedAPI(t, 200, `{}`)
	accessToken = ""
	tokenExpiry = time.Time{}
	if err := AddToPlaylist("playlist1", "spotify:track:t1"); err != nil {
		t.Fatalf("AddToPlaylist with 200: %v", err)
	}
}

func TestAddToPlaylistErrors(t *testing.T) {
	installTransport(t, nil)

	scriptedAPI(t, 403, `{"error":"forbidden"}`)
	if err := AddToPlaylist("playlist1", "spotify:track:t1"); err == nil {
		t.Error("expected an error for a rejected add")
	}

	// transport failure on the API call
	Init("id", "secret", "refresh")
	accessToken = ""
	tokenExpiry = time.Time{}
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Host, "accounts.spotify.com") {
			return jsonResponse(200, `{"access_token":"tok","expires_in":3600}`), nil
		}
		return nil, errors.New("connection reset")
	})}
	if err := AddToPlaylist("playlist1", "spotify:track:t1"); err == nil {
		t.Error("expected an error for a failed add request")
	}

	// token failure propagates
	installTransport(t, tokenHandler(`{}`, 401))
	Init("id", "secret", "refresh")
	if err := AddToPlaylist("playlist1", "spotify:track:t1"); err == nil {
		t.Error("expected the token error to propagate from AddToPlaylist")
	}
}
