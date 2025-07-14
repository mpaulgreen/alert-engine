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
