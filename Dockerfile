# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o pivot ./cmd/pivot

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add git bash

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/pivot /usr/local/bin/pivot

# Create pivot config directory
RUN mkdir -p /root/.pivot

ENTRYPOINT ["pivot"]
CMD ["--help"]
