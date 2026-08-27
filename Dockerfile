# ---- Build stage ----
FROM golang:1.22-alpine AS builder

# Install build dependencies for cgo (required by go-sqlite3).
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/bin/waste-dispatch \
    ./cmd/server

# ---- Final stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary.
COPY --from=builder /app/bin/waste-dispatch .

# Copy migrations.
COPY --from=builder /app/migrations ./migrations

# Create data directory.
RUN mkdir -p /app/data

ENV SERVER_HOST=0.0.0.0
ENV SERVER_PORT=8080
ENV DB_PATH=/app/data/waste_dispatch.db
ENV DB_MIGRATIONS_PATH=file:///app/migrations
ENV LOG_LEVEL=info
ENV LOG_PRETTY=false

EXPOSE 8080

VOLUME ["/app/data"]

ENTRYPOINT ["/app/waste-dispatch"]
