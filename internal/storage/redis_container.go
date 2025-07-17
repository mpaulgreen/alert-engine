//go:build integration
// +build integration

package storage

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"encoding/json"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RedisContainer wraps the testcontainers Redis container
type RedisContainer struct {
	Container testcontainers.Container
	Host      string
	Port      int
	Password  string
	ctx       context.Context
	client    *redis.Client
}

// NewRedisContainer creates and starts a new Redis test container
func NewRedisContainer(ctx context.Context, t *testing.T) (*RedisContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		Env: map[string]string{
			"REDIS_PASSWORD": "testpass",
		},
		Cmd: []string{"redis-server", "--requirepass", "testpass"},
		WaitingFor: wait.ForAll(
			wait.ForLog("Ready to accept connections").WithOccurrence(1),
			wait.ForListeningPort("6379/tcp"),
		).WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start Redis container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get Redis host: %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, "6379")
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get Redis port: %w", err)
	}

	port, err := strconv.Atoi(mappedPort.Port())
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to convert Redis port: %w", err)
	}

	redisContainer := &RedisContainer{
		Container: container,
		Host:      host,
		Port:      port,
		Password:  "testpass",
		ctx:       ctx,
	}

	// Create Redis client
	redisContainer.client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: "testpass",
		DB:       0,
	})

	// Test connection
	_, err = redisContainer.client.Ping(ctx).Result()
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return redisContainer, nil
}

// GetConnectionString returns the Redis connection string
func (rc *RedisContainer) GetConnectionString() string {
	return fmt.Sprintf("%s:%d", rc.Host, rc.Port)
}

// GetHost returns the container host
func (rc *RedisContainer) GetHost() string {
	return rc.Host
}

// GetPort returns the mapped Redis port
func (rc *RedisContainer) GetPort() int {
	return rc.Port
}

// GetPassword returns the Redis password
func (rc *RedisContainer) GetPassword() string {
	return rc.Password
}

// GetClient returns the Redis client
func (rc *RedisContainer) GetClient() *redis.Client {
	return rc.client
}

// FlushAll flushes all data from Redis
func (rc *RedisContainer) FlushAll() error {
	return rc.client.FlushAll(rc.ctx).Err()
}

// FlushDB flushes current database
func (rc *RedisContainer) FlushDB() error {
	return rc.client.FlushDB(rc.ctx).Err()
}

// Set sets a key-value pair in Redis
func (rc *RedisContainer) Set(key, value string, expiration time.Duration) error {
	return rc.client.Set(rc.ctx, key, value, expiration).Err()
}

// Get gets a value from Redis
func (rc *RedisContainer) Get(key string) (string, error) {
	return rc.client.Get(rc.ctx, key).Result()
}

// Del deletes a key from Redis
func (rc *RedisContainer) Del(key string) error {
	return rc.client.Del(rc.ctx, key).Err()
}

// Exists checks if a key exists in Redis
func (rc *RedisContainer) Exists(key string) (bool, error) {
	result, err := rc.client.Exists(rc.ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// Keys returns all keys matching a pattern
func (rc *RedisContainer) Keys(pattern string) ([]string, error) {
	return rc.client.Keys(rc.ctx, pattern).Result()
}

// TTL returns the time to live for a key
func (rc *RedisContainer) TTL(key string) (time.Duration, error) {
	return rc.client.TTL(rc.ctx, key).Result()
}

// Incr increments a key's value
func (rc *RedisContainer) Incr(key string) (int64, error) {
	return rc.client.Incr(rc.ctx, key).Result()
}

// Expire sets an expiration time for a key
func (rc *RedisContainer) Expire(key string, expiration time.Duration) error {
	return rc.client.Expire(rc.ctx, key, expiration).Err()
}

// DBSize returns the number of keys in the current database
func (rc *RedisContainer) DBSize() (int64, error) {
	return rc.client.DBSize(rc.ctx).Result()
}

// Info returns Redis server information
func (rc *RedisContainer) Info() (string, error) {
	return rc.client.Info(rc.ctx).Result()
}

// ConfigGet gets a configuration parameter
func (rc *RedisContainer) ConfigGet(parameter string) (map[string]string, error) {
	return rc.client.ConfigGet(rc.ctx, parameter).Result()
}

// TestRedisAvailability tests if Redis is available and responsive
func (rc *RedisContainer) TestRedisAvailability() error {
	// Test basic operations
	testKey := "test:availability"
	testValue := "available"

	// Test SET
	err := rc.Set(testKey, testValue, time.Minute)
	if err != nil {
		return fmt.Errorf("failed to set test key: %w", err)
	}

	// Test GET
	result, err := rc.Get(testKey)
	if err != nil {
		return fmt.Errorf("failed to get test key: %w", err)
	}

	if result != testValue {
		return fmt.Errorf("expected %s, got %s", testValue, result)
	}

	// Test DEL
	err = rc.Del(testKey)
	if err != nil {
		return fmt.Errorf("failed to delete test key: %w", err)
	}

	return nil
}

// GetRedisVersion returns the Redis server version
func (rc *RedisContainer) GetRedisVersion() (string, error) {
	info, err := rc.Info()
	if err != nil {
		return "", fmt.Errorf("failed to get Redis info: %w", err)
	}

	// Parse version from info string
	// Info contains lines like "redis_version:7.0.0"
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "redis_version:") {
			version := strings.TrimPrefix(line, "redis_version:")
			return strings.TrimSpace(version), nil
		}
	}

	return "unknown", nil
}

// LoadTestData loads test data from fixtures
func (rc *RedisContainer) LoadTestData(data map[string]interface{}) error {
	pipe := rc.client.Pipeline()

	for key, value := range data {
		switch v := value.(type) {
		case string:
			pipe.Set(rc.ctx, key, v, 0)
		case map[string]interface{}:
			// Convert to JSON string
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("failed to marshal JSON for key %s: %w", key, err)
			}
			pipe.Set(rc.ctx, key, string(jsonBytes), 0)
		default:
			// Convert to JSON string
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
			}
			pipe.Set(rc.ctx, key, string(jsonBytes), 0)
		}
	}

	_, err := pipe.Exec(rc.ctx)
	return err
}

// WaitForConnection waits for Redis connection to be available
func (rc *RedisContainer) WaitForConnection(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		_, err := rc.client.Ping(rc.ctx).Result()
		if err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for Redis connection")
}

// Cleanup terminates the Redis container
func (rc *RedisContainer) Cleanup() error {
	if rc.client != nil {
		rc.client.Close()
	}
	if rc.Container != nil {
		return rc.Container.Terminate(rc.ctx)
	}
	return nil
}

// NewRedisContainerForTesting creates a Redis container specifically for testing
// This is a convenience function that sets up commonly used testing configurations
func NewRedisContainerForTesting(t *testing.T) (*RedisContainer, error) {
	t.Helper()

	ctx := context.Background()
	container, err := NewRedisContainer(ctx, t)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis container: %w", err)
	}

	// Configure for testing
	err = container.ConfigureForTesting()
	if err != nil {
		container.Cleanup()
		return nil, fmt.Errorf("failed to configure Redis for testing: %w", err)
	}

	return container, nil
}

// ConfigureForTesting configures Redis for optimal testing
func (rc *RedisContainer) ConfigureForTesting() error {
	// Configure Redis for testing (disable persistence, etc.)
	commands := [][]interface{}{
		{"CONFIG", "SET", "save", ""},         // Disable RDB persistence
		{"CONFIG", "SET", "appendonly", "no"}, // Disable AOF persistence
		{"CONFIG", "SET", "timeout", "0"},     // Disable client timeout
	}

	for _, cmd := range commands {
		_, err := rc.client.Do(rc.ctx, cmd...).Result()
		if err != nil {
			return fmt.Errorf("failed to configure Redis: %w", err)
		}
	}

	return nil
}
