# Storage Package

The storage package provides Redis-based persistence for the Alert Engine system. It implements state management for alert rules, alerts, counters, and metrics.

## Overview

This package contains:
- **Redis Store**: Main implementation using Redis for data persistence
- **Unit Tests**: Tests for data validation, JSON marshaling, and configuration
- **Integration Tests**: Tests requiring actual Redis instance (with fallback support)
- **Test Fixtures**: Sample data for testing

## Components

### RedisStore
The main storage implementation that provides:
- Alert rule management (CRUD operations)
- Alert storage and retrieval
- Counter tracking for time-windowed metrics
- Alert status management
- Log statistics storage
- Health monitoring and metrics

### Test Files
- `redis_test.go`: Unit tests for data models and configuration
- `integration_test.go`: Integration tests requiring Redis
- `redis_container.go`: Test container setup for integration tests
- `test_data.json`: Test fixtures

## Testing

### Unit Tests
Unit tests focus on data validation, JSON marshaling, and configuration without requiring Redis:

```bash
# Run unit tests
go test -tags=unit ./internal/storage/... -v

# Run with coverage
go test -tags=unit -coverprofile=coverage.out ./internal/storage/... && go tool cover -func=coverage.out

# Generate HTML coverage report
go test -tags=unit -coverprofile=coverage.out ./internal/storage/... && go tool cover -html=coverage.out -o coverage.html
```

**Current Coverage**: 5.2% (primarily constructor and configuration logic)

### Integration Tests
Integration tests require a Redis instance and test actual Redis operations:

#### Option 1: Use Docker/Podman (Recommended)
```bash
# Run integration tests with Docker containers
go test -tags=integration ./internal/storage/... -v
```

#### Option 2: Use External Redis
If Docker is not available, you can run tests against an external Redis instance:

```bash
# Set environment variables for external Redis
export REDIS_TEST_ADDR="localhost:6379"
export REDIS_TEST_PASSWORD="your_password"

# Run integration tests
go test -tags=integration ./internal/storage/... -v
```

#### Option 3: Use Test Infrastructure
The project provides Docker Compose setup for testing:

```bash
# Start test infrastructure
cd scripts
docker-compose -f docker-compose.test.yml up -d redis-test

# Wait for Redis to be ready
sleep 5

# Run integration tests against test Redis
REDIS_TEST_ADDR="localhost:6380" REDIS_TEST_PASSWORD="testpass" go test -tags=integration ./internal/storage/... -v

# Stop test infrastructure
docker-compose -f docker-compose.test.yml down
```

### Using the Integration Test Script
The project provides a comprehensive integration test script:

```bash
# Run all integration tests (includes storage tests)
cd scripts
./run_integration_tests.sh
```

## Configuration

### Redis Store Configuration

#### Single Node Configuration
```go
store := storage.NewRedisStore("localhost:6379", "password")
```

#### Cluster Configuration
```go
// Automatic cluster detection (comma-separated addresses)
store := storage.NewRedisStoreWithConfig("node1:6379,node2:6379", "password", false)

// Explicit cluster mode
store := storage.NewRedisStoreWithConfig("node1:6379", "password", true)
```

### Environment Variables for Testing
- `REDIS_TEST_ADDR`: Redis address for integration tests (default: use testcontainers)
- `REDIS_TEST_PASSWORD`: Redis password for integration tests

## Key Features

### Alert Rules
- Save, retrieve, and delete alert rules
- Bulk operations for multiple rules
- Search functionality

### Alerts
- Store triggered alerts
- Retrieve recent alerts
- Status tracking

### Counters
- Time-windowed counter tracking
- Automatic expiration
- Threshold monitoring

### Metrics
- System health monitoring
- Performance metrics
- Redis connection status

## Data Models

The package works with these main data models from `pkg/models`:
- `AlertRule`: Rule definitions for triggering alerts
- `Alert`: Triggered alert instances
- `AlertStatus`: Status tracking for rules
- `LogStats`: Log aggregation statistics

## Error Handling

The storage layer provides consistent error handling:
- Connection errors are wrapped with context
- Not found errors are clearly distinguished
- JSON marshaling errors are handled gracefully
- Timeouts are properly managed

## Development

### Adding New Tests
1. Unit tests go in `redis_test.go` with `//go:build unit` tag
2. Integration tests go in `integration_test.go` with `//go:build integration` tag
3. Use test fixtures from `test_data.json` when possible

### Test Data
Test fixtures are available in `test_data.json` and can be loaded in tests:

```go
// Load test data
data, err := os.ReadFile("test_data.json")
// Parse and use in tests
```

## Known Limitations

1. **Unit Test Coverage**: Currently at 5.2% as most functionality requires Redis
2. **Docker Dependency**: Integration tests work best with Docker/Podman
3. **External Redis**: Limited test isolation when using external Redis instances

## Troubleshooting

### Integration Tests Failing
1. **Docker not available**: Use external Redis with environment variables
2. **Network issues**: Check container networking and firewall settings
3. **Redis connection**: Verify Redis is running and accessible

### Low Test Coverage
- Unit tests focus on data validation and configuration
- Actual Redis operations require integration tests
- Consider using Redis mocks for additional unit test coverage

## Future Improvements

1. Add Redis mock for better unit test coverage
2. Implement connection pooling optimization
3. Add performance benchmarks
4. Enhance error recovery mechanisms
5. Add distributed locking for cluster operations 