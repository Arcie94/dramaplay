# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy Go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o ingest cmd/ingest/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o dedup cmd/dedup/main.go

# Runtime Stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies (e.g. for sqlite cgo if needed, though alpine usually has musl)
# But standard binary might need libc compatibility or we use CGO_ENABLED=1 above with musl-dev.
RUN apk add --no-cache ca-certificates tzdata

# Create directories for data and uploads
RUN mkdir -p /app/data /app/public/uploads

# Copy binaries from builder
COPY --from=builder /app/server .
COPY --from=builder /app/ingest .
COPY --from=builder /app/dedup .
# Copy optional .env if needed, but we prefer docker env vars in compose

# Expose port
EXPOSE 3000

# Run
CMD ["./server"]
