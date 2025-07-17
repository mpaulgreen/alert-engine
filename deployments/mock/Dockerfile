# Use Red Hat Universal Base Image 8 with Python 3.9
FROM registry.access.redhat.com/ubi8/python-39:latest

# Red Hat standard labels
LABEL name="mock-log-generator" \
      version="1.0.0" \
      release="1" \
      summary="Mock Log Generator for Alert Engine Testing" \
      description="Generates realistic application logs for Alert Engine testing in OpenShift environments. \
                  Outputs structured JSON logs to stdout for collection by Vector/ClusterLogForwarder." \
      maintainer="Alert Engine Team" \
      vendor="Alert Engine Project" \
      io.k8s.description="Mock Log Generator for Alert Engine Testing" \
      io.k8s.display-name="Mock Log Generator" \
      io.openshift.tags="logging,testing,mock-data,alert-engine" \
      usage="docker run -e LOG_MODE=continuous mock-log-generator"

# Set working directory
WORKDIR /app

# Switch to root for system package installation (temporarily)
USER root

# Install system dependencies and clean up in single layer for efficiency
RUN dnf update -y && \
    dnf install -y gcc python3-devel && \
    dnf clean all && \
    rm -rf /var/cache/dnf

# Use existing default user from UBI8 Python base image (UID 1001)
# Set proper ownership and permissions for OpenShift compatibility
RUN chown -R 1001:0 /app && \
    chmod -R g=u /app

# Copy requirements file first for better layer caching
COPY requirements.txt /app/

# Switch to non-root user for package installation
USER 1001

# Install Python dependencies (no --user flag needed in virtual environment)
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY --chown=1001:0 mock_log_generator.py /app/

# Make script executable (Red Hat prefers explicit permissions)
RUN chmod 755 /app/mock_log_generator.py

# Set environment variables following Red Hat conventions
ENV PYTHONPATH=/app \
    PYTHONUNBUFFERED=1 \
    HOME=/app

# Expose no ports (this is a log generator, not a service)
# EXPOSE - none needed

# Add health check with Red Hat recommended practices
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD python3 -c "import json, time, random, sys; print('MockLogGenerator health check passed'); sys.exit(0)" || exit 1

# Use exec form and run as non-root user
USER 1001

# Default command - use exec form for better signal handling
CMD ["python3", "/app/mock_log_generator.py", "--mode", "continuous"] 