FROM golang:1.24-alpine AS builder

RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN templ generate
RUN CGO_ENABLED=0 go build -o foedus .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/foedus .
COPY --from=builder /app/static ./static

EXPOSE 3000
CMD ["./foedus"]
