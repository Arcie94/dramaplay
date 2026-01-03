# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev python3 py3-pip python3-dev libffi-dev

# Copy Go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy Python requirements
COPY python/requirements.txt ./python/requirements.txt
RUN pip3 install -r ./python/requirements.txt --break-system-packages

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

# Install runtime dependencies
# We need python3 and pip. We also need build deps for pycryptodome, then remove them.
RUN apk add --no-cache python3 py3-pip \
    && apk add --no-cache --virtual .build-deps gcc musl-dev python3-dev libffi-dev \
    && mkdir -p /app/python

# Copy python script and requirements
COPY --from=builder /app/python /app/python

# Install python requirements in runtime
RUN pip3 install -r /app/python/requirements.txt --break-system-packages \
    && apk del .build-deps

# Install other runtime deps
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
