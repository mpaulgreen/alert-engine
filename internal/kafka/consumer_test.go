//go:build unit
// +build unit

package kafka

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/log-monitoring/alert-engine/internal/kafka/mocks"
	"github.com/log-monitoring/alert-engine/pkg/models"
)

func TestNewConsumer(t *testing.T) {
	t.Run("creates consumer successfully", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Second,
		}

		consumer := NewConsumer(config, mockAlertEngine)

		assert.NotNil(t, consumer)
	})

	t.Run("creates consumer with default config", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := DefaultConsumerConfig()

		consumer := NewConsumer(config, mockAlertEngine)

		assert.NotNil(t, consumer)
	})
}

func TestConsumerConfig(t *testing.T) {
	t.Run("validates correct config", func(t *testing.T) {
		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Second,
		}

		// Test that we can create a consumer with this config
		mockAlertEngine := mocks.NewMockAlertEngine()
		consumer := NewConsumer(config, mockAlertEngine)
		assert.NotNil(t, consumer)
	})

	t.Run("handles empty brokers", func(t *testing.T) {
		config := ConsumerConfig{
			Brokers: []string{}, // Empty brokers
			Topic:   "test-topic",
			GroupID: "test-group",
		}

		mockAlertEngine := mocks.NewMockAlertEngine()
		// Constructor validates brokers and will panic with empty brokers
		assert.Panics(t, func() {
			NewConsumer(config, mockAlertEngine)
		})
	})

	t.Run("handles empty topic", func(t *testing.T) {
		config := ConsumerConfig{
			Brokers: []string{"localhost:9092"},
			Topic:   "", // Empty topic
			GroupID: "test-group",
		}

		mockAlertEngine := mocks.NewMockAlertEngine()
		// Constructor validates topic and will panic with empty topic
		assert.Panics(t, func() {
			NewConsumer(config, mockAlertEngine)
		})
	})
}

func TestDefaultConsumerConfig(t *testing.T) {
	t.Run("returns valid default config", func(t *testing.T) {
		config := DefaultConsumerConfig()

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
		config := DefaultConsumerConfig()

		group := NewConsumerGroup(config, mockAlertEngine, 3)
		assert.NotNil(t, group)
	})

	t.Run("gets group statistics", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := DefaultConsumerConfig()

		group := NewConsumerGroup(config, mockAlertEngine, 2)
		stats := group.GetGroupStats()

		assert.Len(t, stats, 2) // Should have stats for 2 consumers
		for _, stat := range stats {
			assert.NotNil(t, stat)
		}
	})

	t.Run("creates group with different consumer counts", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := DefaultConsumerConfig()

		// Test various consumer counts
		for _, count := range []int{1, 3, 5, 10} {
			group := NewConsumerGroup(config, mockAlertEngine, count)
			assert.NotNil(t, group)

			stats := group.GetGroupStats()
			assert.Len(t, stats, count)
		}
	})
}

func TestMessageProcessor(t *testing.T) {
	t.Run("creates message processor", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := DefaultConsumerConfig()
		consumer := NewConsumer(config, mockAlertEngine)

		processor := NewMessageProcessor(consumer, 10, time.Second, mockAlertEngine)
		assert.NotNil(t, processor)
	})

	t.Run("creates processor with different batch sizes", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := DefaultConsumerConfig()
		consumer := NewConsumer(config, mockAlertEngine)

		// Test various batch sizes
		for _, batchSize := range []int{1, 5, 10, 100} {
			processor := NewMessageProcessor(consumer, batchSize, time.Second, mockAlertEngine)
			assert.NotNil(t, processor)
		}
	})

	t.Run("creates processor with different flush intervals", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := DefaultConsumerConfig()
		consumer := NewConsumer(config, mockAlertEngine)

		// Test various flush intervals
		intervals := []time.Duration{
			100 * time.Millisecond,
			time.Second,
			5 * time.Second,
			time.Minute,
		}

		for _, interval := range intervals {
			processor := NewMessageProcessor(consumer, 10, interval, mockAlertEngine)
			assert.NotNil(t, processor)
		}
	})
}

func TestConsumer_BasicOperations(t *testing.T) {
	t.Run("consumer provides statistics", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := DefaultConsumerConfig()
		consumer := NewConsumer(config, mockAlertEngine)

		stats := consumer.GetStats()
		assert.NotNil(t, stats)
		// Default stats should have zero values
		assert.Equal(t, int64(0), stats.Messages)
		assert.Equal(t, int64(0), stats.Bytes)
	})

	t.Run("consumer reports health status", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := DefaultConsumerConfig()
		consumer := NewConsumer(config, mockAlertEngine)

		healthy := consumer.HealthCheck()
		assert.True(t, healthy) // Should be healthy by default
	})

	t.Run("consumer can be closed", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := DefaultConsumerConfig()
		consumer := NewConsumer(config, mockAlertEngine)

		err := consumer.Close()
		assert.NoError(t, err)
	})
}

func TestConsumerEdgeCases(t *testing.T) {
	t.Run("handles very large message config", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()

		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 10485760, // 10MB
			MaxWait:  time.Second,
		}

		consumer := NewConsumer(config, mockAlertEngine)
		assert.NotNil(t, consumer)
	})

	t.Run("handles zero time durations", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()

		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1,
			MaxBytes: 1024,
			MaxWait:  0, // Zero wait time
		}

		// Zero wait time should be acceptable
		consumer := NewConsumer(config, mockAlertEngine)
		assert.NotNil(t, consumer)
	})

	t.Run("handles nil alert engine gracefully", func(t *testing.T) {
		config := DefaultConsumerConfig()

		// Constructor doesn't validate alertEngine, but it will panic when used
		consumer := NewConsumer(config, nil)
		assert.NotNil(t, consumer)

		// The panic would occur when trying to use the alert engine
		// For unit tests, we just verify the constructor doesn't panic
	})
}

func TestMockIntegration(t *testing.T) {
	t.Run("mock alert engine works correctly", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()

		// Test mock functionality
		assert.Equal(t, 0, mockAlertEngine.GetCallCount())
		assert.Nil(t, mockAlertEngine.GetLastCall())

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
		assert.Equal(t, 1, mockAlertEngine.GetCallCount())

		lastLog := mockAlertEngine.GetLastCall()
		assert.NotNil(t, lastLog)
		assert.Equal(t, "Test message", lastLog.Message)
		assert.Equal(t, "test", lastLog.GetNamespace())
	})

	t.Run("mock alert engine can be reset", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()

		// Add some data
		logEntry := models.LogEntry{
			Message:    "Test",
			Kubernetes: models.KubernetesInfo{Namespace: "test"},
		}
		mockAlertEngine.EvaluateLog(logEntry)

		assert.Equal(t, 1, mockAlertEngine.GetCallCount())

		// Reset
		mockAlertEngine.Reset()

		// Verify reset state
		assert.Equal(t, 0, mockAlertEngine.GetCallCount())
		assert.Nil(t, mockAlertEngine.GetLastCall())
	})

	t.Run("mock alert engine handles panic simulation", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		mockAlertEngine.SimulatePanic = true

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

func TestDefaultConsumerConfigFromEnv(t *testing.T) {
	t.Run("uses environment variables when set", func(t *testing.T) {
		// Set environment variables
		os.Setenv("KAFKA_BROKERS", "broker1:9092,broker2:9092")
		os.Setenv("KAFKA_TOPIC", "custom-topic")
		os.Setenv("KAFKA_GROUP_ID", "custom-group")
		defer func() {
			os.Unsetenv("KAFKA_BROKERS")
			os.Unsetenv("KAFKA_TOPIC")
			os.Unsetenv("KAFKA_GROUP_ID")
		}()

		config := DefaultConsumerConfigFromEnv()

		assert.Equal(t, []string{"broker1:9092", "broker2:9092"}, config.Brokers)
		assert.Equal(t, "custom-topic", config.Topic)
		assert.Equal(t, "custom-group", config.GroupID)
	})

	t.Run("uses defaults when environment variables not set", func(t *testing.T) {
		// Clear any existing environment variables
		os.Unsetenv("KAFKA_BROKERS")
		os.Unsetenv("KAFKA_TOPIC")
		os.Unsetenv("KAFKA_GROUP_ID")

		config := DefaultConsumerConfigFromEnv()
		defaultConfig := DefaultConsumerConfig()

		assert.Equal(t, defaultConfig.Brokers, config.Brokers)
		assert.Equal(t, defaultConfig.Topic, config.Topic)
		assert.Equal(t, defaultConfig.GroupID, config.GroupID)
	})
}

func TestConsumer_Start(t *testing.T) {
	t.Run("handles context cancellation gracefully", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Second,
		}

		consumer := NewConsumer(config, mockAlertEngine)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := consumer.Start(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("handles timeout context", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Millisecond * 10, // Very short timeout
		}

		consumer := NewConsumer(config, mockAlertEngine)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()

		err := consumer.Start(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

func TestConsumer_ProcessMessage(t *testing.T) {
	t.Run("handles timeout gracefully during message processing", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Second,
		}

		consumer := NewConsumer(config, mockAlertEngine)

		// Create a test context with timeout to prevent hanging
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()

		// This will attempt to read from Kafka and should handle timeout gracefully
		err := consumer.processMessage(ctx)

		// We expect an error since we're not connected to real Kafka
		assert.Error(t, err)
		// Should be one of these expected error types
		expectedErrors := []string{
			"context deadline exceeded",
			"context canceled",
			"no brokers available",
			"connection refused",
			"fetching message:",
		}

		errorFound := false
		for _, expectedErr := range expectedErrors {
			if err.Error() == expectedErr ||
				err == context.DeadlineExceeded ||
				err == context.Canceled ||
				strings.Contains(err.Error(), expectedErr) {
				errorFound = true
				break
			}
		}
		assert.True(t, errorFound, "Expected one of the known error types, got: %v", err)
	})
}

func TestMessageProcessor_ProcessBatch(t *testing.T) {
	t.Run("creates and configures batch processor", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Second,
		}

		consumer := NewConsumer(config, mockAlertEngine)
		processor := NewMessageProcessor(consumer, 100, 5*time.Second, mockAlertEngine)
		assert.NotNil(t, processor)
	})

	t.Run("handles context cancellation in batch processing", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Millisecond * 10,
		}

		consumer := NewConsumer(config, mockAlertEngine)
		processor := NewMessageProcessor(consumer, 10, time.Millisecond*50, mockAlertEngine)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := processor.ProcessBatch(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestConsumerGroup_Start(t *testing.T) {
	t.Run("handles consumer group startup and context cancellation", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Millisecond * 10,
		}

		group := NewConsumerGroup(config, mockAlertEngine, 2)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()

		// Start the consumer group - should handle timeout gracefully
		err := group.Start(ctx)

		// Start method might return nil or an error depending on implementation
		if err != nil {
			// If error is returned, it should be one of the expected types
			assert.True(t, err == context.DeadlineExceeded ||
				err.Error() == "context deadline exceeded" ||
				err.Error() == "context canceled" ||
				strings.Contains(err.Error(), "context"))
		} else {
			// If no error, that's also acceptable for this test
			assert.Nil(t, err)
		}
	})
}

func TestConsumerGroup_GetGroupStats(t *testing.T) {
	t.Run("returns stats for all consumers in group", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := DefaultConsumerConfig()

		group := NewConsumerGroup(config, mockAlertEngine, 2)

		stats := group.GetGroupStats()
		assert.Equal(t, 2, len(stats))
	})

	t.Run("group manages multiple consumers", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := DefaultConsumerConfig()

		group := NewConsumerGroup(config, mockAlertEngine, 3)

		stats := group.GetGroupStats()
		assert.Equal(t, 3, len(stats))

		// Each consumer should have its own stats
		for _, stat := range stats {
			assert.NotNil(t, stat)
		}
	})
}

func TestMessageProcessor_ReadAndBuffer(t *testing.T) {
	t.Run("handles timeout during message reading", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Millisecond * 10, // Short timeout
		}

		consumer := NewConsumer(config, mockAlertEngine)
		processor := NewMessageProcessor(consumer, 5, time.Second, mockAlertEngine)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()

		// This will attempt to read from Kafka and handle timeout
		err := processor.readAndBuffer(ctx)

		// We expect an error since we're not connected to real Kafka
		assert.Error(t, err)
		// Should be one of these expected error types
		expectedErrors := []string{
			"context deadline exceeded",
			"context canceled",
			"no brokers available",
			"connection refused",
			"fetching message:",
		}

		errorFound := false
		for _, expectedErr := range expectedErrors {
			if err.Error() == expectedErr ||
				err == context.DeadlineExceeded ||
				err == context.Canceled ||
				strings.Contains(err.Error(), expectedErr) {
				errorFound = true
				break
			}
		}
		assert.True(t, errorFound, "Expected one of the known error types, got: %v", err)
	})
}

func TestMessageProcessor_FlushBuffer(t *testing.T) {
	t.Run("flushes buffer when batch size reached", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Second,
		}

		consumer := NewConsumer(config, mockAlertEngine)
		processor := NewMessageProcessor(consumer, 3, time.Minute, mockAlertEngine) // Small batch size

		// Manually add items to buffer to test flush logic
		testLogs := []models.LogEntry{
			{Level: "INFO", Message: "Test 1"},
			{Level: "WARN", Message: "Test 2"},
			{Level: "ERROR", Message: "Test 3"},
		}

		processor.buffer = testLogs
		initialCallCount := mockAlertEngine.GetCallCount()

		processor.flushBuffer()

		// Verify all messages were processed
		assert.Equal(t, 0, len(processor.buffer)) // Buffer should be empty
		assert.Equal(t, initialCallCount+len(testLogs), mockAlertEngine.GetCallCount())
	})

	t.Run("handles empty buffer gracefully", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Second,
		}

		consumer := NewConsumer(config, mockAlertEngine)
		processor := NewMessageProcessor(consumer, 10, time.Second, mockAlertEngine)
		processor.buffer = []models.LogEntry{} // Empty buffer

		initialCallCount := mockAlertEngine.GetCallCount()

		processor.flushBuffer()

		// Should not crash and no new calls should be made
		assert.Equal(t, 0, len(processor.buffer))
		assert.Equal(t, initialCallCount, mockAlertEngine.GetCallCount())
	})
}

func TestProcessorFactory_CreateBatchProcessor(t *testing.T) {
	t.Run("creates batch processor successfully", func(t *testing.T) {
		config := ProcessorConfig{
			ConsumerConfig: ConsumerConfig{
				Brokers:  []string{"localhost:9092"},
				Topic:    "test-topic",
				GroupID:  "test-group",
				MinBytes: 1024,
				MaxBytes: 1048576,
				MaxWait:  time.Second,
			},
			LogProcessingConfig: LogProcessingConfig{
				BatchSize:     100,
				FlushInterval: 5 * time.Second,
				RetryAttempts: 3,
				RetryDelay:    time.Second,
				EnableMetrics: true,
			},
		}

		factory := NewProcessorFactory(config)
		mockAlertEngine := mocks.NewMockAlertEngine()
		mockStateStore := mocks.NewMockStateStore()

		batchProcessor, err := factory.CreateBatchProcessor(
			[]string{"localhost:9092"},
			"test-topic",
			"test-group",
			mockAlertEngine,
			mockStateStore,
		)

		assert.NoError(t, err)
		assert.NotNil(t, batchProcessor)
	})

	t.Run("handles invalid configuration gracefully", func(t *testing.T) {
		config := ProcessorConfig{
			ConsumerConfig: ConsumerConfig{
				Brokers:  []string{"localhost:9092"}, // Valid brokers in config
				Topic:    "test-topic",
				GroupID:  "test-group",
				MinBytes: 1024,
				MaxBytes: 1048576,
				MaxWait:  time.Second,
			},
			LogProcessingConfig: LogProcessingConfig{
				BatchSize:     100,
				FlushInterval: 5 * time.Second,
				RetryAttempts: 3,
				RetryDelay:    time.Second,
				EnableMetrics: true,
			},
		}

		factory := NewProcessorFactory(config)
		mockAlertEngine := mocks.NewMockAlertEngine()
		mockStateStore := mocks.NewMockStateStore()

		// Test with invalid parameters that should be caught gracefully
		assert.NotPanics(t, func() {
			batchProcessor, err := factory.CreateBatchProcessor(
				[]string{"invalid-broker:9999"}, // Invalid broker
				"test-topic",
				"test-group",
				mockAlertEngine,
				mockStateStore,
			)
			// Either should succeed or fail gracefully without panic
			if err != nil {
				assert.Nil(t, batchProcessor)
			} else {
				assert.NotNil(t, batchProcessor)
			}
		})
	})
}

func TestBatchLogProcessor_ReadAndBuffer(t *testing.T) {
	t.Run("handles timeout during batch message reading", func(t *testing.T) {
		processor := createTestProcessor()
		batchProcessor := NewBatchLogProcessor(processor, 10, time.Second)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()

		// This will attempt to read from Kafka and handle timeout
		err := batchProcessor.readAndBuffer(ctx)

		// We expect an error since we're not connected to real Kafka
		assert.Error(t, err)
		expectedErrors := []string{
			"context deadline exceeded",
			"context canceled",
			"no brokers available",
			"connection refused",
			"fetching message:",
		}

		errorFound := false
		for _, expectedErr := range expectedErrors {
			if err.Error() == expectedErr ||
				err == context.DeadlineExceeded ||
				err == context.Canceled ||
				strings.Contains(err.Error(), expectedErr) {
				errorFound = true
				break
			}
		}
		assert.True(t, errorFound, "Expected one of the known error types, got: %v", err)
	})

	t.Run("handles context cancellation during buffer read", func(t *testing.T) {
		processor := createTestProcessor()
		batchProcessor := NewBatchLogProcessor(processor, 10, time.Millisecond*10)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := batchProcessor.readAndBuffer(ctx)

		assert.Error(t, err)
		// Allow wrapped context canceled errors
		assert.True(t, err == context.Canceled || strings.Contains(err.Error(), "context canceled"),
			"Expected context canceled error, got: %v", err)
	})
}

func TestConsumer_HealthCheck_Enhanced(t *testing.T) {
	t.Run("reports healthy status for new consumer", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := DefaultConsumerConfig()

		consumer := NewConsumer(config, mockAlertEngine)

		healthy := consumer.HealthCheck()
		assert.True(t, healthy)
	})

	t.Run("handles health check correctly", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := DefaultConsumerConfig()

		consumer := NewConsumer(config, mockAlertEngine)

		healthy := consumer.HealthCheck()
		assert.True(t, healthy) // Should be healthy for new consumer
	})
}

// Additional edge case tests
func TestConsumerEdgeCases_Enhanced(t *testing.T) {
	t.Run("handles very large message config", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024 * 1024,       // 1MB
			MaxBytes: 100 * 1024 * 1024, // 100MB
			MaxWait:  time.Minute,
		}

		consumer := NewConsumer(config, mockAlertEngine)
		assert.NotNil(t, consumer)

		stats := consumer.GetStats()
		assert.NotNil(t, stats)
	})

	t.Run("handles zero time durations", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  0, // Zero duration
		}

		consumer := NewConsumer(config, mockAlertEngine)
		assert.NotNil(t, consumer)
	})

	t.Run("handles nil alert engine gracefully", func(t *testing.T) {
		config := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Second,
		}

		// This should not panic even with nil alert engine
		assert.NotPanics(t, func() {
			consumer := NewConsumer(config, nil)
			assert.NotNil(t, consumer)
		})
	})
}
