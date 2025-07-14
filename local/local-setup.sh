#!/bin/bash

# Alert Engine Local Setup Script
# This script helps set up the Alert Engine for local development with OpenShift infrastructure

set -e

echo "🚀 Alert Engine Local Setup Script"
echo "=================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Check prerequisites
echo ""
echo "📋 Checking Prerequisites..."

# Check Go
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.23+ from https://golang.org/downloads/"
    exit 1
else
    GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
    print_status "Go $GO_VERSION is installed"
fi

# Check OpenShift CLI
if ! command -v oc &> /dev/null; then
    print_error "OpenShift CLI (oc) is not installed"
    exit 1
else
    print_status "OpenShift CLI is installed"
fi

# Check OpenShift connection
if ! oc whoami &> /dev/null; then
    print_error "Not connected to OpenShift cluster. Please run 'oc login' first"
    exit 1
else
    CURRENT_USER=$(oc whoami)
    CURRENT_PROJECT=$(oc project --short=true 2>/dev/null || echo "none")
    print_status "Connected to OpenShift as $CURRENT_USER (project: $CURRENT_PROJECT)"
fi

# Verify OpenShift infrastructure
echo ""
echo "🔍 Verifying OpenShift Infrastructure..."

# Check Kafka
if oc get kafka alert-kafka-cluster -n amq-streams-kafka &> /dev/null; then
    KAFKA_STATUS=$(oc get kafka alert-kafka-cluster -n amq-streams-kafka -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null)
    if [[ "$KAFKA_STATUS" == "True" ]]; then
        print_status "Kafka cluster is ready"
    else
        print_error "Kafka cluster is not ready"
        exit 1
    fi
else
    print_error "Kafka cluster not found. Please run the infrastructure setup first"
    exit 1
fi

# Check Redis
REDIS_PODS=$(oc get pods -l app=redis-cluster -n redis-cluster --no-headers 2>/dev/null | grep Running | wc -l)
if [[ "$REDIS_PODS" -eq 6 ]]; then
    print_status "Redis cluster is ready (6/6 pods running)"
else
    print_error "Redis cluster is not ready ($REDIS_PODS/6 pods running)"
    exit 1
fi

# Check ClusterLogForwarder
CLF_STATUS=$(oc get clusterlogforwarder kafka-forwarder -n openshift-logging -o jsonpath='{.status.conditions[?(@.type=="observability.openshift.io/Valid")].status}' 2>/dev/null)
if [[ "$CLF_STATUS" == "True" ]]; then
    print_status "ClusterLogForwarder is valid"
else
    print_warning "ClusterLogForwarder may not be properly configured"
fi

# Build Alert Engine
echo ""
echo "🔨 Building Alert Engine..."
cd "$(dirname "$0")/.."

if go mod tidy && go mod download; then
    print_status "Go modules updated"
else
    print_error "Failed to update Go modules"
    exit 1
fi

if go build -o alert-engine ./cmd/server/; then
    print_status "Alert Engine built successfully"
else
    print_error "Failed to build Alert Engine"
    exit 1
fi

# Create .env file if it doesn't exist
echo ""
echo "⚙️  Setting up environment..."

if [[ ! -f .env ]]; then
    cat > .env << 'EOF'
# Alert Engine Local Development Environment Variables
# Update SLACK_WEBHOOK_URL with your actual webhook URL

# Slack Integration (REQUIRED)
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/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"

# Configuration
export CONFIG_PATH="./configs/config.yaml"
export ENVIRONMENT="development"
export LOG_LEVEL="debug"

# Optional overrides
export REDIS_ADDRESS="localhost:6379"
export KAFKA_BROKERS="localhost:9092"
export SERVER_ADDRESS=":8080"
EOF
    print_status "Created .env file template"
    print_warning "Please edit .env file and update SLACK_WEBHOOK_URL with your actual webhook URL"
else
    print_status ".env file already exists"
fi

# Check if port forwards are needed
echo ""
echo "🔗 Checking port forwards..."

KAFKA_PF=$(lsof -ti:9092 2>/dev/null || echo "")
REDIS_PF=$(lsof -ti:6379 2>/dev/null || echo "")

if [[ -z "$KAFKA_PF" ]]; then
    print_warning "Kafka port forward (9092) is not active"
    echo "  Run: oc port-forward -n amq-streams-kafka svc/alert-kafka-cluster-kafka-bootstrap 9092:9092"
else
    print_status "Kafka port forward is active (PID: $KAFKA_PF)"
fi

if [[ -z "$REDIS_PF" ]]; then
    print_warning "Redis port forward (6379) is not active"
    echo "  Run: oc port-forward -n redis-cluster svc/redis-cluster-access 6379:6379"
else
    print_status "Redis port forward is active (PID: $REDIS_PF)"
fi

# Create start script
cat > start-local.sh << 'EOF'
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
EOF

chmod +x start-local.sh
print_status "Created start-local.sh script"

# Create port forward helper script
cat > setup-port-forwards.sh << 'EOF'
#!/bin/bash

# Port Forward Setup Script
# Run this script to set up port forwards for local development

echo "🔗 Setting up port forwards for Alert Engine local development..."

# Check if tmux is available for better session management
if command -v tmux &> /dev/null; then
    echo "Using tmux for session management..."
    
    # Create new tmux session
    tmux new-session -d -s alert-engine-pf
    
    # Split window horizontally
    tmux split-window -h -t alert-engine-pf
    
    # Setup Kafka port forward in first pane
    tmux send-keys -t alert-engine-pf:0.0 'echo "🔗 Setting up Kafka port forward..."; oc port-forward -n amq-streams-kafka svc/alert-kafka-cluster-kafka-bootstrap 9092:9092' Enter
    
    # Setup Redis port forward in second pane
    tmux send-keys -t alert-engine-pf:0.1 'echo "🔗 Setting up Redis port forward..."; oc port-forward -n redis-cluster svc/redis-cluster-access 6379:6379' Enter
    
    echo "✅ Port forwards started in tmux session 'alert-engine-pf'"
    echo "   To view: tmux attach -t alert-engine-pf"
    echo "   To stop: tmux kill-session -t alert-engine-pf"
    
else
    echo "⚠️  tmux not available. Please run these commands in separate terminals:"
    echo ""
    echo "Terminal 1 (Kafka):"
    echo "oc port-forward -n amq-streams-kafka svc/alert-kafka-cluster-kafka-bootstrap 9092:9092"
    echo ""
    echo "Terminal 2 (Redis):"
    echo "oc port-forward -n redis-cluster svc/redis-cluster-access 6379:6379"
    echo ""
    echo "Then run: ./start-local.sh"
fi
EOF

chmod +x setup-port-forwards.sh
print_status "Created setup-port-forwards.sh script"

# Final instructions
echo ""
echo "🎉 Setup complete!"
echo ""
echo "Next steps:"
echo "1. Edit .env file and update SLACK_WEBHOOK_URL with your webhook URL"
echo "2. Set up port forwards:"
echo "   ./setup-port-forwards.sh"
echo "3. Start Alert Engine:"
echo "   ./start-local.sh"
echo ""
echo "Or run manually:"
echo "1. oc port-forward -n amq-streams-kafka svc/alert-kafka-cluster-kafka-bootstrap 9092:9092"
echo "2. oc port-forward -n redis-cluster svc/redis-cluster-access 6379:6379"
echo "3. source .env && ./alert-engine"
echo ""
print_info "For detailed instructions, see LOCAL_SETUP_GUIDE.md" 