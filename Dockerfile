FROM golang:1.27.1-alpine AS builder

WORKDIR /app

# The module and build caches live in BuildKit cache mounts, exported
# with the layer cache, so a cold layer cache no longer means a full
# module download and recompile.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download && \
    go install github.com/a-h/templ/cmd/templ@v0.3.1020

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    templ generate && \
    ASSET_VERSION="$(date -u +%Y%m%d%H%M%S)" && \
    CGO_ENABLED=0 go build \
    -ldflags "-X github.com/streambinder/foedus/internal/buildinfo.AssetVersion=${ASSET_VERSION}" \
    -o foedus .

FROM alpine:3.24.2
RUN apk add --no-cache ca-certificates=20260909-r0
WORKDIR /app
COPY --from=builder /app/foedus .
COPY --from=builder /app/static ./static

RUN addgroup -S foedus && adduser -S foedus -G foedus
USER foedus

EXPOSE 3000
HEALTHCHECK --interval=60s --timeout=3s --start-period=5s --retries=3 CMD ["wget", "--quiet", "--spider", "--tries=1", "http://localhost:3000/healthz"]
CMD ["./foedus"]
