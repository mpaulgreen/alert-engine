# Multi-stage build for Go application following Red Hat standards
# Stage 1: Build stage using Red Hat UBI8 Go builder
FROM registry.access.redhat.com/ubi8/go-toolset:latest AS builder

# Red Hat standard labels
LABEL stage="builder"

# Set working directory
WORKDIR /opt/app-root/src

# Switch to root for file operations (temporarily)
USER root

# Copy go module files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary, GOOS=linux for Linux target
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags="-w -s -X main.version=${BUILD_VERSION:-1.0.0} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o alert-engine \
    ./cmd/server

# Stage 2: Runtime stage using Red Hat UBI8 Micro for minimal attack surface
FROM registry.access.redhat.com/ubi8/ubi-micro:latest

# Red Hat standard labels
LABEL name="alert-engine" \
      version="1.0.0" \
      release="1" \
      summary="Alert Engine for Real-time Log Monitoring and Alerting" \
      description="Consumes logs from Kafka, processes them against configurable rules, \
                  and sends alerts to Slack. Designed for OpenShift environments with \
                  Redis state management and high availability." \
      maintainer="Alert Engine Team" \
      vendor="Alert Engine Project" \
      io.k8s.description="Alert Engine for Real-time Log Monitoring and Alerting" \
      io.k8s.display-name="Alert Engine" \
      io.openshift.tags="monitoring,alerting,kafka,redis,slack,logging" \
      usage="docker run -e CONFIG_PATH=/etc/alert-engine/config.yaml alert-engine"

# Create application directory with proper permissions
RUN mkdir -p /app /etc/alert-engine /tmp && \
    chown -R 1001:0 /app /etc/alert-engine /tmp && \
    chmod -R g=u /app /etc/alert-engine /tmp

# Copy binary from builder stage
COPY --from=builder --chown=1001:0 /opt/app-root/src/alert-engine /app/alert-engine

# Make binary executable
RUN chmod 755 /app/alert-engine

# Set environment variables following Red Hat conventions
ENV PATH=/app:$PATH \
    HOME=/app \
    USER=alertengine \
    CONFIG_PATH=/etc/alert-engine/config.yaml

# Expose application port
EXPOSE 8080

# Add health check with Red Hat recommended practices
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD /app/alert-engine --health-check || exit 1

# Use non-root user for security
USER 1001

# Set working directory
WORKDIR /app

# Default command - use exec form for better signal handling
CMD ["/app/alert-engine"] 