# syntax=docker/dockerfile:1

# ==============================================================================
# Stage 1: Verify automated test suite & compile static Go binary
# ==============================================================================
FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY static/ ./static/

RUN go test ./... && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bioscope ./cmd/server

# ==============================================================================
# Stage 2: Minimal hardened production runtime (Hugging Face Spaces compatible)
# ==============================================================================
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget && \
    adduser -D -u 1000 bioscope

WORKDIR /app

COPY --from=builder --chown=bioscope:bioscope /out/bioscope /app/bioscope
COPY --chown=bioscope:bioscope static/ /app/static/

USER bioscope

ENV BIOSCOPE_HOST=0.0.0.0 \
    PORT=7860 \
    BIOSCOPE_LOG_LEVEL=info \
    BIOSCOPE_OFFLINE_FALLBACK=true

EXPOSE 7860

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:${PORT}/healthz || exit 1

ENTRYPOINT ["/app/bioscope"]
