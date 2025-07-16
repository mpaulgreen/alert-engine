# Local E2E Environment Setup

This directory contains the complete setup for running end-to-end testing of the Alert Engine in a local environment with isolated Docker services.

## Overview

The local E2E environment provides:

- **Isolated Kafka cluster** (port 9094) for log ingestion
- **Isolated Redis instance** (port 6379) for state storage  
- **Mock log forwarder** to generate realistic test logs
- **Alert engine configuration** optimized for local testing
- **Automated setup/teardown** scripts for easy management
- **Podman-based containerization** for lightweight container management

## Quick Start

### 1. Setup Environment

```bash
cd alert-engine/local_e2e/setup
source .env
./setup_local_e2e.sh
```

### 2. Start Alert Engine

⚠️ **CRITICAL**: The alert engine requires proper environment variable inheritance for Slack notifications to work.

```bash
# In a new terminal (RECOMMENDED METHOD)
cd alert-engine
source local_e2e/setup/.env && export SLACK_WEBHOOK_URL && CONFIG_PATH=./local_e2e/setup/config_local_e2e.yaml go run cmd/server/main.go
```

**Alternative method using binary:**
```bash
cd alert-engine  
source local_e2e/setup/.env && export SLACK_WEBHOOK_URL && CONFIG_PATH=./local_e2e/setup/config_local_e2e.yaml ./bin/alert-engine
```

### 3. Generate Test Logs

```bash
# In another new terminal
cd alert-engine/local_e2e/setup
source .env && source venv/bin/activate
python3 mock_log_forwarder.py --mode test
```

### 4. Run E2E Tests

```bash
# In another terminal
cd alert-engine/local_e2e/tests
./run_e2e_tests.sh
```

### 5. Cleanup

```bash
cd alert-engine/local_e2e/setup
./teardown_local_e2e.sh
```

## Environment Configuration

### Critical: Environment Variable Handling

⚠️ **IMPORTANT**: The alert engine requires proper environment variable inheritance for Slack notifications to function correctly.

#### The `.env` File

The setup script automatically creates `.env` with the following variables:

```bash
# Slack Configuration (REQUIRED for notifications)
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR_WORKSPACE/YOUR_CHANNEL/YOUR_WEBHOOK_TOKEN

# Infrastructure Configuration
REDIS_ADDRESS=127.0.0.1:6379
KAFKA_BROKERS=localhost:9094
CONFIG_PATH=./local_e2e/setup/config_local_e2e.yaml
```

#### Proper Usage Pattern

**Always use this pattern when starting the alert engine:**

```bash
# Step 1: Source the .env file to load variables
source local_e2e/setup/.env

# Step 2: Explicitly export critical variables for Go process inheritance
export SLACK_WEBHOOK_URL

# Step 3: Start the alert engine with inherited environment
CONFIG_PATH=./local_e2e/setup/config_local_e2e.yaml go run cmd/server/main.go
```

#### Why This Is Required

- **Go Process Inheritance**: Go applications inherit environment variables from their parent shell
- **Slack Webhook Access**: The `SLACK_WEBHOOK_URL` must be accessible to the Go process for notifications
- **Configuration Loading**: The alert engine reads webhook URLs from environment variables as fallback
- **Previous Issue**: Without explicit export, the webhook URL was not passed to the Go process, causing `webhook=%!s(bool=false)`

#### Verification

You can verify proper environment loading by checking the alert engine startup logs:

```bash
# ✅ Correct (working):
2025/07/15 15:36:34 Slack notifier configured: webhook=%!s(bool=true), channel=#test-mp-channel

# ❌ Incorrect (broken):
2025/07/15 15:36:34 Slack notifier configured: webhook=%!s(bool=false), channel=#test-mp-channel
```

## Directory Structure

```
setup/
├── .env                           # Environment variables (auto-generated)
├── config_local_e2e.yaml         # Alert engine configuration
├── docker-compose-local-e2e.yml  # Docker services
├── mock_log_forwarder.py          # Python log generator
├── requirements.txt               # Python dependencies
├── setup_local_e2e.sh            # Setup automation (includes env export)
├── start_alert_engine.sh          # Helper: Start alert engine with proper env vars
├── teardown_local_e2e.sh         # Cleanup automation
├── test_slack.sh                  # Helper: Test Slack webhook connectivity
└── README.md                      # This file
```

## Components

### Podman Services

The `docker-compose-local-e2e.yml` provides:

- **Zookeeper** (port 2182): Kafka coordination
- **Kafka** (port 9094): Log message broker
- **Redis** (port 6379): Alert state storage
- **Kafka UI** (port 8081): Debug interface (optional)
- **Redis Commander** (port 8082): Debug interface (optional)

### Mock Log Forwarder

The `mock_log_forwarder.py` script generates realistic logs for all 11 supported alert patterns:

1. **High Error Rate**: ERROR level logs exceeding threshold
2. **Payment Failures**: Payment service specific failures
3. **Database Errors**: Database connection and query issues
4. **Authentication Failures**: Auth service login failures
5. **Service Timeouts**: Various timeout scenarios
6. **Critical Namespace Alerts**: Production critical issues
7. **Inventory Warnings**: Stock level warnings
8. **Notification Failures**: Notification delivery issues
9. **High Warning Rate**: High volume warning patterns
10. **Audit Issues**: Security and compliance alerts
11. **Cross-Service Errors**: Service communication failures

#### Usage Modes

**Test Mode** (recommended for E2E testing):
```bash
python3 mock_log_forwarder.py --mode test
```
Generates one burst for each pattern to trigger alerts.

**Continuous Mode** (for ongoing testing):
```bash
python3 mock_log_forwarder.py --mode continuous
```
Continuously generates logs with periodic pattern bursts.

## Configuration

### Environment Variables

Key environment variables in `.env`:

```bash
CONFIG_PATH=./setup/config_local_e2e.yaml  # Alert engine config
KAFKA_BROKERS=localhost:9094               # Kafka connection
REDIS_HOST=localhost                       # Redis connection
REDIS_PORT=6379                           # Redis port
REDIS_PASSWORD=e2epass                    # Redis password
E2E_MODE=true                             # Enable E2E mode
ENABLE_DEBUG_UI=false                     # Debug UIs
```

### Service Ports

The environment uses the following ports:

| Service | Port | Purpose |
|---------|------|---------|
| Kafka | 9094 | Log message broker |
| Redis | 6379 | State storage |
| Zookeeper | 2182 | Kafka coordination |
| Alert Engine | 8080 | HTTP API |
| Metrics | 9090 | Prometheus metrics |
| Kafka UI | 8081 | Debug interface |
| Redis Commander | 8082 | Debug interface |

## Scripts

### Setup Script (`setup_local_e2e.sh`)

Performs complete environment setup:

1. Checks prerequisites (Podman, Go, Python)
2. Cleans up existing resources
3. Starts Podman services with health checks
4. Creates Kafka topics
5. Sets up Python virtual environment
6. Builds alert engine binary
7. Tests connectivity
8. Displays status and next steps

### Teardown Script (`teardown_local_e2e.sh`)

Cleanup options:

```bash
./teardown_local_e2e.sh                  # Basic cleanup
./teardown_local_e2e.sh --remove-volumes # Also remove data
./teardown_local_e2e.sh --remove-venv    # Also remove Python env
./teardown_local_e2e.sh --remove-logs    # Also remove log files
./teardown_local_e2e.sh --remove-all     # Complete cleanup
./teardown_local_e2e.sh --kill-processes # Kill running processes
```

## Testing Workflow

### Complete E2E Test

1. **Setup**: Run setup script
2. **Start Alert Engine**: In separate terminal
3. **Generate Logs**: Run mock log forwarder in test mode
4. **Verify Alerts**: Check API endpoints for triggered alerts
5. **Run Tests**: Execute automated E2E test suite
6. **Cleanup**: Run teardown script

### Pattern-Specific Testing

Test individual patterns:

```bash
# Generate logs for specific pattern
python3 mock_log_forwarder.py --mode test --pattern payment_failures

# Check for alerts via API
curl http://localhost:8080/api/alerts | jq '.[] | select(.rule_id | contains("payment"))'
```

## Debug and Monitoring

### Enable Debug UIs

Set `ENABLE_DEBUG_UI=true` in `.env` before setup:

- **Kafka UI**: http://localhost:8081 - View topics, messages, consumers
- **Redis Commander**: http://localhost:8082 - Browse Redis data

### Log Monitoring

Monitor alert engine logs:
```bash
tail -f setup.log
```

Monitor Podman service logs:
```bash
podman compose -f docker-compose-local-e2e.yml logs -f kafka-e2e
podman compose -f docker-compose-local-e2e.yml logs -f redis-e2e
```

### API Endpoints

Test alert engine APIs:

```bash
# Health check
curl http://localhost:8080/health

# List all alerts
curl http://localhost:8080/api/alerts

# Get alert statistics
curl http://localhost:8080/api/alerts/stats

# List alert rules
curl http://localhost:8080/api/rules
```

## Troubleshooting

### Environment Variable Issues

**Slack notifications not working** (Most Common Issue):

1. **Check environment variable loading**:
   ```bash
   source .env && echo "SLACK_WEBHOOK_URL: ${SLACK_WEBHOOK_URL:0:50}..."
   ```

2. **Verify webhook URL format**:
   ```bash
   # Should start with: https://hooks.slack.com/services/
   echo $SLACK_WEBHOOK_URL | grep "hooks.slack.com"
   ```

3. **Check alert engine logs for webhook status**:
   ```bash
   # Look for this line in alert engine startup:
   # ✅ Good: webhook=%!s(bool=true)
   # ❌ Bad:  webhook=%!s(bool=false)
   ```

4. **Test webhook manually**:
   ```bash
   source .env
   curl -X POST "$SLACK_WEBHOOK_URL" \
     -H 'Content-Type: application/json' \
     -d '{"text":"Test from local E2E setup"}'
   ```

5. **Verify environment export**:
   ```bash
   # Check if variable is exported to child processes
   source .env && export SLACK_WEBHOOK_URL
   env | grep SLACK_WEBHOOK_URL
   ```

**Alert engine not inheriting environment**:

This is the **#1 cause** of notification failures. Always use:
```bash
source local_e2e/setup/.env && export SLACK_WEBHOOK_URL && CONFIG_PATH=./local_e2e/setup/config_local_e2e.yaml go run cmd/server/main.go
```

**NOT**:
```bash
# ❌ Wrong - environment not inherited:
CONFIG_PATH=./local_e2e/setup/config_local_e2e.yaml go run cmd/server/main.go
```

### Common Issues

**Port conflicts**:
```bash
# Check port usage
lsof -i :9094  # Kafka
lsof -i :6379  # Redis
lsof -i :8080  # Alert Engine
```

**Podman issues**:
```bash
# Reset Podman state
./teardown_local_e2e.sh --remove-all
podman system prune -f
./setup_local_e2e.sh
```

**Kafka connection issues**:
```bash
# Test Kafka connectivity
podman compose -f docker-compose-local-e2e.yml exec kafka-e2e kafka-broker-api-versions --bootstrap-server localhost:9094
```

**Redis connection issues**:
```bash
# Test Redis connectivity
podman compose -f docker-compose-local-e2e.yml exec redis-e2e redis-cli -a e2epass ping
```

### Log Analysis

Check setup logs:
```bash
cat setup.log | grep ERROR
cat teardown.log | grep WARNING
```

Check mock log forwarder output:
```bash
python3 mock_log_forwarder.py --mode test 2>&1 | tee forwarder.log
```

## Integration with E2E Tests

This setup environment integrates with the E2E test suite in `../tests/`:

1. Tests assume this environment is running
2. Tests use the same service endpoints and credentials
3. Tests can trigger specific patterns using the mock log forwarder
4. Tests validate alert generation and API responses

For complete testing workflow, see `../tests/README.md`.

## Performance Considerations

### Resource Usage

The local environment is optimized for testing:

- **Kafka**: 1-hour log retention, single replica
- **Redis**: 256MB memory limit with LRU eviction
- **Podman**: Limited resource allocation for local development

### Scaling

For load testing:

1. Increase Kafka partitions in `docker-compose-local-e2e.yml`
2. Adjust Redis memory limits
3. Run multiple mock log forwarder instances
4. Monitor resource usage with Podman stats

## Security Notes

⚠️ **This environment is for testing only**:

- Uses default passwords (`e2epass`)
- Exposes services on localhost
- No encryption or authentication
- Not suitable for production use

## Contributing

When modifying the setup:

1. Update configuration in `config_local_e2e.yaml`
2. Adjust Docker services in `docker-compose-local-e2e.yml`
3. Update mock patterns in `mock_log_forwarder.py`
4. Test setup/teardown scripts
5. Update this README with changes 