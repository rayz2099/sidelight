# Build stage
FROM --platform=$BUILDPLATFORM golang:1.25.4-alpine AS builder
ARG TARGETOS TARGETARCH
WORKDIR /app
RUN apk add --no-cache git ca-certificates
ENV GOPROXY=https://proxy.golang.org,direct
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o bin/sidelight ./cmd/sidelight

# Runtime stage
FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    libimage-exiftool-perl \
    rawtherapee \
    curl \
    && rm -rf /var/lib/apt/lists/*
RUN which rawtherapee-cli || \
    (test -f /usr/bin/rawtherapee && ln -s /usr/bin/rawtherapee /usr/bin/rawtherapee-cli) || \
    echo "Warning: rawtherapee-cli not found"

RUN useradd -r -s /bin/false -m -d /app sidelight
WORKDIR /app
COPY --from=builder /app/bin/sidelight /usr/local/bin/sidelight
COPY --from=builder /app/assets /app/assets
RUN mkdir -p /app/data /app/config /app/tmp && \
    chown -R sidelight:sidelight /app

ENV RT_CLI_PATH=/usr/bin/rawtherapee-cli
USER sidelight
HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=10s \
    CMD curl -f http://localhost:12700/health || exit 1
EXPOSE 12700
ENTRYPOINT ["sidelight"]
CMD ["server", "--port", "12700", "-t", "/app/tmp"]
