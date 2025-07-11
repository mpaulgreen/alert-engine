//go:build unit
// +build unit

package tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/log-monitoring/alert-engine/internal/kafka"
	"github.com/log-monitoring/alert-engine/internal/kafka/tests/mocks"
	"github.com/log-monitoring/alert-engine/pkg/models"
)

func TestNewConsumer(t *testing.T) {
	t.Run("creates consumer successfully", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := kafka.ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Second,
		}

		consumer := kafka.NewConsumer(config, mockAlertEngine)

		assert.NotNil(t, consumer)
	})

	t.Run("creates consumer with default config", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := kafka.DefaultConsumerConfig()

		consumer := kafka.NewConsumer(config, mockAlertEngine)

		assert.NotNil(t, consumer)
	})
}

func TestConsumerConfig(t *testing.T) {
	t.Run("validates correct config", func(t *testing.T) {
		config := kafka.ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Second,
		}

		// Test that we can create a consumer with this config
		mockAlertEngine := mocks.NewMockAlertEngine()
		consumer := kafka.NewConsumer(config, mockAlertEngine)
		assert.NotNil(t, consumer)
	})

	t.Run("handles empty brokers", func(t *testing.T) {
		config := kafka.ConsumerConfig{
			Brokers: []string{}, // Empty brokers
			Topic:   "test-topic",
			GroupID: "test-group",
		}

		mockAlertEngine := mocks.NewMockAlertEngine()
		// Constructor validates brokers and will panic with empty brokers
		assert.Panics(t, func() {
			kafka.NewConsumer(config, mockAlertEngine)
		})
	})

	t.Run("handles empty topic", func(t *testing.T) {
		config := kafka.ConsumerConfig{
			Brokers: []string{"localhost:9092"},
			Topic:   "", // Empty topic
			GroupID: "test-group",
		}

		mockAlertEngine := mocks.NewMockAlertEngine()
		// Constructor validates topic and will panic with empty topic
		assert.Panics(t, func() {
			kafka.NewConsumer(config, mockAlertEngine)
		})
	})
}

func TestDefaultConsumerConfig(t *testing.T) {
	t.Run("returns valid default config", func(t *testing.T) {
		config := kafka.DefaultConsumerConfig()

		assert.NotEmpty(t, config.Brokers)
		assert.NotEmpty(t, config.Topic)
		assert.NotEmpty(t, config.GroupID)
		assert.Greater(t, config.MinBytes, 0)
		assert.Greater(t, config.MaxBytes, config.MinBytes)
		assert.Greater(t, config.MaxWait, time.Duration(0))
	})
}

func TestConsumerGroup(t *testing.T) {
	t.Run("creates consumer group", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := kafka.DefaultConsumerConfig()

		group := kafka.NewConsumerGroup(config, mockAlertEngine, 3)
		assert.NotNil(t, group)
	})

	t.Run("gets group statistics", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := kafka.DefaultConsumerConfig()

		group := kafka.NewConsumerGroup(config, mockAlertEngine, 2)
		stats := group.GetGroupStats()

		assert.Len(t, stats, 2) // Should have stats for 2 consumers
		for _, stat := range stats {
			assert.NotNil(t, stat)
		}
	})

	t.Run("creates group with different consumer counts", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := kafka.DefaultConsumerConfig()

		// Test various consumer counts
		for _, count := range []int{1, 3, 5, 10} {
			group := kafka.NewConsumerGroup(config, mockAlertEngine, count)
			assert.NotNil(t, group)

			stats := group.GetGroupStats()
			assert.Len(t, stats, count)
		}
	})
}

func TestMessageProcessor(t *testing.T) {
	t.Run("creates message processor", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := kafka.DefaultConsumerConfig()
		consumer := kafka.NewConsumer(config, mockAlertEngine)

		processor := kafka.NewMessageProcessor(consumer, 10, time.Second, mockAlertEngine)
		assert.NotNil(t, processor)
	})

	t.Run("creates processor with different batch sizes", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := kafka.DefaultConsumerConfig()
		consumer := kafka.NewConsumer(config, mockAlertEngine)

		// Test various batch sizes
		for _, batchSize := range []int{1, 5, 10, 100} {
			processor := kafka.NewMessageProcessor(consumer, batchSize, time.Second, mockAlertEngine)
			assert.NotNil(t, processor)
		}
	})

	t.Run("creates processor with different flush intervals", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := kafka.DefaultConsumerConfig()
		consumer := kafka.NewConsumer(config, mockAlertEngine)

		// Test various flush intervals
		intervals := []time.Duration{
			100 * time.Millisecond,
			time.Second,
			5 * time.Second,
			time.Minute,
		}

		for _, interval := range intervals {
			processor := kafka.NewMessageProcessor(consumer, 10, interval, mockAlertEngine)
			assert.NotNil(t, processor)
		}
	})
}

func TestConsumer_BasicOperations(t *testing.T) {
	t.Run("consumer provides statistics", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := kafka.DefaultConsumerConfig()
		consumer := kafka.NewConsumer(config, mockAlertEngine)

		stats := consumer.GetStats()
		assert.NotNil(t, stats)
		// Default stats should have zero values
		assert.Equal(t, int64(0), stats.Messages)
		assert.Equal(t, int64(0), stats.Bytes)
	})

	t.Run("consumer reports health status", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := kafka.DefaultConsumerConfig()
		consumer := kafka.NewConsumer(config, mockAlertEngine)

		healthy := consumer.HealthCheck()
		assert.True(t, healthy) // Should be healthy by default
	})

	t.Run("consumer can be closed", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := kafka.DefaultConsumerConfig()
		consumer := kafka.NewConsumer(config, mockAlertEngine)

		err := consumer.Close()
		assert.NoError(t, err)
	})
}

func TestConsumerEdgeCases(t *testing.T) {
	t.Run("handles very large message config", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()

		config := kafka.ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 10485760, // 10MB
			MaxWait:  time.Second,
		}

		consumer := kafka.NewConsumer(config, mockAlertEngine)
		assert.NotNil(t, consumer)
	})

	t.Run("handles zero time durations", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()

		config := kafka.ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1,
			MaxBytes: 1024,
			MaxWait:  0, // Zero wait time
		}

		// Zero wait time should be acceptable
		consumer := kafka.NewConsumer(config, mockAlertEngine)
		assert.NotNil(t, consumer)
	})

	t.Run("handles nil alert engine gracefully", func(t *testing.T) {
		config := kafka.DefaultConsumerConfig()

		// Constructor doesn't validate alertEngine, but it will panic when used
		consumer := kafka.NewConsumer(config, nil)
		assert.NotNil(t, consumer)

		// The panic would occur when trying to use the alert engine
		// For unit tests, we just verify the constructor doesn't panic
	})
}

func TestMockIntegration(t *testing.T) {
	t.Run("mock alert engine works correctly", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()

		// Test mock functionality
		assert.False(t, mockAlertEngine.WasCalled())
		assert.Equal(t, 0, mockAlertEngine.GetCallCount())
		assert.Nil(t, mockAlertEngine.GetLastEvaluatedLog())

		// Create test log entry
		logEntry := models.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   "Test message",
			Kubernetes: models.KubernetesInfo{
				Namespace: "test",
				Pod:       "test-pod",
			},
		}

		// Evaluate log
		mockAlertEngine.EvaluateLog(logEntry)

		// Verify mock state
		assert.True(t, mockAlertEngine.WasCalled())
		assert.Equal(t, 1, mockAlertEngine.GetCallCount())

		lastLog := mockAlertEngine.GetLastEvaluatedLog()
		assert.NotNil(t, lastLog)
		assert.Equal(t, "Test message", lastLog.Message)
		assert.Equal(t, "test", lastLog.Kubernetes.Namespace)
	})

	t.Run("mock alert engine can be reset", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()

		// Add some data
		logEntry := models.LogEntry{
			Message:    "Test",
			Kubernetes: models.KubernetesInfo{Namespace: "test"},
		}
		mockAlertEngine.EvaluateLog(logEntry)

		assert.True(t, mockAlertEngine.WasCalled())

		// Reset
		mockAlertEngine.Reset()

		// Verify reset state
		assert.False(t, mockAlertEngine.WasCalled())
		assert.Equal(t, 0, mockAlertEngine.GetCallCount())
		assert.Nil(t, mockAlertEngine.GetLastEvaluatedLog())
	})

	t.Run("mock alert engine handles panic simulation", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		mockAlertEngine.SetShouldPanic(true)

		logEntry := models.LogEntry{Message: "Test"}

		// Should panic when configured to do so
		assert.Panics(t, func() {
			mockAlertEngine.EvaluateLog(logEntry)
		})
	})
}

func TestJSONHandling(t *testing.T) {
	t.Run("handles valid JSON log entry", func(t *testing.T) {
		// Test JSON marshaling/unmarshaling
		logEntry := models.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   "Test error message",
			Kubernetes: models.KubernetesInfo{
				Namespace: "production",
				Pod:       "test-pod",
				Container: "test-container",
				Labels: map[string]string{
					"app": "test-service",
				},
			},
			Host: "test-host",
		}

		// Marshal to JSON
		logJSON, err := json.Marshal(logEntry)
		require.NoError(t, err)
		assert.NotEmpty(t, logJSON)

		// Unmarshal back
		var unmarshaledLog models.LogEntry
		err = json.Unmarshal(logJSON, &unmarshaledLog)
		require.NoError(t, err)

		assert.Equal(t, logEntry.Level, unmarshaledLog.Level)
		assert.Equal(t, logEntry.Message, unmarshaledLog.Message)
		assert.Equal(t, logEntry.Kubernetes.Namespace, unmarshaledLog.Kubernetes.Namespace)
		assert.Equal(t, logEntry.Host, unmarshaledLog.Host)
	})

	t.Run("handles invalid JSON gracefully", func(t *testing.T) {
		invalidJSON := "{ invalid json structure"

		var logEntry models.LogEntry
		err := json.Unmarshal([]byte(invalidJSON), &logEntry)
		assert.Error(t, err)
	})

	t.Run("handles empty JSON object", func(t *testing.T) {
		emptyJSON := "{}"

		var logEntry models.LogEntry
		err := json.Unmarshal([]byte(emptyJSON), &logEntry)
		assert.NoError(t, err)

		// Should have zero values
		assert.Empty(t, logEntry.Message)
		assert.Empty(t, logEntry.Level)
		assert.True(t, logEntry.Timestamp.IsZero())
	})
}
