# Installation

## Docker

The recommended way to run Fœdus is via Docker, pulling the prebuilt image from GHCR:

```bash
docker pull ghcr.io/streambinder/foedus:latest
```

A `docker-compose.yml` is shipped at the repo root, ready to be tweaked:

```bash
ADMIN_USER=admin ADMIN_PASSWORD=secret docker compose up -d
```

The compose file persists the SQLite database in a named volume (`foedus_data`) mounted at `/data` inside the container.

## Custom build

Fœdus is plain Go, so the usual toolchain works. `templ` is needed once to generate the view code:

```bash
git clone https://github.com/streambinder/foedus.git
cd foedus
go install github.com/a-h/templ/cmd/templ@latest
templ generate
go build
./foedus
```

## Configuration

Fœdus reads its configuration from environment variables:

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `ADMIN_USER` | yes | — | HTTP basic auth username for `/dashboard` |
| `ADMIN_PASSWORD` | yes | — | HTTP basic auth password for `/dashboard` |
| `DATABASE_URL` | no | `foedus.db` | Path to the SQLite file |
| `PORT` | no | `3000` | TCP port to bind |
| `TRUSTED_PROXIES` | no | `127.0.0.1,::1` | Extra reverse-proxy IPs to trust for `X-Forwarded-For` |
| `OPENROUTER_API_KEY` | no | — | Enables the homepage chatbot via OpenRouter |
| `OPENROUTER_MODEL` | no | `meta-llama/llama-3.3-70b-instruct:free` | OpenRouter model name |
| `SPOTIFY_CLIENT_ID` | no | — | Enables the collaborative soundtrack |
| `SPOTIFY_CLIENT_SECRET` | no | — | Same as above |
| `SPOTIFY_REFRESH_TOKEN` | no | — | Same as above |

Everything else (couple names, venues, photos, gift list, accommodations, guest list, invitations, polls) lives in the SQLite database and is configured from the admin dashboard at `/dashboard`.
