#!/bin/bash

# Slack Webhook Test Script
# Tests Slack webhook connectivity and configuration

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🧪 Testing Slack Webhook Configuration..."

# Check if .env file exists
if [[ ! -f "${SCRIPT_DIR}/.env" ]]; then
    echo "❌ Error: .env file not found in ${SCRIPT_DIR}"
    echo "Run setup_local_e2e.sh first to create the environment"
    exit 1
fi

# Source environment variables
echo "📋 Loading environment variables..."
source "${SCRIPT_DIR}/.env"

# Validate SLACK_WEBHOOK_URL
if [[ -z "${SLACK_WEBHOOK_URL:-}" ]]; then
    echo "❌ Error: SLACK_WEBHOOK_URL not set in .env file"
    echo "Edit ${SCRIPT_DIR}/.env and add your Slack webhook URL"
    exit 1
fi

if [[ "$SLACK_WEBHOOK_URL" == "https://hooks.slack.com/services/YOUR_WORKSPACE/YOUR_CHANNEL/YOUR_WEBHOOK_TOKEN" ]]; then
    echo "❌ Error: SLACK_WEBHOOK_URL is using placeholder value"
    echo "Edit ${SCRIPT_DIR}/.env and replace with your actual Slack webhook URL"
    echo "Get webhook URL from: https://api.slack.com/apps"
    exit 1
fi

echo "✅ SLACK_WEBHOOK_URL configured: ${SLACK_WEBHOOK_URL:0:50}..."

# Test basic connectivity
echo ""
echo "🌐 Testing webhook connectivity..."

# Create test payload
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
TEST_PAYLOAD=$(cat << EOF
{
  "text": "🧪 **Alert Engine E2E Test**",
  "attachments": [
    {
      "color": "good",
      "fields": [
        {
          "title": "Test Status",
          "value": "Slack webhook connectivity test",
          "short": true
        },
        {
          "title": "Timestamp",
          "value": "${TIMESTAMP}",
          "short": true
        },
        {
          "title": "Environment",
          "value": "Local E2E Testing",
          "short": true
        },
        {
          "title": "Source",
          "value": "test_slack.sh",
          "short": true
        }
      ]
    }
  ]
}
EOF
)

# Send test message
RESPONSE=$(curl -X POST "${SLACK_WEBHOOK_URL}" \
  -H 'Content-Type: application/json' \
  -d "${TEST_PAYLOAD}" \
  -w "%{http_code}" \
  -o /tmp/slack_response.txt \
  -s)

# Check response
if [[ "$RESPONSE" == "200" ]]; then
    echo "✅ SUCCESS: Test message sent to Slack!"
    echo "📱 Check your Slack channel for the test message"
    
    # Additional verification steps
    echo ""
    echo "🔍 Verification Steps:"
    echo "1. ✅ Environment variable loaded: SLACK_WEBHOOK_URL"
    echo "2. ✅ Webhook URL format valid"
    echo "3. ✅ HTTP POST request successful (200)"
    echo "4. ✅ Slack API accepted the payload"
    
    echo ""
    echo "🎯 Next Steps:"
    echo "- Start alert engine: ./start_alert_engine.sh"
    echo "- Generate test logs to trigger alerts"
    echo "- Watch for alert notifications in Slack"
    
else
    echo "❌ FAILED: Slack webhook test failed"
    echo "HTTP Response Code: ${RESPONSE}"
    echo "Response Body:"
    cat /tmp/slack_response.txt
    echo ""
    echo "🔧 Troubleshooting:"
    echo "1. Verify webhook URL is correct"
    echo "2. Check if webhook is enabled in Slack app"
    echo "3. Confirm channel permissions"
    echo "4. Test webhook manually with curl"
    exit 1
fi

# Environment export test
echo ""
echo "🔧 Testing environment variable export..."
export SLACK_WEBHOOK_URL

if env | grep -q "SLACK_WEBHOOK_URL"; then
    echo "✅ SLACK_WEBHOOK_URL successfully exported"
else
    echo "❌ Warning: SLACK_WEBHOOK_URL export may have issues"
fi

echo ""
echo "🎉 Slack webhook test completed successfully!"
echo "The alert engine should now be able to send notifications." 