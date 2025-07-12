# Scripts Documentation

This directory contains scripts for running various tests and operations for the Alert Engine project.

## Integration Test Script

The `run_integration_tests.sh` script provides comprehensive integration testing using Docker containers for external dependencies (Kafka, Redis, Zookeeper).

### Quick Start

```bash
# Show help and usage information
./scripts/run_integration_tests.sh --help

# Run integration tests with full Docker setup
./scripts/run_integration_tests.sh

# Skip health checks (faster, bypasses Docker connectivity issues)
SKIP_HEALTH_CHECK=true ./scripts/run_integration_tests.sh
```

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
**Purpose**: Run complete integration test suite with Docker containers
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
- Faster execution (bypasses potential Docker networking issues)
- Still uses containers but doesn't wait for health confirmation
- Recommended when Docker health checks fail but containers are working

**Expected Duration**: 3-5 minutes

### Test Coverage

The integration test script covers:

| **Package** | **Test Count** | **Timeout** | **Description** |
|-------------|---------------|-------------|-----------------|
| **API Tests** | ~34 tests | 5m | HTTP server testing, CRUD operations, performance |
| **Kafka Tests** | ~11 tests | 5m | Message processing, consumer groups, error handling |
| **Storage Tests** | ~19 tests | 5m | Redis operations, testcontainers, data persistence |
| **Notifications Tests** | ~27 tests | 3m | Slack integration, mock HTTP server, retry logic |

### Performance Benchmarks

The script includes performance testing for:
- **API Endpoints**: 17,000+ requests/second
- **Kafka Processing**: Message throughput testing
- **Storage Operations**: Redis read/write performance
- **Notification Delivery**: Slack webhook performance

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
   # Solution: Skip health checks
   SKIP_HEALTH_CHECK=true ./scripts/run_integration_tests.sh
   ```

2. **Docker Container Startup Issues**
   ```bash
   # Check container status
   docker ps -a | grep alert-engine-test
   
   # Clean up stale containers
   docker-compose -f docker-compose.test.yml -p alert-engine-test down -v
   ```

3. **Port Conflicts**
   ```bash
   # Check if ports are in use
   netstat -an | grep -E "(9092|9093|6379)"
   
   # Stop conflicting services
   docker stop $(docker ps -q --filter "name=kafka")
   ```

#### Alternative Direct Testing

If the script continues to fail, run tests directly:

```bash
# Run all integration tests directly (no containers)
go test -tags=integration -v ./internal/api/tests/... -timeout=5m
go test -tags=integration -v ./internal/notifications/tests/... -timeout=3m
go test -tags=integration -v ./internal/storage/tests/... -timeout=5m
go test -tags=integration -v ./internal/kafka/tests/... -timeout=5m
```

### Expected Output

#### Successful Run
```
🚀 Starting Alert Engine Integration Tests...
✅ Docker compose file found
✅ Test containers started
✅ Service health checks passed
✅ API integration tests PASSED (34/34)
✅ Kafka integration tests PASSED (11/11)
✅ Storage integration tests PASSED (19/19)
✅ Notifications integration tests PASSED (27/27)
✅ All integration tests PASSED! 🎉
```

#### With Health Check Skipped
```
🚀 Starting Alert Engine Integration Tests...
✅ Docker compose file found
✅ Test containers started
⚠️  Skipping health check (SKIP_HEALTH_CHECK=true)
✅ All integration tests PASSED! 🎉
```

### Additional Scripts

- `run_unit_tests.sh` - Executes unit tests across all packages
- Other utility scripts may be added as the project evolves

### Notes

- The script automatically handles container cleanup on exit
- All tests use proper timeouts to prevent hanging
- Performance benchmarks are included but optional
- The script is compatible with both Docker and Podman (auto-detection)

### Support

For issues or questions about the integration test script:
1. Check the troubleshooting section above
2. Review the script's help output: `./scripts/run_integration_tests.sh --help`
3. Run tests directly as documented in the alternative testing section 