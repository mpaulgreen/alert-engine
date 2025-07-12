# Scripts Documentation

This directory contains scripts for running various tests and operations for the Alert Engine project.

## Integration Test Script

The `run_integration_tests.sh` script provides comprehensive integration testing using Docker/Podman containers for external dependencies (Kafka, Redis, Zookeeper).

### Quick Start

```bash
# Show help and usage information
./scripts/run_integration_tests.sh --help

# Run integration tests with full container setup
./scripts/run_integration_tests.sh

# Skip health checks (faster, bypasses container connectivity issues)
SKIP_HEALTH_CHECK=true ./scripts/run_integration_tests.sh
```

### Container Engine Support

The script automatically detects and supports both **Docker** and **Podman**:
- **Docker**: Standard Docker Engine with docker-compose
- **Podman**: Podman with docker-compose compatibility layer
- **Auto-detection**: Script automatically chooses the appropriate container engine
- **Podman machine**: Automatically starts Podman machine if needed

### Available Commands

#### 1. Help Command
```bash
./scripts/run_integration_tests.sh --help
```
**Purpose**: Display usage information, environment variables, and troubleshooting tips

#### 2. Full Integration Tests
```bash
./scripts/run_integration_tests.sh
```
**Purpose**: Run complete integration test suite with containers
- Starts Kafka, Redis, and Zookeeper containers
- Performs health checks on all services
- Runs all integration tests with proper timeouts
- Includes performance benchmarks
- Automatic cleanup after completion

**Expected Duration**: 5-10 minutes (including container startup)

#### 3. Skip Health Check Mode
```bash
SKIP_HEALTH_CHECK=true ./scripts/run_integration_tests.sh
```
**Purpose**: Bypass service health checks and run tests directly
- Skips Kafka/Redis connectivity verification
- Faster execution (bypasses potential networking issues)
- Still uses containers but doesn't wait for health confirmation
- Recommended when health checks fail but containers are working

**Expected Duration**: 3-5 minutes

#### 4. Container Mode
```bash
./scripts/run_integration_tests.sh container
```
**Purpose**: Run tests inside the Go container rather than on the host

#### 5. Performance Mode
```bash
./scripts/run_integration_tests.sh performance
```
**Purpose**: Run both integration and performance tests

#### 6. Logs Mode
```bash
./scripts/run_integration_tests.sh logs
```
**Purpose**: Display container logs for debugging

### Test Coverage

The integration test script covers:

| **Package** | **Test Count** | **Timeout** | **Description** |
|-------------|---------------|-------------|-----------------|
| **API Tests** | 12 test suites | 5m | HTTP server testing, CRUD operations, performance |
| **Kafka Tests** | 5 test suites | 5m | Message processing, consumer groups, error handling |
| **Storage Tests** | 19 test cases | 5m | Redis operations, testcontainers, data persistence |
| **Notifications Tests** | 7 test suites | 3m | Slack integration, mock HTTP server, retry logic |

### Performance Benchmarks

The script includes performance testing with actual results:
- **API Endpoints**: ~17,250 requests/second
- **Kafka Processing**: High volume message processing with testcontainers
- **Storage Operations**: ~161,000 ops/sec (bulk operations), ~6,800 ops/sec (retrievals)
- **Notification Delivery**: Slack webhook performance with mock servers

### Environment Variables

| **Variable** | **Default** | **Description** |
|--------------|-------------|-----------------|
| `COMPOSE_FILE` | `docker-compose.test.yml` | Docker compose configuration file |
| `PROJECT_NAME` | `alert-engine-test` | Container project name prefix |
| `SKIP_HEALTH_CHECK` | `false` | Skip service health verification |

### Troubleshooting

#### Common Issues

1. **Kafka Health Check Failures**
   ```bash
   # Solution: Skip health checks (containers are healthy but networking test fails)
   SKIP_HEALTH_CHECK=true ./scripts/run_integration_tests.sh
   ```

2. **Container Engine Not Found**
   ```bash
   # The script will show this error if neither Docker nor Podman is available
   # Install Docker Desktop or Podman Desktop
   ```

3. **Podman Machine Not Running**
   ```bash
   # Script automatically starts Podman machine, but you can also do it manually:
   podman machine start podman-machine-default
   ```

4. **Port Conflicts**
   ```bash
   # Check if ports are in use
   netstat -an | grep -E "(9093|6380|2182)"
   
   # Clean up stale containers
   docker-compose -f docker-compose.test.yml -p alert-engine-test down -v
   ```

5. **Container Startup Issues**
   ```bash
   # Check container status
   docker ps -a | grep alert-engine-test
   
   # View specific container logs
   ./scripts/run_integration_tests.sh logs
   ```

#### Recent Fixes Applied

- **Fixed Kafka Health Check**: Updated to use internal port 29092 instead of 9092
- **Container Engine Detection**: Now properly supports both Docker and Podman
- **Removed Obsolete Version**: Cleaned up docker-compose.test.yml warnings
- **Enhanced Error Handling**: Better error messages and troubleshooting guidance

#### Alternative Direct Testing

If the script continues to fail, run tests directly:

```bash
# Run all integration tests directly (using testcontainers)
go test -tags=integration -v ./internal/api/tests/... -timeout=5m
go test -tags=integration -v ./internal/notifications/tests/... -timeout=3m
go test -tags=integration -v ./internal/storage/tests/... -timeout=5m
go test -tags=integration -v ./internal/kafka/tests/... -timeout=5m
```

### Expected Output

#### Successful Run
```
==========================================
Alert Engine - Integration Test Suite
==========================================
Using Docker/Podman for container orchestration
✅ Test containers started
✅ All services are healthy
✅ Kafka integration tests PASSED
✅ Storage integration tests PASSED  
✅ Notifications integration tests PASSED
✅ API integration tests PASSED
🎉 Integration tests completed successfully!
```

#### With Health Check Skipped
```
==========================================
Alert Engine - Integration Test Suite
==========================================
Using Docker/Podman for container orchestration
✅ Test containers started
⚠️  Skipping health check (SKIP_HEALTH_CHECK=true)
✅ All integration tests PASSED
🎉 Integration tests completed successfully!
```

### Additional Scripts

- `run_unit_tests.sh` - Executes unit tests across all packages
- Other utility scripts may be added as the project evolves

### Notes

- The script automatically handles container cleanup on exit
- All tests use proper timeouts to prevent hanging
- Performance benchmarks are included but optional
- The script is compatible with both Docker and Podman (auto-detection)
- Uses testcontainers for isolated test environments
- Supports concurrent test execution for better performance

### Support

For issues or questions about the integration test script:
1. Check the troubleshooting section above
2. Review the script's help output: `./scripts/run_integration_tests.sh --help`
3. Run tests directly as documented in the alternative testing section
4. Use `SKIP_HEALTH_CHECK=true` for networking-related issues 