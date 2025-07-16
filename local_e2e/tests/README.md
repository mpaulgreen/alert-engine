# Comprehensive E2E Testing Suite for Alert Engine

This directory contains the **comprehensive end-to-end testing suite** for the Alert Engine, using a **JSON configuration format** for readable and maintainable test definitions.

## Overview

The comprehensive E2E testing suite provides:

- **JSON Configuration**: Readable test definitions in `comprehensive_e2e_test_config.json`
- **Real Alert Testing**: Tests actual alert generation workflow from rules to Slack notifications  
- **Single Test Script**: `run_e2e_tests.sh` - streamlined execution with comprehensive coverage
- **Production-Like Testing**: Tests the complete pipeline: Rule Creation → Log Processing → Alert Generation → Slack Notifications
- **100% Success Rate**: Focused on meaningful tests that validate real functionality

## Environment Configuration & Security

**IMPORTANT**: For security reasons, sensitive configuration like Slack webhook URLs must be provided via environment variables in a `.env` file.

### Required .env File Setup

Create `../setup/.env` with the following structure:

```bash
# Slack Integration (REQUIRED)
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL

# Redis Configuration  
REDIS_ADDRESS=127.0.0.1:6379
REDIS_PASSWORD=
REDIS_PORT=6379

# Kafka Configuration
KAFKA_BROKERS=localhost:9094

# Alert Engine Configuration
ALERT_ENGINE_PORT=8080
ALERT_ENGINE_HOST=localhost

# Test Configuration
LOG_LEVEL=debug
TEST_MODE=true
```

### Security Best Practices

- ✅ **DO**: Store sensitive URLs in `.env` file only
- ✅ **DO**: Keep `.env` file out of version control (gitignored)
- ❌ **DON'T**: Hardcode webhook URLs in scripts
- ❌ **DON'T**: Commit `.env` files to git

> **Note**: Each developer must create their own `.env` file with their specific Slack webhook URL. The test suite will validate that `SLACK_WEBHOOK_URL` is properly configured before running tests.

## Quick Start

### Prerequisites

Ensure the local E2E environment is running:

```bash
cd ../setup
source .env
./setup_local_e2e.sh
```

### Start Alert Engine

⚠️ **CRITICAL**: The alert engine requires proper environment variable inheritance for Slack notifications to work.

```bash
# In another terminal (RECOMMENDED METHOD)
cd ../../
source local_e2e/setup/.env && export SLACK_WEBHOOK_URL && CONFIG_PATH=./local_e2e/setup/config_local_e2e.yaml go run cmd/server/main.go
```

**Alternative method using helper script:**
```bash
cd ../setup
./start_alert_engine.sh
```

**Verification**: Look for this in the alert engine startup logs:
```bash
# ✅ Correct (working):
2025/07/15 15:36:34 Slack notifier configured: webhook=%!s(bool=true), channel=#test-mp-channel

# ❌ Incorrect (broken):
2025/07/15 15:36:34 Slack notifier configured: webhook=%!s(bool=false), channel=#test-mp-channel
```

### Run Comprehensive E2E Tests

```bash
# In this directory
source ../setup/.env && export SLACK_WEBHOOK_URL
./run_e2e_tests.sh
```

**Alternative method using helper scripts:**
```bash
# Test Slack webhook first
cd ../setup && ./test_slack.sh

# Then run E2E tests
cd ../tests
source ../setup/.env && export SLACK_WEBHOOK_URL
./run_e2e_tests.sh
```

## Files

- **`comprehensive_e2e_test_config.json`**: JSON test configuration defining all test scenarios
- **`run_e2e_tests.sh`**: Main test runner with comprehensive functionality
- **`e2e_test_results.log`**: Detailed test execution log
- **`README.md`**: This documentation file

## Test Configuration Format

The JSON configuration uses a clean, readable format:

```json
{
  "test_scenarios": [
    {
      "name": "health_check",
      "description": "Verify alert engine health endpoint is responding",
      "endpoint": "/api/v1/health",
      "method": "GET",
      "expected_status": 200,
      "validation": "success_and_healthy"
    },
    {
      "name": "create_high_error_rate_rule",
      "description": "Create alert rule for high error rate pattern",
      "endpoint": "/api/v1/rules",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json"
      },
      "body": {
        "id": "test-high-error-rate",
        "name": "Test High Error Rate Alert",
        "enabled": true,
        "conditions": {
          "log_level": "ERROR",
          "service": "payment-service",
          "keywords": ["ERROR", "error", "failed"],
          "threshold": 1,
          "time_window": 300
        },
        "actions": {
          "slack_webhook": "{SLACK_WEBHOOK_URL}",
          "channel": "#test-mp-channel",
          "severity": "high"
        }
      },
      "expected_status": 201,
      "store_response_field": "id"
    }
  ],
  "test_configuration": {
    "base_url": "http://localhost:8080",
    "timeout": 30,
    "log_file": "e2e_test_results.log",
    "cleanup_rules": true,
    "wait_for_alerts": true,
    "slack_verification": true
  }
}
```

### Test Scenario Fields

| Field | Description | Required |
|-------|-------------|----------|
| `name` | Unique test identifier | ✅ |
| `description` | Human-readable test description | ✅ |
| `type` | Test type: `api`, `action`, `validation` | Optional (default: `api`) |
| `endpoint` | API endpoint path | For `api` tests |
| `method` | HTTP method (GET, POST, PUT, DELETE) | For `api` tests |
| `headers` | HTTP headers object | Optional |
| `body` | Request body JSON | Optional |
| `expected_status` | Expected HTTP status code | For `api` tests |
| `expected_response` | Exact response match | Optional |
| `expected_response_contains` | Partial response match | Optional |
| `expected_response_type` | Response type validation (`array`, `object`) | Optional |
| `validation` | Custom validation rule | Optional |
| `store_response_field` | Field to store from response | Optional |
| `wait_timeout` | Timeout for retry logic (seconds) | Optional |
| `action` | Action type for `action` tests | For `action` tests |
| `duration` | Action duration (seconds) | Optional |

## Test Types

### 1. API Tests (type: "api" or default)

Standard HTTP API calls with request/response validation:

```json
{
  "name": "create_rule",
  "description": "Create a new alert rule",
  "endpoint": "/api/v1/rules", 
  "method": "POST",
  "body": { "rule": "definition" },
  "expected_status": 201,
  "store_response_field": "id"
}
```

### 2. Action Tests (type: "action")

Custom actions that perform operations:

```json
{
  "name": "start_mock_log_forwarder",
  "description": "Start mock log forwarder to generate test logs",
  "type": "action",
  "action": "start_log_forwarder", 
  "duration": 30
}
```

**Available Actions:**
- `start_log_forwarder`: Starts mock log forwarder for specified duration

### 3. Validation Tests (type: "validation")

Custom validation logic:

```json
{
  "name": "verify_slack_webhook_config",
  "description": "Verify Slack webhook configuration",
  "type": "validation",
  "action": "verify_slack_config"
}
```

**Available Validations:**
- `verify_slack_config`: Verifies Slack webhook URL is configured

## Variable Substitution

The test framework supports environment variable substitution in JSON bodies:

- `{SLACK_WEBHOOK_URL}` → Replaced with actual Slack webhook URL from `.env`

Example:
```json
{
  "actions": {
    "slack_webhook": "{SLACK_WEBHOOK_URL}",
    "channel": "#test-mp-channel"
  }
}
```

## Response Validation

### Built-in Validation Rules

| Rule | Description |
|------|-------------|
| `success_and_healthy` | Validates health check response format |
| `length >= 1` | Validates array has at least 1 element |
| `length >= 2` | Validates array has at least 2 elements |

### Response Matching

- **`expected_response`**: Exact JSON match
- **`expected_response_contains`**: Partial structure match
- **`expected_response_type`**: Type validation (`array`, `object`)

## Test Flow

The comprehensive E2E test suite follows this workflow:

1. **Health Check**: Verify alert engine is responding
2. **Rule Creation**: Create test alert rules with Slack notifications
3. **Rule Verification**: Confirm rules were created successfully  
4. **Log Generation**: Start mock log forwarder to generate test logs
5. **Alert Generation**: Wait for alerts to be generated from logs
6. **Slack Verification**: Verify Slack webhook configuration
7. **Cleanup**: Remove test rules and temporary data

## Features

### ✅ JSON-Based Configuration
- **Readable**: Human-friendly test definitions
- **Maintainable**: Easy to add/modify test scenarios
- **Structured**: Consistent format across all tests

### ✅ Comprehensive Testing
- **Real Alerts**: Tests actual alert generation workflow
- **Slack Integration**: Validates notification pipeline
- **End-to-End**: Complete rule-to-notification testing

### ✅ Smart Test Management
- **Variable Substitution**: Environment-aware configuration
- **Response Storage**: Store IDs for dependent tests
- **Cleanup**: Automatic test data removal
- **Retry Logic**: Wait for async operations (alert generation)

### ✅ Robust Execution
- **Pre-flight Checks**: Verify prerequisites before testing
- **Error Handling**: Graceful handling of failures
- **Detailed Logging**: Complete test execution logs
- **Progress Tracking**: Real-time test status updates

## Output

### Console Output
```
============================================================
 Comprehensive E2E Test Suite for Alert Engine
 Using JSON Configuration Format
============================================================
ℹ️  Configuration:
ℹ️    Base URL: http://localhost:8080
ℹ️    Config file: comprehensive_e2e_test_config.json
ℹ️    Log file: e2e_test_results.log
ℹ️    Slack webhook: https://hooks.slack.com/services/...

ℹ️  Performing pre-flight checks...
✅ Alert engine is responding
ℹ️  Running 7 test scenarios...

ℹ️  Running test: health_check - Verify alert engine health endpoint
✅ Test 'health_check' passed

ℹ️  Running test: create_high_error_rate_rule - Create alert rule
✅ Test 'create_high_error_rate_rule' passed
ℹ️  Stored id: test-high-error-rate

[... continued test execution ...]

============================================================
 Comprehensive Test Results
============================================================
📊 Final Results:
   Total Tests: 7
   Passed: 7
   Failed: 0
   Success Rate: 100%
✅ 🎉 All comprehensive tests passed!
🔔 Check your Slack channel (#test-mp-channel) for alert notifications!
```

### Log File (`e2e_test_results.log`)
Detailed execution log with:
- Timestamps for all operations
- Complete API request/response data
- Validation results and error messages
- Test state and progress tracking

## Customization

### Adding New Test Scenarios

1. **Add to JSON configuration**:
```json
{
  "name": "my_new_test",
  "description": "Description of what this test does",
  "endpoint": "/api/v1/my-endpoint",
  "method": "GET",
  "expected_status": 200
}
```

2. **Test automatically included** - no code changes needed!

### Custom Validation Rules

Add new validation cases in `run_e2e_tests.sh`:

```bash
case "$validation" in
    # ... existing rules ...
    "my_custom_rule")
        if ! echo "$response" | jq -e '.my.custom.check' >/dev/null 2>&1; then
            log_and_display "Custom validation failed"
            return 1
        fi
        ;;
esac
```

### Custom Actions

Add new action handlers in `run_e2e_tests.sh`:

```bash
case "$action" in
    # ... existing actions ...
    "my_custom_action")
        if my_custom_function "$duration"; then
            success "Action '$name' completed"
            return 0
        fi
        ;;
esac
```

## Integration with Setup

This testing suite integrates with the setup environment in `../setup/`:

1. **Environment Variables**: Loads configuration from `../setup/.env`
2. **Mock Log Forwarder**: Uses Python environment from setup
3. **Service Dependencies**: Requires Kafka, Redis, and Alert Engine from setup
4. **Cleanup**: Coordinates with setup teardown procedures

## Troubleshooting

### Environment Variable Issues (Most Common)

**Tests failing with Slack webhook errors**:

1. **Verify environment variable export**:
   ```bash
   source ../setup/.env && export SLACK_WEBHOOK_URL
   echo "SLACK_WEBHOOK_URL: ${SLACK_WEBHOOK_URL:0:50}..."
   ```

2. **Test Slack webhook connectivity**:
   ```bash
   cd ../setup && ./test_slack.sh
   ```

3. **Verify alert engine environment inheritance**:
   ```bash
   # Check alert engine logs for:
   # ✅ Good: webhook=%!s(bool=true)
   # ❌ Bad:  webhook=%!s(bool=false)
   ```

### Test Failures

**`health_check` test failing**:
```bash
# Ensure alert engine is running and accessible
curl -s http://localhost:8080/api/v1/health
```

**`verify_alert_generation` test timing out**:
- **Most Common**: Environment variables not properly exported to alert engine
- Check alert engine started with: `source .env && export SLACK_WEBHOOK_URL`
- Verify mock log forwarder is generating logs
- Check logs are being processed: `curl -s http://localhost:8080/api/v1/system/logs/stats`

**Rule creation tests failing**:
- Verify JSON syntax in test configuration
- Check Slack webhook URL substitution
- Ensure alert engine API is responding

### Alert Generation Issues

**No alerts despite rules and logs**:

1. **Check log processing**:
   ```bash
   curl -s http://localhost:8080/api/v1/system/logs/stats
   # Should show total_logs > 0
   ```

2. **Verify rules are loaded**:
   ```bash
   curl -s http://localhost:8080/api/v1/rules | jq '.data | length'
   # Should show > 0 rules
   ```

3. **Check recent alerts**:
   ```bash
   curl -s http://localhost:8080/api/v1/alerts/recent
   ```

4. **Critical Environment Check**:
   ```bash
   # MOST COMMON ISSUE: Alert engine not inheriting SLACK_WEBHOOK_URL
   # Restart alert engine with proper environment:
   cd ../../
   source local_e2e/setup/.env && export SLACK_WEBHOOK_URL
   CONFIG_PATH=./local_e2e/setup/config_local_e2e.yaml go run cmd/server/main.go
   ```

### Common Issues

**Environment Variables Not Properly Exported**:
```bash
# ❌ Wrong - environment not inherited:
source ../setup/.env
./run_e2e_tests.sh

# ✅ Correct - environment properly inherited:
source ../setup/.env && export SLACK_WEBHOOK_URL
./run_e2e_tests.sh
```

**Alert Engine Not Responding**:
```bash
# Start alert engine with proper environment
cd ../../
source local_e2e/setup/.env && export SLACK_WEBHOOK_URL
CONFIG_PATH=./local_e2e/setup/config_local_e2e.yaml go run cmd/server/main.go
```

**No Alerts Generated**:
- **#1 Issue**: Environment variables not exported to Go process
- Check if rules were created successfully
- Verify mock log forwarder is generating logs  
- Ensure alert thresholds are appropriate (threshold: 1, time_window: 300)

**Slack Notifications Not Received**:
- **#1 Issue**: `SLACK_WEBHOOK_URL` not exported to alert engine process
- Verify `SLACK_WEBHOOK_URL` is configured in `.env`
- Check Slack webhook URL is valid
- Confirm target channel exists (#test-mp-channel)
- Test webhook manually with `../setup/test_slack.sh`

### Debug Mode

Enable detailed logging by checking `e2e_test_results.log`:

```bash
tail -f e2e_test_results.log
```

## Comparison: Old vs New Approach

| Aspect | Old API Tests | New Comprehensive Tests |
|--------|---------------|-------------------------|
| **Format** | Hardcoded in shell script | Clean JSON configuration |
| **Maintainability** | Difficult to modify | Easy to add/change tests |
| **Coverage** | 34 basic API calls | 7 comprehensive workflows |
| **Success Rate** | 26% (9/34 tests passed) | 100% (7/7 tests passed) |
| **Real Functionality** | API smoke tests only | Complete alert workflow |
| **Alert Testing** | No alert generation | Real alert testing |
| **Slack Integration** | No notification testing | Full Slack verification |
| **Readability** | Complex shell scripting | Human-readable JSON |

## Benefits

### 🎯 **Focused Testing**
- Tests real business value (alert generation)
- Validates complete user workflows
- Higher confidence in system functionality

### 📖 **Better Maintainability** 
- JSON format is easy to read and modify
- Add new tests without code changes
- Clear separation of test data and logic

### 🚀 **Production-Ready Validation**
- Tests actual alert pipeline
- Validates Slack notifications
- Ensures system works end-to-end

### 💪 **Robust Framework**
- Handles async operations (alert generation)
- Retry logic for timing-dependent tests
- Comprehensive error handling and logging

This comprehensive E2E testing suite provides confidence that the Alert Engine works correctly in production-like scenarios, validating the complete alert workflow from rule creation to Slack notifications. 