#!/bin/bash

# Alert Engine Starter Script with Environment Variable Fix
# This script ensures proper environment variable inheritance for Slack notifications

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "🚀 Starting Alert Engine with proper environment configuration..."

# Check if .env file exists
if [[ ! -f "${SCRIPT_DIR}/.env" ]]; then
    echo "❌ Error: .env file not found in ${SCRIPT_DIR}"
    echo "Run setup_local_e2e.sh first to create the environment"
    exit 1
fi

# Source environment variables
echo "📋 Loading environment variables from .env..."
source "${SCRIPT_DIR}/.env"

# Verify critical variables
if [[ -z "${SLACK_WEBHOOK_URL:-}" ]]; then
    echo "⚠️  Warning: SLACK_WEBHOOK_URL not set in .env file"
    echo "Slack notifications will not work"
elif [[ "$SLACK_WEBHOOK_URL" == "https://hooks.slack.com/services/YOUR_WORKSPACE/YOUR_CHANNEL/YOUR_WEBHOOK_TOKEN" ]]; then
    echo "⚠️  Warning: SLACK_WEBHOOK_URL is using placeholder value"
    echo "Edit .env file with your actual Slack webhook URL for notifications to work"
else
    echo "✅ SLACK_WEBHOOK_URL configured: ${SLACK_WEBHOOK_URL:0:50}..."
fi

# Export critical environment variables for Go process inheritance
export SLACK_WEBHOOK_URL
export CONFIG_PATH="${SCRIPT_DIR}/config_local_e2e.yaml"

if [[ -n "${REDIS_ADDRESS:-}" ]]; then
    export REDIS_ADDRESS
fi

if [[ -n "${KAFKA_BROKERS:-}" ]]; then
    export KAFKA_BROKERS
fi

echo "🔧 Environment variables exported for Go process inheritance"

# Navigate to project root
cd "${PROJECT_ROOT}"

echo "🎯 Starting alert engine..."
echo "   Config: ${CONFIG_PATH}"
echo "   Working directory: $(pwd)"
echo ""
echo "👀 Watch for this line in the logs:"
echo "   ✅ Slack notifier configured: webhook=%!s(bool=true)"
echo "   ❌ If you see webhook=%!s(bool=false), there's an environment issue"
echo ""

# Start the alert engine with inherited environment
exec go run cmd/server/main.go 