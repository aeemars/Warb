# ============================================
# Warba Bank — Opportunity Engine
# Multi-stage Docker build for Render deployment
# ============================================

# Stage 1: Build the Go binary
FROM golang:alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build pure-Go static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server

# Stage 2: Minimal runtime image
FROM alpine:latest

WORKDIR /app

# Copy root SSL certs from builder stage (no apk network call needed)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary and static frontend assets
COPY --from=builder /app/server .
COPY --from=builder /app/web ./web

# Create persistent data directory
RUN mkdir -p /data

# Default environment
ENV PORT=8080
ENV DB_PATH=/data/opportunity.db
ENV ENV=production

EXPOSE 8080

CMD ["./server"]
