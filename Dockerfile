# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.26-alpine AS builder

WORKDIR /src

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

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/im /app/im

# config.yaml is provided at runtime (mounted by docker-compose)
RUN mkdir -p /app/logs && chown -R appuser:appuser /app

USER appuser

# http, tcp, gateway
EXPOSE 8080 9000 8000

ENTRYPOINT ["/app/im"]
