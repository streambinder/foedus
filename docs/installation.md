# Installation

## Docker

The recommended way to run Fœdus is via Docker, pulling the prebuilt image from GHCR:

```bash
docker pull ghcr.io/streambinder/foedus:latest
```

A `docker-compose.yml` is shipped at the repository root, ready to be tweaked:

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

To stamp the binary with a cache-busting asset version (used to invalidate `/static/*` and `/media/*` on the client), pass an `ASSET_VERSION` value through `-ldflags` — this is what the `Dockerfile` does on every build:

```bash
ASSET_VERSION="$(date -u +%Y%m%d%H%M%S)" \
go build -ldflags "-X github.com/streambinder/foedus/internal/buildinfo.AssetVersion=${ASSET_VERSION}"
```

## Configuration

Fœdus reads its configuration from environment variables:

| Variable                | Required | Default                                  | Purpose                                                                                                                                                                |
| ----------------------- | -------- | ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ADMIN_USER`            | yes\*    | —                                        | HTTP basic auth username for `/dashboard`                                                                                                                              |
| `ADMIN_PASSWORD`        | yes\*    | —                                        | HTTP basic auth password for `/dashboard`                                                                                                                              |
| `ADMIN_USER1..9`        | no       | —                                        | Additional admin usernames (numbered suffix); paired with the same-numbered `ADMIN_PASSWORD1..9`. At least one pair is required overall, else the app refuses to boot. |
| `ADMIN_PASSWORD1..9`    | no       | —                                        | See above                                                                                                                                                              |
| `DATABASE_URL`          | no       | `foedus.db`                              | Path to the SQLite file                                                                                                                                                |
| `PORT`                  | no       | `3000`                                   | TCP port to bind                                                                                                                                                       |
| `TRUSTED_PROXIES`       | no       | `127.0.0.1,::1`                          | Extra reverse-proxy IPs to trust for `X-Forwarded-For` (loopback is always trusted regardless)                                                                         |
| `LOG_LEVEL`             | no       | `info`                                   | slog level: `debug`, `info`, `warn`, `error`                                                                                                                           |
| `LOG_FORMAT`            | no       | `text`                                   | `text` (default) or `json` for structured logging                                                                                                                      |
| `OPENROUTER_API_KEY`    | no       | —                                        | Enables the homepage chatbot via OpenRouter                                                                                                                            |
| `OPENROUTER_MODEL`      | no       | `meta-llama/llama-3.3-70b-instruct:free` | OpenRouter model name                                                                                                                                                  |
| `SPOTIFY_CLIENT_ID`     | no       | —                                        | Enables the collaborative soundtrack                                                                                                                                   |
| `SPOTIFY_CLIENT_SECRET` | no       | —                                        | Same as above                                                                                                                                                          |
| `SPOTIFY_REFRESH_TOKEN` | no       | —                                        | Same as above                                                                                                                                                          |

\* The app accepts either the unsuffixed `ADMIN_USER` / `ADMIN_PASSWORD` pair or any number of suffixed `ADMIN_USER1..9` / `ADMIN_PASSWORD1..9` pairs — at least one valid pair must be set, otherwise the process panics at startup rather than expose the dashboard with default credentials.

Everything else (couple names, venues, photos, gift list, accommodations, guest list, invitations, polls) lives in the SQLite database and is configured from the admin dashboard at `/dashboard`.
