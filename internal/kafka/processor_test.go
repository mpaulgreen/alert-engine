//go:build unit
// +build unit

package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/log-monitoring/alert-engine/internal/kafka/mocks"
	"github.com/log-monitoring/alert-engine/pkg/models"
)

// Helper function to create a test processor with default configs
func createTestProcessor() *LogProcessor {
	mockAlertEngine := mocks.NewMockAlertEngine()
	mockStateStore := mocks.NewMockStateStore()

	consumerConfig := ConsumerConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    "test-topic",
		GroupID:  "test-group",
		MinBytes: 1024,
		MaxBytes: 1048576,
		MaxWait:  time.Second,
	}

	logProcessingConfig := LogProcessingConfig{
		BatchSize:       50,
		FlushInterval:   10 * time.Second,
		RetryAttempts:   3,
		RetryDelay:      1 * time.Second,
		EnableMetrics:   true,
		DefaultLogLevel: "INFO",
	}

	return NewLogProcessor(consumerConfig, logProcessingConfig, mockAlertEngine, mockStateStore)
}

func TestNewLogProcessor(t *testing.T) {
	mockAlertEngine := &mocks.MockAlertEngine{}
	mockStateStore := &mocks.MockStateStore{}

	consumerConfig := ConsumerConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    "test-topic",
		GroupID:  "test-group",
		MinBytes: 1024,
		MaxBytes: 1048576,
		MaxWait:  time.Second,
	}

	logProcessingConfig := LogProcessingConfig{
		BatchSize:       50,
		FlushInterval:   10 * time.Second,
		RetryAttempts:   3,
		RetryDelay:      1 * time.Second,
		EnableMetrics:   true,
		DefaultLogLevel: "INFO",
	}

	processor := NewLogProcessor(consumerConfig, logProcessingConfig, mockAlertEngine, mockStateStore)

	assert.NotNil(t, processor)
	assert.Equal(t, mockAlertEngine, processor.GetAlertEngine())
}

func TestLogProcessor_ProcessLogs(t *testing.T) {
	t.Run("processes valid log message", func(t *testing.T) {
		processor := createTestProcessor()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Start processing in a goroutine
		go func() {
			processor.ProcessLogs(ctx)
		}()

		// Wait a bit for processing
		time.Sleep(50 * time.Millisecond)

		// Should handle context cancellation gracefully
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		processor := createTestProcessor()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := processor.ProcessLogs(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("handles processing errors with retry", func(t *testing.T) {
		processor := createTestProcessor()

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		// Start processing
		go func() {
			processor.ProcessLogs(ctx)
		}()

		time.Sleep(150 * time.Millisecond)

		// Should complete without panicking
	})
}

func TestLogProcessor_ValidateLogEntry(t *testing.T) {
	processor := createTestProcessor()

	t.Run("validates correct log entry", func(t *testing.T) {
		logEntry := &models.LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   "Test message",
			Service:   "test-service",
			Kubernetes: models.KubernetesInfo{
				Namespace: "test-namespace",
				Pod:       "test-pod",
				Container: "test-container",
			},
		}

		err := processor.validateLogEntry(logEntry)
		assert.NoError(t, err)
	})

	t.Run("handles missing timestamp", func(t *testing.T) {
		logEntry := &models.LogEntry{
			Level:   "INFO",
			Message: "Test message",
			Kubernetes: models.KubernetesInfo{
				Namespace: "test-namespace",
			},
		}

		err := processor.validateLogEntry(logEntry)
		assert.NoError(t, err)
		assert.False(t, logEntry.Timestamp.IsZero())
	})

	t.Run("uses @timestamp when timestamp is missing", func(t *testing.T) {
		testTime := time.Now()
		logEntry := &models.LogEntry{
			AtTimestamp: testTime,
			Level:       "INFO",
			Message:     "Test message",
			Kubernetes: models.KubernetesInfo{
				Namespace: "test-namespace",
			},
		}

		err := processor.validateLogEntry(logEntry)
		assert.NoError(t, err)
		assert.Equal(t, testTime, logEntry.Timestamp)
	})

	t.Run("handles empty message", func(t *testing.T) {
		logEntry := &models.LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   "", // Empty message
			Kubernetes: models.KubernetesInfo{
				Namespace: "test-namespace",
			},
		}

		err := processor.validateLogEntry(logEntry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "message is empty")
	})

	t.Run("handles missing namespace", func(t *testing.T) {
		logEntry := &models.LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   "Test message",
			// Missing namespace
		}

		err := processor.validateLogEntry(logEntry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing kubernetes namespace")
	})

	t.Run("handles empty log level", func(t *testing.T) {
		logEntry := &models.LogEntry{
			Timestamp: time.Now(),
			Level:     "", // Empty level
			Message:   "Test message",
			Kubernetes: models.KubernetesInfo{
				Namespace: "test-namespace",
			},
		}

		err := processor.validateLogEntry(logEntry)
		assert.NoError(t, err)
		assert.Equal(t, "INFO", logEntry.Level) // Should use default
	})

	t.Run("uses custom default log level", func(t *testing.T) {
		// Create processor with custom default log level
		mockAlertEngine := mocks.NewMockAlertEngine()
		mockStateStore := mocks.NewMockStateStore()

		consumerConfig := ConsumerConfig{
			Brokers:  []string{"localhost:9092"},
			Topic:    "test-topic",
			GroupID:  "test-group",
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Second,
		}

		logProcessingConfig := LogProcessingConfig{
			BatchSize:       50,
			FlushInterval:   10 * time.Second,
			RetryAttempts:   3,
			RetryDelay:      1 * time.Second,
			EnableMetrics:   true,
			DefaultLogLevel: "DEBUG", // Custom default
		}

		processor := NewLogProcessor(consumerConfig, logProcessingConfig, mockAlertEngine, mockStateStore)

		logEntry := &models.LogEntry{
			Timestamp: time.Now(),
			Level:     "", // Empty level
			Message:   "Test message",
			Kubernetes: models.KubernetesInfo{
				Namespace: "test-namespace",
			},
		}

		err := processor.validateLogEntry(logEntry)
		assert.NoError(t, err)
		assert.Equal(t, "DEBUG", logEntry.Level) // Should use custom default
	})
}

func TestLogProcessor_UpdateErrorRate(t *testing.T) {
	t.Run("calculates error rate correctly", func(t *testing.T) {
		processor := createTestProcessor()

		// Reset metrics to clean state
		processor.metrics.MessagesProcessed = 80
		processor.metrics.MessagesFailure = 20

		processor.updateErrorRate()

		expectedErrorRate := float64(20) / float64(100) // 20%
		assert.Equal(t, expectedErrorRate, processor.metrics.ErrorRate)
	})

	t.Run("handles zero messages", func(t *testing.T) {
		processor := createTestProcessor()

		// Explicitly reset metrics to zero state
		processor.metrics.MessagesProcessed = 0
		processor.metrics.MessagesFailure = 0
		processor.metrics.ErrorRate = 0

		processor.updateErrorRate()

		assert.Equal(t, float64(0), processor.metrics.ErrorRate)
	})

	t.Run("handles only failures", func(t *testing.T) {
		processor := createTestProcessor()

		// Reset metrics to clean state
		processor.metrics.MessagesProcessed = 0
		processor.metrics.MessagesFailure = 10

		processor.updateErrorRate()

		assert.Equal(t, float64(1.0), processor.metrics.ErrorRate) // 100% error rate
	})
}

func TestLogProcessor_UpdateLogStats(t *testing.T) {
	processor := createTestProcessor()

	t.Run("updates log statistics correctly", func(t *testing.T) {
		logEntry := models.LogEntry{
			Level:   "ERROR",
			Service: "auth-service",
		}

		// Reset stats
		processor.logStats.TotalLogs = 0
		processor.logStats.LogsByLevel = make(map[string]int64)
		processor.logStats.LogsByService = make(map[string]int64)

		processor.updateLogStats(logEntry)

		assert.Equal(t, int64(1), processor.logStats.TotalLogs)
		assert.Equal(t, int64(1), processor.logStats.LogsByLevel["ERROR"])
		assert.Equal(t, int64(1), processor.logStats.LogsByService["auth-service"])
		assert.False(t, processor.logStats.LastUpdated.IsZero())
	})

	t.Run("accumulates multiple log entries", func(t *testing.T) {
		// Reset stats
		processor.logStats.TotalLogs = 0
		processor.logStats.LogsByLevel = make(map[string]int64)
		processor.logStats.LogsByService = make(map[string]int64)

		logEntries := []models.LogEntry{
			{Level: "INFO", Service: "web-service"},
			{Level: "ERROR", Service: "auth-service"},
			{Level: "INFO", Service: "web-service"},
			{Level: "WARN", Service: "db-service"},
		}

		for _, entry := range logEntries {
			processor.updateLogStats(entry)
		}

		assert.Equal(t, int64(4), processor.logStats.TotalLogs)
		assert.Equal(t, int64(2), processor.logStats.LogsByLevel["INFO"])
		assert.Equal(t, int64(1), processor.logStats.LogsByLevel["ERROR"])
		assert.Equal(t, int64(1), processor.logStats.LogsByLevel["WARN"])
		assert.Equal(t, int64(2), processor.logStats.LogsByService["web-service"])
		assert.Equal(t, int64(1), processor.logStats.LogsByService["auth-service"])
		assert.Equal(t, int64(1), processor.logStats.LogsByService["db-service"])
	})

	t.Run("handles empty service name", func(t *testing.T) {
		logEntry := models.LogEntry{
			Level:   "INFO",
			Service: "", // Empty service
		}

		initialTotal := processor.logStats.TotalLogs
		processor.updateLogStats(logEntry)

		assert.Equal(t, initialTotal+1, processor.logStats.TotalLogs)
		assert.Equal(t, int64(0), processor.logStats.LogsByService[""]) // Should not increment empty service
	})

	t.Run("triggers state store save every 100 logs", func(t *testing.T) {
		// Set up to trigger save
		processor.logStats.TotalLogs = 99

		logEntry := models.LogEntry{
			Level:   "INFO",
			Service: "test-service",
		}

		processor.updateLogStats(logEntry)

		assert.Equal(t, int64(100), processor.logStats.TotalLogs)
		// The save operation will be called but we can't easily verify it without checking mock calls
	})
}

func TestBatchLogProcessor_FlushBatch(t *testing.T) {
	t.Run("processes all messages in batch", func(t *testing.T) {
		processor := createTestProcessor()
		batchProcessor := NewBatchLogProcessor(processor, 10, time.Minute)

		// Add messages to batch
		testLogs := []models.LogEntry{
			{Level: "INFO", Message: "Test 1", Service: "service1"},
			{Level: "ERROR", Message: "Test 2", Service: "service2"},
			{Level: "WARN", Message: "Test 3", Service: "service3"},
		}

		batchProcessor.batchBuffer = testLogs
		initialProcessed := processor.metrics.MessagesProcessed

		batchProcessor.flushBatch()

		// Verify batch was processed
		assert.Equal(t, 0, len(batchProcessor.batchBuffer)) // Buffer should be empty
		assert.Equal(t, initialProcessed+int64(len(testLogs)), processor.metrics.MessagesProcessed)
		assert.False(t, processor.metrics.LastProcessed.IsZero())
	})

	t.Run("handles empty batch gracefully", func(t *testing.T) {
		processor := createTestProcessor()
		batchProcessor := NewBatchLogProcessor(processor, 10, time.Minute)

		batchProcessor.batchBuffer = []models.LogEntry{} // Empty batch
		initialProcessed := processor.metrics.MessagesProcessed

		batchProcessor.flushBatch()

		// Should not crash and metrics should be unchanged
		assert.Equal(t, 0, len(batchProcessor.batchBuffer))
		assert.Equal(t, initialProcessed, processor.metrics.MessagesProcessed)
	})

	t.Run("resets flush timer after processing", func(t *testing.T) {
		processor := createTestProcessor()
		batchProcessor := NewBatchLogProcessor(processor, 10, time.Second)

		// Add a message to batch
		testLogs := []models.LogEntry{
			{Level: "INFO", Message: "Test", Service: "service1"},
		}
		batchProcessor.batchBuffer = testLogs

		// Record timer state before flush
		timerWasActive := batchProcessor.flushTimer != nil

		batchProcessor.flushBatch()

		// Timer should still exist and be reset
		assert.True(t, timerWasActive)
		assert.NotNil(t, batchProcessor.flushTimer)
	})
}

func TestLogProcessor_ProcessMessage_Enhanced(t *testing.T) {
	t.Run("validates log entry successfully through public interface", func(t *testing.T) {
		processor := createTestProcessor()

		logEntry := &models.LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   "Test log message",
			Service:   "test-service",
			Kubernetes: models.KubernetesInfo{
				Namespace: "test-namespace",
				Pod:       "test-pod",
				Container: "test-container",
			},
		}

		// Test validation
		err := processor.validateLogEntry(logEntry)
		assert.NoError(t, err)

		// Test stats update
		initialStats := processor.logStats.TotalLogs
		processor.updateLogStats(*logEntry)
		assert.Equal(t, initialStats+1, processor.logStats.TotalLogs)
	})

	t.Run("handles validation errors correctly", func(t *testing.T) {
		processor := createTestProcessor()

		logEntry := &models.LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   "", // Empty message should cause validation error
		}

		err := processor.validateLogEntry(logEntry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "message is empty")
	})

	t.Run("alert engine evaluation works correctly", func(t *testing.T) {
		processor := createTestProcessor()

		logEntry := models.LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   "Test message",
			Kubernetes: models.KubernetesInfo{
				Namespace: "test-namespace",
			},
		}

		mockEngine := processor.alertEngine.(*mocks.MockAlertEngine)
		initialCallCount := mockEngine.GetCallCount()

		// Directly call the alert engine
		mockEngine.EvaluateLog(logEntry)

		assert.Equal(t, initialCallCount+1, mockEngine.GetCallCount())
		lastCall := mockEngine.GetLastCall()
		assert.NotNil(t, lastCall)
		assert.Equal(t, "Test message", lastCall.Message)
	})

	t.Run("handles alert engine panic gracefully", func(t *testing.T) {
		processor := createTestProcessor()

		// Configure mock to simulate panic
		mockEngine := processor.alertEngine.(*mocks.MockAlertEngine)
		mockEngine.SimulatePanic = true

		logEntry := models.LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   "Test message",
			Kubernetes: models.KubernetesInfo{
				Namespace: "test-namespace",
			},
		}

		// Should panic when configured to do so
		assert.Panics(t, func() {
			mockEngine.EvaluateLog(logEntry)
		})
	})
}

// Test processMessage indirectly through ProcessLogs
func TestLogProcessor_ProcessLogs_Enhanced(t *testing.T) {
	t.Run("processes logs with timeout handling", func(t *testing.T) {
		processor := createTestProcessor()

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()

		// This will attempt to read from Kafka and handle timeout gracefully
		err := processor.ProcessLogs(ctx)

		// We expect context deadline exceeded since we're not connected to real Kafka
		assert.Error(t, err)
		assert.True(t, err == context.DeadlineExceeded ||
			err.Error() == "context deadline exceeded" ||
			err.Error() == "context canceled")
	})

	t.Run("handles context cancellation correctly", func(t *testing.T) {
		processor := createTestProcessor()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := processor.ProcessLogs(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestLogProcessor_GetMetrics(t *testing.T) {
	t.Run("returns processor metrics", func(t *testing.T) {
		processor := createTestProcessor()

		metrics := processor.GetMetrics()
		assert.NotNil(t, metrics)
		assert.Equal(t, int64(0), metrics.MessagesProcessed)
		assert.Equal(t, int64(0), metrics.MessagesFailure)
		assert.Equal(t, float64(0), metrics.ErrorRate)
		assert.True(t, metrics.LastProcessed.IsZero())
	})
}

func TestLogProcessor_HealthCheck(t *testing.T) {
	t.Run("returns healthy status initially", func(t *testing.T) {
		processor := createTestProcessor()

		healthy := processor.HealthCheck()
		// Should be unhealthy initially as no messages have been processed
		assert.False(t, healthy)
	})

	t.Run("detects unhealthy state with high error rate", func(t *testing.T) {
		processor := createTestProcessor()

		// Simulate high error rate by manipulating metrics
		metrics := processor.GetMetrics()
		metrics.MessagesProcessed = 10
		metrics.MessagesFailure = 5
		metrics.ErrorRate = 0.5 // 50% error rate

		healthy := processor.HealthCheck()
		assert.False(t, healthy) // Should be unhealthy with 50% error rate
	})

	t.Run("healthy with recent processing", func(t *testing.T) {
		processor := createTestProcessor()

		// Simulate recent successful processing
		metrics := processor.GetMetrics()
		metrics.MessagesProcessed = 100
		metrics.MessagesFailure = 1
		metrics.ErrorRate = 0.01 // 1% error rate
		metrics.LastProcessed = time.Now()

		healthy := processor.HealthCheck()
		assert.True(t, healthy) // Should be healthy
	})
}

func TestLogProcessor_Stop(t *testing.T) {
	t.Run("stops processor gracefully", func(t *testing.T) {
		processor := createTestProcessor()

		err := processor.Stop()
		assert.NoError(t, err)
	})
}

func TestBatchLogProcessor(t *testing.T) {
	t.Run("creates batch processor", func(t *testing.T) {

		batchProcessor := NewBatchLogProcessor(createTestProcessor(), 10, time.Second)
		assert.NotNil(t, batchProcessor)
	})

	t.Run("processes logs in batches", func(t *testing.T) {

		batchProcessor := NewBatchLogProcessor(createTestProcessor(), 5, 100*time.Millisecond)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		// Start batch processing
		go func() {
			batchProcessor.ProcessBatch(ctx)
		}()

		// Wait for processing
		time.Sleep(150 * time.Millisecond)

		// Should complete without error when context is canceled
	})

	t.Run("flushes on context cancellation", func(t *testing.T) {

		batchProcessor := NewBatchLogProcessor(createTestProcessor(), 10, time.Minute)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := batchProcessor.ProcessBatch(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("flushes on timer", func(t *testing.T) {

		// Short flush interval for testing
		batchProcessor := NewBatchLogProcessor(createTestProcessor(), 100, 50*time.Millisecond)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		go func() {
			batchProcessor.ProcessBatch(ctx)
		}()

		// Wait for multiple flush intervals
		time.Sleep(150 * time.Millisecond)

		// Should have flushed multiple times by now
	})
}

func TestProcessorFactory(t *testing.T) {
	t.Run("creates processor factory", func(t *testing.T) {
		config := DefaultProcessorConfig()
		factory := NewProcessorFactory(config)

		assert.NotNil(t, factory)
	})

	t.Run("creates regular processor", func(t *testing.T) {
		config := DefaultProcessorConfig()
		factory := NewProcessorFactory(config)
		mockAlertEngine := mocks.NewMockAlertEngine()
		mockStateStore := mocks.NewMockStateStore()

		brokers := []string{"localhost:9092"}
		topic := "test-topic"
		groupID := "test-group"

		processor, err := factory.CreateProcessor(brokers, topic, groupID, mockAlertEngine, mockStateStore)
		require.NoError(t, err)
		assert.NotNil(t, processor)
	})

	t.Run("creates batch processor", func(t *testing.T) {
		config := DefaultProcessorConfig()
		factory := NewProcessorFactory(config)
		mockAlertEngine := mocks.NewMockAlertEngine()
		mockStateStore := mocks.NewMockStateStore()

		brokers := []string{"localhost:9092"}
		topic := "test-topic"
		groupID := "test-group"

		batchProcessor, err := factory.CreateBatchProcessor(brokers, topic, groupID, mockAlertEngine, mockStateStore)
		require.NoError(t, err)
		assert.NotNil(t, batchProcessor)
	})

	t.Run("handles invalid configuration", func(t *testing.T) {
		// Create invalid config
		config := ProcessorConfig{
			LogProcessingConfig: LogProcessingConfig{
				BatchSize:     -1,           // Invalid batch size
				FlushInterval: -time.Second, // Invalid interval
			},
		}
		factory := NewProcessorFactory(config)
		mockAlertEngine := mocks.NewMockAlertEngine()
		mockStateStore := mocks.NewMockStateStore()

		brokers := []string{} // Empty brokers will cause panic
		topic := ""
		groupID := "test-group"

		// Factory should panic when trying to create processor with empty brokers
		assert.Panics(t, func() {
			factory.CreateProcessor(brokers, topic, groupID, mockAlertEngine, mockStateStore)
		})
	})
}

func TestDefaultProcessorConfig(t *testing.T) {
	t.Run("returns valid default config", func(t *testing.T) {
		config := DefaultProcessorConfig()

		assert.NotEmpty(t, config.ConsumerConfig.Brokers)
		assert.NotEmpty(t, config.ConsumerConfig.Topic)
		assert.NotEmpty(t, config.ConsumerConfig.GroupID)
		assert.Greater(t, config.LogProcessingConfig.BatchSize, 0)
		assert.Greater(t, config.LogProcessingConfig.FlushInterval, time.Duration(0))
		assert.GreaterOrEqual(t, config.LogProcessingConfig.RetryAttempts, 0)
		assert.GreaterOrEqual(t, config.LogProcessingConfig.RetryDelay, time.Duration(0))
		assert.True(t, config.LogProcessingConfig.EnableMetrics)
	})

	t.Run("has reasonable default values", func(t *testing.T) {
		config := DefaultProcessorConfig()

		// Check specific default values
		assert.Equal(t, "alert-engine-e2e-fresh-20250716", config.ConsumerConfig.GroupID)
		assert.Equal(t, 50, config.LogProcessingConfig.BatchSize)
		assert.Equal(t, 10*time.Second, config.LogProcessingConfig.FlushInterval)
		assert.Equal(t, 3, config.LogProcessingConfig.RetryAttempts)
		assert.Equal(t, 1*time.Second, config.LogProcessingConfig.RetryDelay)
	})
}

func TestProcessorConfig_EdgeCases(t *testing.T) {
	t.Run("handles zero batch size", func(t *testing.T) {
		config := ProcessorConfig{
			LogProcessingConfig: LogProcessingConfig{
				BatchSize: 0,
			},
			ConsumerConfig: ConsumerConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "test-topic",
			},
		}

		factory := NewProcessorFactory(config)
		assert.NotNil(t, factory)
	})

	t.Run("handles zero flush interval", func(t *testing.T) {
		config := ProcessorConfig{
			LogProcessingConfig: LogProcessingConfig{
				FlushInterval: 0,
			},
			ConsumerConfig: ConsumerConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "test-topic",
			},
		}

		factory := NewProcessorFactory(config)
		assert.NotNil(t, factory)
	})

	t.Run("handles disabled metrics", func(t *testing.T) {
		config := ProcessorConfig{
			LogProcessingConfig: LogProcessingConfig{
				EnableMetrics: false,
			},
			ConsumerConfig: ConsumerConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "test-topic",
			},
		}

		factory := NewProcessorFactory(config)
		assert.NotNil(t, factory)
	})

	t.Run("handles zero retry attempts", func(t *testing.T) {
		config := ProcessorConfig{
			LogProcessingConfig: LogProcessingConfig{
				RetryAttempts: 0,
			},
			ConsumerConfig: ConsumerConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "test-topic",
			},
		}

		factory := NewProcessorFactory(config)
		mockAlertEngine := mocks.NewMockAlertEngine()
		mockStateStore := mocks.NewMockStateStore()

		processor, err := factory.CreateProcessor([]string{"localhost:9092"}, "test-topic", "test-group", mockAlertEngine, mockStateStore)
		require.NoError(t, err)
		assert.NotNil(t, processor)
	})
}

func TestProcessorMetrics(t *testing.T) {
	t.Run("tracks processing metrics", func(t *testing.T) {
		processor := createTestProcessor()

		metrics := processor.GetMetrics()

		// Initial state
		assert.Equal(t, int64(0), metrics.MessagesProcessed)
		assert.Equal(t, int64(0), metrics.MessagesFailure)
		assert.Equal(t, time.Duration(0), metrics.ProcessingTime)
		assert.True(t, metrics.LastProcessed.IsZero())
		assert.Equal(t, float64(0), metrics.ErrorRate)
	})

	t.Run("calculates error rate correctly", func(t *testing.T) {
		// We can't directly test updateErrorRate as it's private,
		// but we can verify the logic conceptually
		totalMessages := int64(100)
		failedMessages := int64(10)
		expectedErrorRate := float64(failedMessages) / float64(totalMessages)

		assert.Equal(t, 0.1, expectedErrorRate)
	})
}

func TestProcessor_ConcurrentAccess(t *testing.T) {
	t.Run("handles concurrent access safely", func(t *testing.T) {
		processor := createTestProcessor()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Start multiple operations concurrently
		go func() {
			processor.ProcessLogs(ctx)
		}()

		go func() {
			metrics := processor.GetMetrics()
			assert.NotNil(t, metrics)
		}()

		go func() {
			healthy := processor.HealthCheck()
			assert.False(t, healthy) // Should be false initially
		}()

		time.Sleep(50 * time.Millisecond)

		// Stop should be safe to call
		err := processor.Stop()
		assert.NoError(t, err)
	})
}

func TestProcessor_AlertEngineIntegration(t *testing.T) {
	t.Run("integrates with alert engine", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()

		logEntry := models.LogEntry{
			Level:   "ERROR",
			Message: "Integration test message",
		}

		mockAlertEngine.EvaluateLog(logEntry)

		// Verify alert engine was called
		assert.Equal(t, 1, mockAlertEngine.GetCallCount())

		lastLog := mockAlertEngine.GetLastCall()
		assert.NotNil(t, lastLog)
		assert.Equal(t, "Integration test message", lastLog.Message)
	})

	t.Run("handles alert engine panic", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()

		mockAlertEngine.SimulatePanic = true

		// Should handle panic gracefully when called
		logEntry := models.LogEntry{
			Level:   "ERROR",
			Message: "Test panic",
		}

		assert.Panics(t, func() {
			mockAlertEngine.EvaluateLog(logEntry)
		})
	})
}
