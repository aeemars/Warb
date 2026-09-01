# ============================================
# Warba Bank — Opportunity Engine
# Multi-stage Docker build for Render deployment
# ============================================

# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/server ./cmd/server

# Stage 2: Minimal runtime image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

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
