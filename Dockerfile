FROM golang:1.25-alpine AS builder

RUN go install github.com/a-h/templ/cmd/templ@v0.3.1020

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN templ generate
RUN ASSET_VERSION="$(date -u +%Y%m%d%H%M%S)" && \
    CGO_ENABLED=0 go build \
    -ldflags "-X github.com/streambinder/foedus/internal/buildinfo.AssetVersion=${ASSET_VERSION}" \
    -o foedus .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates=20260413-r0
WORKDIR /app
COPY --from=builder /app/foedus .
COPY --from=builder /app/static ./static

RUN addgroup -S foedus && adduser -S foedus -G foedus
USER foedus

EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD wget --quiet --spider --tries=1 http://localhost:3000/ || exit 1
CMD ["./foedus"]
