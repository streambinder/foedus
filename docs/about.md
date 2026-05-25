# About

Fœdus is a self-hosted web app for wedding logistics: a landing site for the couple, dedicated invitations for each guest (or guest group), RSVP collection, a collaborative Spotify playlist, a money-transfer gift registry and a small AI chatbot to answer guests' questions.

It's designed to be deployed once per wedding: spin it up, configure it via the admin dashboard, share the public URL and per-guest invitation links, and let it collect everything you'd otherwise gather via WhatsApp threads and spreadsheets.

## Usage

### Public site

The homepage (`/`) shows the couple's landing page: ceremony and reception venues, an optional collaborative Spotify soundtrack, places that mattered to the couple, honeymoon stops on a map, and a gift registry with per-item progress.

Each guest gets a dedicated invitation URL of the form `https://your-domain/<code>`, where `<code>` is a short opaque identifier generated for each invitation from the dashboard. The invitation page renders the personalized invite, lets the guest RSVP, and — when visited the first time — flips a "viewed" flag in the database so the couple can see who has actually opened theirs.

Loading the homepage with `?invite=<code>` unlocks two extra affordances: a floating button to update the RSVP, and the input box on the soundtrack section that lets the guest add tracks to the Spotify playlist.

### Admin dashboard

The admin lives at `/dashboard`, behind HTTP basic auth (credentials set via `ADMIN_USER` and `ADMIN_PASSWORD`). From there, the couple can:

- set everything that shows up on the landing page (names, venues, dates, photos, places, soundtrack, gift registry items, accommodations, OG metadata);
- manage the guest list, with bulk CSV import and per-guest RSVP/confirmation tracking;
- generate invitations, group guests into a single invitation when needed, and reset the viewed-state if a guest needs a fresh link;
- review gifts received and tweak amounts when somebody transfers a wrong figure;
- run polls to gather guests' opinions on something (e.g. song requests, dietary needs);
- override homepage labels in any supported language without touching the source.

### Docker

The simplest way to run it is via the bundled compose file:

```bash
ADMIN_USER=admin ADMIN_PASSWORD=secret docker compose up -d
```

This starts Fœdus on port 3000, with the SQLite database persisted in a named Docker volume.

### Reverse proxy

Fœdus expects to sit behind a reverse proxy in production (nginx, caddy, etc.) — the proxy terminates TLS and forwards to Fœdus on its plain HTTP port. Make sure to forward the `X-Forwarded-For` header so per-IP rate limiters bucket correctly: set `TRUSTED_PROXIES` to the comma-separated list of upstream IPs in addition to loopback.

### Chatbot

If `OPENROUTER_API_KEY` is set, a small chat bubble appears on the homepage. It uses [OpenRouter](https://openrouter.ai) as the LLM gateway and defaults to `meta-llama/llama-3.3-70b-instruct:free`; override via `OPENROUTER_MODEL`.

The bot is fed the wedding settings as context, so it can answer common guest questions ("what time does the ceremony start?", "is there parking?") without the couple having to repeat themselves.

### Collaborative soundtrack

If `SPOTIFY_CLIENT_ID`, `SPOTIFY_CLIENT_SECRET` and `SPOTIFY_REFRESH_TOKEN` are set, and a playlist URL is configured in the dashboard, invited guests can search Spotify and queue tracks to the wedding playlist directly from the homepage. The search box is gated on the invite query param — visitors arriving without an invite see a disabled hint instead.
