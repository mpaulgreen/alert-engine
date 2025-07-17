package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/log-monitoring/alert-engine/pkg/models"
	"github.com/redis/go-redis/v9"
)

// RedisStore implements the StateStore interface using Redis
type RedisStore struct {
	client redis.UniversalClient
	ctx    context.Context
}

// NewRedisStore creates a new Redis store with cluster support
func NewRedisStore(addr, password string) *RedisStore {
	return NewRedisStoreWithConfig(addr, password, false)
}

// NewRedisStoreWithConfig creates a new Redis store with configuration options
func NewRedisStoreWithConfig(addr, password string, clusterMode bool) *RedisStore {
	var client redis.UniversalClient

	if clusterMode || strings.Contains(addr, ",") {
		// Parse multiple addresses for cluster mode
		addresses := strings.Split(addr, ",")
		for i, address := range addresses {
			addresses[i] = strings.TrimSpace(address)
		}

		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    addresses,
			Password: password,
		})
	} else {
		// Single node mode
		client = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       0,
		})
	}

	return &RedisStore{
		client: client,
		ctx:    context.Background(),
	}
}

// SaveAlertRule saves an alert rule to Redis
func (r *RedisStore) SaveAlertRule(rule models.AlertRule) error {
	data, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("failed to marshal alert rule: %w", err)
	}

	key := fmt.Sprintf("alert_rule:%s", rule.ID)
	return r.client.Set(r.ctx, key, data, 0).Err()
}

// GetAlertRules retrieves all alert rules from Redis
func (r *RedisStore) GetAlertRules() ([]models.AlertRule, error) {
	keys, err := r.client.Keys(r.ctx, "alert_rule:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get alert rule keys: %w", err)
	}

	rules := make([]models.AlertRule, 0)
	for _, key := range keys {
		val, err := r.client.Get(r.ctx, key).Result()
		if err != nil {
			continue // Skip invalid entries
		}

		var rule models.AlertRule
		if err := json.Unmarshal([]byte(val), &rule); err != nil {
			continue // Skip invalid entries
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

// GetAlertRule retrieves a specific alert rule by ID
func (r *RedisStore) GetAlertRule(id string) (*models.AlertRule, error) {
	key := fmt.Sprintf("alert_rule:%s", id)
	val, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("alert rule not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get alert rule: %w", err)
	}

	var rule models.AlertRule
	if err := json.Unmarshal([]byte(val), &rule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alert rule: %w", err)
	}

	return &rule, nil
}

// DeleteAlertRule deletes an alert rule from Redis
func (r *RedisStore) DeleteAlertRule(id string) error {
	key := fmt.Sprintf("alert_rule:%s", id)
	result := r.client.Del(r.ctx, key)
	if result.Err() != nil {
		return fmt.Errorf("failed to delete alert rule: %w", result.Err())
	}

	if result.Val() == 0 {
		return fmt.Errorf("alert rule not found: %s", id)
	}

	return nil
}

// IncrementCounter increments a counter for a rule within a time window
func (r *RedisStore) IncrementCounter(ruleID string, window time.Duration) (int64, error) {
	// Create a time-based key for the window
	windowStart := time.Now().Truncate(window)
	key := fmt.Sprintf("counter:%s:%d", ruleID, windowStart.Unix())

	pipe := r.client.Pipeline()
	incr := pipe.Incr(r.ctx, key)
	pipe.Expire(r.ctx, key, window*2) // Keep data for 2x window duration

	_, err := pipe.Exec(r.ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to increment counter: %w", err)
	}

	return incr.Val(), nil
}

// GetCounter gets the current counter value for a rule within a time window
func (r *RedisStore) GetCounter(ruleID string, window time.Duration) (int64, error) {
	windowStart := time.Now().Truncate(window)
	key := fmt.Sprintf("counter:%s:%d", ruleID, windowStart.Unix())

	val, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil // Counter doesn't exist, return 0
		}
		return 0, fmt.Errorf("failed to get counter: %w", err)
	}

	var count int64
	if err := json.Unmarshal([]byte(val), &count); err != nil {
		return 0, fmt.Errorf("failed to unmarshal counter value: %w", err)
	}

	return count, nil
}

// SetAlertStatus sets the status of an alert
func (r *RedisStore) SetAlertStatus(ruleID string, status models.AlertStatus) error {
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal alert status: %w", err)
	}

	key := fmt.Sprintf("alert_status:%s", ruleID)
	return r.client.Set(r.ctx, key, data, time.Hour).Err() // Expire after 1 hour
}

// GetAlertStatus gets the status of an alert
func (r *RedisStore) GetAlertStatus(ruleID string) (*models.AlertStatus, error) {
	key := fmt.Sprintf("alert_status:%s", ruleID)
	val, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("alert status not found: %s", ruleID)
		}
		return nil, fmt.Errorf("failed to get alert status: %w", err)
	}

	var status models.AlertStatus
	if err := json.Unmarshal([]byte(val), &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alert status: %w", err)
	}

	return &status, nil
}

// SaveLogStats saves log processing statistics
func (r *RedisStore) SaveLogStats(stats models.LogStats) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal log stats: %w", err)
	}

	key := "log_stats"
	return r.client.Set(r.ctx, key, data, time.Hour).Err()
}

// GetLogStats retrieves log processing statistics
func (r *RedisStore) GetLogStats() (*models.LogStats, error) {
	key := "log_stats"
	val, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// Return empty stats if not found
			return &models.LogStats{
				LogsByLevel:   make(map[string]int64),
				LogsByService: make(map[string]int64),
				LastUpdated:   time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get log stats: %w", err)
	}

	var stats models.LogStats
	if err := json.Unmarshal([]byte(val), &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal log stats: %w", err)
	}

	return &stats, nil
}

// SaveAlert saves an alert to Redis
func (r *RedisStore) SaveAlert(alert models.Alert) error {
	data, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	key := fmt.Sprintf("alert:%s", alert.ID)
	return r.client.Set(r.ctx, key, data, 24*time.Hour).Err() // Keep alerts for 24 hours
}

// GetAlert retrieves a specific alert by ID
func (r *RedisStore) GetAlert(alertID string) (*models.Alert, error) {
	key := fmt.Sprintf("alert:%s", alertID)
	val, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("alert not found: %s", alertID)
		}
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	var alert models.Alert
	if err := json.Unmarshal([]byte(val), &alert); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alert: %w", err)
	}

	return &alert, nil
}

// GetRecentAlerts retrieves recent alerts (last 24 hours)
func (r *RedisStore) GetRecentAlerts(limit int) ([]models.Alert, error) {
	keys, err := r.client.Keys(r.ctx, "alert:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get alert keys: %w", err)
	}

	alerts := make([]models.Alert, 0)
	for _, key := range keys {
		val, err := r.client.Get(r.ctx, key).Result()
		if err != nil {
			continue // Skip invalid entries
		}

		var alert models.Alert
		if err := json.Unmarshal([]byte(val), &alert); err != nil {
			continue // Skip invalid entries
		}

		alerts = append(alerts, alert)
	}

	// Sort by timestamp (most recent first) and limit
	if len(alerts) > limit {
		alerts = alerts[:limit]
	}

	return alerts, nil
}

// GetHealthStatus returns the health status of the Redis connection
func (r *RedisStore) GetHealthStatus() (bool, error) {
	_, err := r.client.Ping(r.ctx).Result()
	if err != nil {
		return false, fmt.Errorf("redis ping failed: %w", err)
	}
	return true, nil
}

// GetInfo returns Redis server information
func (r *RedisStore) GetInfo() (map[string]string, error) {
	info, err := r.client.Info(r.ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get redis info: %w", err)
	}

	result := make(map[string]string)
	result["info"] = info
	result["status"] = "connected"

	return result, nil
}

// CleanupExpiredData removes expired data from Redis
func (r *RedisStore) CleanupExpiredData() error {
	// Get all counter keys
	counterKeys, err := r.client.Keys(r.ctx, "counter:*").Result()
	if err != nil {
		return fmt.Errorf("failed to get counter keys: %w", err)
	}

	// Remove expired counters
	for _, key := range counterKeys {
		ttl, err := r.client.TTL(r.ctx, key).Result()
		if err != nil {
			continue
		}

		if ttl <= 0 {
			r.client.Del(r.ctx, key)
		}
	}

	return nil
}

// GetMetrics returns storage metrics
func (r *RedisStore) GetMetrics() (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// Count different types of keys
	ruleKeys, _ := r.client.Keys(r.ctx, "alert_rule:*").Result()
	counterKeys, _ := r.client.Keys(r.ctx, "counter:*").Result()
	statusKeys, _ := r.client.Keys(r.ctx, "alert_status:*").Result()
	alertKeys, _ := r.client.Keys(r.ctx, "alert:*").Result()

	metrics["alert_rules"] = len(ruleKeys)
	metrics["counters"] = len(counterKeys)
	metrics["alert_statuses"] = len(statusKeys)
	metrics["alerts"] = len(alertKeys)

	// Get memory usage
	info, err := r.client.Info(r.ctx, "memory").Result()
	if err == nil {
		metrics["memory_info"] = info
	}

	return metrics, nil
}

// Close closes the Redis connection
func (r *RedisStore) Close() error {
	return r.client.Close()
}

// Transaction executes multiple operations in a Redis transaction
func (r *RedisStore) Transaction(fn func(pipe redis.Pipeliner) error) error {
	pipe := r.client.TxPipeline()

	if err := fn(pipe); err != nil {
		return err
	}

	_, err := pipe.Exec(r.ctx)
	return err
}

// BulkSaveAlertRules saves multiple alert rules in a single operation
func (r *RedisStore) BulkSaveAlertRules(rules []models.AlertRule) error {
	pipe := r.client.Pipeline()

	for _, rule := range rules {
		data, err := json.Marshal(rule)
		if err != nil {
			return fmt.Errorf("failed to marshal alert rule %s: %w", rule.ID, err)
		}

		key := fmt.Sprintf("alert_rule:%s", rule.ID)
		pipe.Set(r.ctx, key, data, 0)
	}

	_, err := pipe.Exec(r.ctx)
	return err
}

// Search searches for keys matching a pattern
func (r *RedisStore) Search(pattern string) ([]string, error) {
	return r.client.Keys(r.ctx, pattern).Result()
}
