FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags=-static' \
    -o logcleaner-controller \
    ./cmd/logcleaner/main.go

RUN ./logcleaner-controller --version || true

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1000 logcleaner && \
    adduser -u 1000 -G logcleaner -s /bin/sh -D logcleaner

COPY --from=builder /build/logcleaner-controller /usr/local/bin/

RUN chown logcleaner:logcleaner /usr/local/bin/logcleaner-controller && \
    chmod +x /usr/local/bin/logcleaner-controller

USER logcleaner

ENTRYPOINT ["/usr/local/bin/logcleaner-controller"]