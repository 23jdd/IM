# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Use a China-friendly Go module proxy to speed up downloads
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=off

# Cache dependencies first
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the rest of the source (frontend is a separate module and is ignored via .dockerignore)
COPY . .

# Build a static binary (all dependencies are pure Go, so CGO can be disabled)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/im .

# ---- Runtime stage ----
FROM alpine:3.20

# Use a China mirror for apk to speed up package installation
RUN sed -i 's#https\?://dl-cdn.alpinelinux.org#https://mirrors.aliyun.com#g' /etc/apk/repositories && \
    apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/im /app/im

# config.yaml is provided at runtime (mounted by docker-compose)
RUN mkdir -p /app/logs && chown -R appuser:appuser /app

USER appuser

# http, tcp, gateway
EXPOSE 8080 9000 8000

ENTRYPOINT ["/app/im"]
