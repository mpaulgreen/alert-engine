# Alert Engine - Quick Start Guide

🚀 **Get your Alert Engine running locally in minutes!**

This guide shows you how to run the Alert Engine locally on your Mac while connecting to the Kafka, Redis, and logging infrastructure we set up on OpenShift.

## 📦 What You'll Get

- ✅ Alert Engine running locally on your Mac
- ✅ Connected to OpenShift Kafka cluster for log consumption  
- ✅ Connected to OpenShift Redis cluster for state storage
- ✅ Slack notifications for alerts
- ✅ Full API access for rule management
- ✅ Metrics and monitoring endpoints
- ✅ Comprehensive testing and validation

## 🏃‍♂️ Quick Start (3 Steps)

### Step 1: Run Setup Script
```bash
cd alert-engine/local
./local-setup.sh
```

### Step 2: Configure Slack
```bash
# Edit .env file and update SLACK_WEBHOOK_URL
nano .env

# Update this line with your actual webhook URL:
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

### Step 3: Start Everything
```bash
# Option A: Automated (using tmux)
./setup-port-forwards.sh
./start-local.sh

# Option B: Manual (3 separate terminals)
# Terminal 1: Kafka port forward
oc port-forward -n amq-streams-kafka svc/alert-kafka-cluster-kafka-bootstrap 9092:9092

# Terminal 2: Redis port forward  
oc port-forward -n redis-cluster svc/redis-cluster-access 6379:6379

# Terminal 3: Start Alert Engine
source .env && ./alert-engine
```

## 🧪 Validate Setup

```bash
# Run comprehensive test suite
./test-local-setup.sh

# Quick health check
curl http://localhost:8080/health

# Check if alerts are working
curl -X POST http://localhost:8080/api/v1/test-alert \
  -H "Content-Type: application/json" \
  -d '{"level": "ERROR", "message": "Test alert", "service": "local-test"}'
```

## 📊 What's Running

| Component | Local Port | Purpose |
|-----------|------------|---------|
| Alert Engine API | `:8080` | Main API and health checks |
| Metrics Server | `:8081` | Prometheus metrics |
| Kafka (via port-forward) | `:9092` | Log message consumption |
| Redis (via port-forward) | `:6379` | State storage |

## 🔗 Key URLs

- **Health Check**: http://localhost:8080/health
- **API Documentation**: http://localhost:8080/api/v1/
- **Metrics**: http://localhost:8081/metrics
- **Performance Profiling**: http://localhost:8081/debug/pprof/

## 🎯 Test Your Setup

### Generate Test Alerts
```bash
# Send error logs to trigger alerts (adjust threshold in config.yaml)
for i in {1..5}; do
  echo '{"timestamp":"'$(date -Iseconds)'","level":"ERROR","message":"Test error '$i'","service":"test-service","namespace":"alert-engine"}' | \
  oc exec -i -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- \
  bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic application-logs
  sleep 2
done
```

### Check Alert Rules
```bash
# List current rules
curl http://localhost:8080/api/v1/rules | jq

# Get alert statistics  
curl http://localhost:8080/api/v1/stats | jq

# Create a new rule
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "id": "my-test-rule",
    "name": "My Test Rule",
    "enabled": true,
    "conditions": {
      "log_level": "WARN",
      "threshold": 1,
      "time_window": "1m",
      "operator": "gt"
    },
    "actions": {
      "channel": "#alerts",
      "severity": "low"
    }
  }'
```

## 📁 Files Created

The setup script creates these helpful files:

- **`.env`** - Environment variables (edit with your Slack webhook)
- **`start-local.sh`** - Start Alert Engine with all checks
- **`setup-port-forwards.sh`** - Set up OpenShift port forwards
- **`alert-engine`** - Compiled binary ready to run

## 🛠️ Configuration

The config is optimized for local development:

- **Lower alert thresholds** (2-3 instead of 5-10)
- **Shorter time windows** (1-2min instead of 5-10min)  
- **Debug logging** enabled
- **Smaller batch sizes** for faster processing
- **Increased timeouts** for network latency

## 🆘 Troubleshooting

### Common Issues

**❌ "connection refused" errors**
```bash
# Restart port forwards
oc port-forward -n amq-streams-kafka svc/alert-kafka-cluster-kafka-bootstrap 9092:9092
oc port-forward -n redis-cluster svc/redis-cluster-access 6379:6379
```

**❌ No Slack notifications**
```bash
# Test webhook directly
curl -X POST -H 'Content-type: application/json' \
  --data '{"text":"Test message"}' \
  $SLACK_WEBHOOK_URL
```

**❌ Alert Engine won't start**
```bash
# Check logs
cat /tmp/alert-engine-local.log

# Verify config
go run ./cmd/server/main.go --config ./configs/config.yaml --validate
```

### Get Help

1. **Run the test script**: `./test-local-setup.sh`
2. **Check detailed guide**: [LOCAL_SETUP_GUIDE.md](LOCAL_SETUP_GUIDE.md)
3. **View logs**: `tail -f /tmp/alert-engine-local.log`
4. **Check port forwards**: `lsof -i :9092` and `lsof -i :6379`

## 🎉 Success Checklist

Your setup is working when:

- ✅ Health check returns `{"status":"healthy"}`
- ✅ Test logs appear in Alert Engine output
- ✅ Error logs trigger Slack notifications
- ✅ Redis stores alert state successfully
- ✅ API endpoints respond correctly
- ✅ Metrics are available

## 🔄 Daily Workflow

```bash
# Start development session
./setup-port-forwards.sh
./start-local.sh

# Run tests
./test-local-setup.sh

# Monitor logs
tail -f /tmp/alert-engine-local.log

# Stop everything
# Ctrl+C in Alert Engine terminal
# tmux kill-session -t alert-engine-pf  # if using tmux
```

---

**🎊 That's it!** Your Alert Engine is now running locally with full OpenShift infrastructure integration.

For detailed information, see [LOCAL_SETUP_GUIDE.md](LOCAL_SETUP_GUIDE.md) 