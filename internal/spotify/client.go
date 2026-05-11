package spotify

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	clientID     string
	clientSecret string
	refreshToken string

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time

	// dedicated client w/ overall timeout — http.DefaultClient has none, so a
	// hung Spotify response would tie up a SQLite conn forever.
	httpClient = &http.Client{Timeout: 10 * time.Second}
)

type Track struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URI      string `json:"uri"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	ImageURL string `json:"image_url"`
}

func Init(id, secret, refresh string) {
	clientID = id
	clientSecret = secret
	refreshToken = refresh
	if id != "" && secret != "" && refresh != "" {
		slog.Info("spotify client enabled")
	} else {
		slog.Warn("spotify client disabled", "reason", "credentials not set")
	}
}

func Enabled() bool {
	return clientID != "" && clientSecret != "" && refreshToken != ""
}

func getAccessToken() (string, error) {
	start := time.Now()
	mu.Lock()
	defer mu.Unlock()

	if accessToken != "" && time.Now().Before(tokenExpiry) {
		slog.Debug("spotify access token cache hit", "expires_in_ms", time.Until(tokenExpiry).Milliseconds())
		return accessToken, nil
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}

	tokReq, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("token refresh request build failed: %w", err)
	}
	tokReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(tokReq)
	if err != nil {
		return "", fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token refresh failed: status=%d body=%s", resp.StatusCode, body)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("token decode failed: %w", err)
	}

	accessToken = result.AccessToken
	// refresh 60s before actual expiry
	tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	slog.Info("spotify token refreshed", "expires_in_seconds", result.ExpiresIn, "duration_ms", time.Since(start).Milliseconds())

	return accessToken, nil
}

func Search(query string, limit int) ([]Track, error) {
	start := time.Now()
	token, err := getAccessToken()
	if err != nil {
		return nil, err
	}

	params := url.Values{
		"q":     {query},
		"type":  {"track"},
		"limit": {fmt.Sprintf("%d", limit)},
	}

	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/search?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: status=%d body=%s", resp.StatusCode, body)
	}

	var result struct {
		Tracks struct {
			Items []struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				URI     string `json:"uri"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
				Album struct {
					Name   string `json:"name"`
					Images []struct {
						URL    string `json:"url"`
						Height int    `json:"height"`
					} `json:"images"`
				} `json:"album"`
			} `json:"items"`
		} `json:"tracks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	tracks := make([]Track, 0, len(result.Tracks.Items))
	for _, item := range result.Tracks.Items {
		var artistNames []string
		for _, a := range item.Artists {
			artistNames = append(artistNames, a.Name)
		}
		// pick smallest image (usually 64px thumbnail)
		imageURL := ""
		if len(item.Album.Images) > 0 {
			imageURL = item.Album.Images[len(item.Album.Images)-1].URL
		}
		tracks = append(tracks, Track{
			ID:       item.ID,
			Name:     item.Name,
			URI:      item.URI,
			Artist:   strings.Join(artistNames, ", "),
			Album:    item.Album.Name,
			ImageURL: imageURL,
		})
	}
	slog.Info("spotify search completed", "query_len", len(strings.TrimSpace(query)), "limit", limit, "results", len(tracks), "duration_ms", time.Since(start).Milliseconds())
	return tracks, nil
}

func AddToPlaylist(playlistID, trackURI string) error {
	start := time.Now()
	token, err := getAccessToken()
	if err != nil {
		return err
	}

	body := fmt.Sprintf(`{"uris":[%q]}`, trackURI)
	req, err := http.NewRequest("POST", "https://api.spotify.com/v1/playlists/"+playlistID+"/items", strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("add track failed: status=%d body=%s", resp.StatusCode, respBody)
	}
	slog.Info("spotify track added to playlist", "playlist_id", playlistID, "track_uri", trackURI, "duration_ms", time.Since(start).Milliseconds())
	return nil
}
