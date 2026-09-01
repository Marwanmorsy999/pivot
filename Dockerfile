# Build stage
FROM golang:1.27.0-alpine3.24 AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with CGO (required for go-sqlite3)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w -X main.version=docker" -o pivot ./cmd/pivot

# Runtime stage
FROM alpine:3.24.1

RUN apk --no-cache add git bash ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/pivot /usr/local/bin/pivot

# Create pivot config directory
RUN mkdir -p /root/.pivot

ENTRYPOINT ["pivot"]
CMD ["--help"]
