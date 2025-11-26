# Waffle - Well Architected Framework for Less Effort
# Multi-stage Docker build

# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE} -s -w" \
    -o waffle \
    ./cmd/waffle

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 waffle && \
    adduser -D -u 1000 -G waffle waffle

# Set working directory
WORKDIR /workspace

# Copy binary from builder
COPY --from=builder /build/waffle /usr/local/bin/waffle

# Copy example configuration
COPY config.example.yaml /etc/waffle/config.example.yaml

# Create directories for waffle data
RUN mkdir -p /home/waffle/.waffle/sessions /home/waffle/.waffle/logs && \
    chown -R waffle:waffle /home/waffle/.waffle

# Switch to non-root user
USER waffle

# Set environment variables
ENV WAFFLE_LOG_LEVEL=info

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD waffle --version || exit 1

# Default command
ENTRYPOINT ["waffle"]
CMD ["--help"]

# Labels
LABEL org.opencontainers.image.title="Waffle" \
      org.opencontainers.image.description="Well Architected Framework for Less Effort - Automated WAFR reviews" \
      org.opencontainers.image.vendor="Waffle" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${DATE}"
