package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestParseMapsURL(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want mapsPlace
		ok   bool
	}{
		{
			name: "place link with pin data",
			raw:  "https://www.google.com/maps/place/Villa+Colonnetta/@42.540000,12.920000,17z/data=!3m1!4b1!4m6!3m5!1s0x132f2b:0x1234!8m2!3d42.538654!4d12.918123",
			want: mapsPlace{Name: "Villa Colonnetta", Lat: 42.538654, Lng: 12.918123},
			ok:   true,
		},
		{
			name: "place link with viewport only",
			raw:  "https://www.google.com/maps/place/Parrocchia+di+San+Clemente/@41.943100,12.523400,19z",
			want: mapsPlace{Name: "Parrocchia di San Clemente", Lat: 41.9431, Lng: 12.5234},
			ok:   true,
		},
		{
			name: "percent encoded name",
			raw:  "https://www.google.it/maps/place/Caff%C3%A8+del+Corso/@45.464100,9.191900,18z",
			want: mapsPlace{Name: "Caffè del Corso", Lat: 45.4641, Lng: 9.1919},
			ok:   true,
		},
		{
			name: "dropped pin without a name",
			raw:  "https://www.google.com/maps?q=42.538654,12.918123",
			want: mapsPlace{Name: "", Lat: 42.538654, Lng: 12.918123},
			ok:   true,
		},
		{
			name: "query parameter coordinates",
			raw:  "https://maps.google.com/?query=41.006381,28.9758715",
			want: mapsPlace{Name: "", Lat: 41.006381, Lng: 28.9758715},
			ok:   true,
		},
		{name: "search query is not coordinates", raw: "https://www.google.com/maps/search/pizzeria+roma", ok: false},
		{name: "out of range coordinates", raw: "https://www.google.com/maps/@142.0,12.0,10z", ok: false},
		{name: "zero coordinates", raw: "https://www.google.com/maps/@0,0,10z", ok: false},
		{name: "not a url", raw: "villa colonnetta", ok: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseMapsURL(tc.raw)
			if ok != tc.ok {
				t.Fatalf("parseMapsURL(%q) ok = %v, want %v", tc.raw, ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Fatalf("parseMapsURL(%q) = %+v, want %+v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestIsMapsHost(t *testing.T) {
	for _, tc := range []struct {
		host string
		want bool
	}{
		{"maps.app.goo.gl", true},
		{"goo.gl", true},
		{"www.google.com", true},
		{"google.it", true},
		{"maps.google.com", true},
		{"www.google.co.uk", true},
		{"evilgoogle.com", false},
		{"google.com.evil.example", false},
		{"example.com", false},
	} {
		if got := isMapsHost(tc.host); got != tc.want {
			t.Errorf("isMapsHost(%q) = %v, want %v", tc.host, got, tc.want)
		}
	}
}

func resolveRequest(t *testing.T, app interface {
	Test(*http.Request, ...int) (*http.Response, error)
}, raw string,
) (int, mapsPlace) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/places/resolve?url="+url.QueryEscape(raw), nil)
	resp, err := app.Test(req, 60000)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	place := mapsPlace{}
	if resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(body, &place); err != nil {
			t.Fatalf("response is not a place: %v", err)
		}
	}
	return resp.StatusCode, place
}

func TestResolveMapsLink(t *testing.T) {
	app := newTestApp()

	status, place := resolveRequest(t, app, "https://www.google.com/maps/place/Villa+Colonnetta/@42.538654,12.918123,17z")
	if status != http.StatusOK || place.Name != "Villa Colonnetta" || place.Lat != 42.538654 || place.Lng != 12.918123 {
		t.Fatalf("full link: status %d place %+v", status, place)
	}

	status, _ = resolveRequest(t, app, "https://example.com/maps/place/X/@1,2")
	if status != http.StatusBadRequest {
		t.Fatalf("foreign host: status %d, want 400", status)
	}

	status, _ = resolveRequest(t, app, "https://www.google.com/maps/search/pizzeria")
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("link without a place: status %d, want 422", status)
	}
}

func TestResolveMapsLinkShort(t *testing.T) {
	final := "https://www.google.com/maps/place/Villa+Colonnetta/@42.538654,12.918123,17z/data=!3m1!4b1!8m2!3d42.538654!4d12.918123"
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Host, "goo.gl") {
			return &http.Response{
				StatusCode: http.StatusFound,
				Header:     http.Header{"Location": []string{final}},
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    r,
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader("<html></html>")),
			Request:    r,
		}, nil
	})
	original := mapsHTTPClient
	mapsHTTPClient = &http.Client{Transport: transport, CheckRedirect: mapsRedirectPolicy}
	t.Cleanup(func() { mapsHTTPClient = original })

	app := newTestApp()
	status, place := resolveRequest(t, app, "https://maps.app.goo.gl/abc123XYZ")
	if status != http.StatusOK || place.Name != "Villa Colonnetta" || place.Lat != 42.538654 || place.Lng != 12.918123 {
		t.Fatalf("short link: status %d place %+v", status, place)
	}
}

func TestResolveMapsLinkRedirectEscape(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"https://example.com/steal"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    r,
		}, nil
	})
	original := mapsHTTPClient
	mapsHTTPClient = &http.Client{Transport: transport, CheckRedirect: mapsRedirectPolicy}
	t.Cleanup(func() { mapsHTTPClient = original })

	app := newTestApp()
	status, _ := resolveRequest(t, app, "https://maps.app.goo.gl/abc123XYZ")
	if status != http.StatusBadGateway {
		t.Fatalf("redirect escape: status %d, want 502", status)
	}
}
