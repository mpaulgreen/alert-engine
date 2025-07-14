#!/bin/bash

# Alert Engine Local Start Script

echo "🚀 Starting Alert Engine locally..."

# Source environment variables
if [[ -f .env ]]; then
    source .env
    echo "✅ Loaded environment variables from .env"
else
    echo "❌ .env file not found. Please run ./local/local-setup.sh first"
    exit 1
fi

# Check if Slack webhook is configured
if [[ "$SLACK_WEBHOOK_URL" == "https://hooks.slack.com/services/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" ]]; then
    echo "⚠️  WARNING: SLACK_WEBHOOK_URL is not configured in .env file"
    echo "   Slack notifications will not work until you update it"
fi

# Check port forwards
KAFKA_PF=$(lsof -ti:9092 2>/dev/null || echo "")
REDIS_PF=$(lsof -ti:6379 2>/dev/null || echo "")

if [[ -z "$KAFKA_PF" ]]; then
    echo "❌ Kafka port forward not active. Run in another terminal:"
    echo "   oc port-forward -n amq-streams-kafka svc/alert-kafka-cluster-kafka-bootstrap 9092:9092"
    exit 1
fi

if [[ -z "$REDIS_PF" ]]; then
    echo "❌ Redis port forward not active. Run in another terminal:"
    echo "   oc port-forward -n redis-cluster svc/redis-cluster-access 6379:6379"
    exit 1
fi

echo "✅ Port forwards are active"
echo "✅ Starting Alert Engine..."

# Run Alert Engine
exec ./alert-engine
