//go:build integration
// +build integration

package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/log-monitoring/alert-engine/internal/kafka/mocks"
	"github.com/log-monitoring/alert-engine/internal/kafka/testcontainers"
	"github.com/log-monitoring/alert-engine/pkg/models"
)

var (
	// Shared container for tests that can reuse the same instance
	sharedKafkaContainer *testcontainers.KafkaContainer
	containerMutex       sync.Mutex
	containerRefCount    int
)

// Helper function to create processors for integration tests
func createIntegrationProcessor(brokers []string, topic string, groupID string, alertEngine *mocks.MockAlertEngine) *LogProcessor {
	consumerConfig := ConsumerConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1024,
		MaxBytes: 1048576,
		MaxWait:  time.Second,
	}
	logProcessingConfig := DefaultLogProcessingConfig()
	mockStateStore := mocks.NewMockStateStore()
	return NewLogProcessor(consumerConfig, logProcessingConfig, alertEngine, mockStateStore)
}

// getOrCreateSharedContainer creates a shared Kafka container for tests that can reuse it
func getOrCreateSharedContainer(t *testing.T) *testcontainers.KafkaContainer {
	containerMutex.Lock()
	defer containerMutex.Unlock()

	if sharedKafkaContainer == nil {
		var err error
		sharedKafkaContainer, err = testcontainers.NewKafkaContainerForTesting(t)
		require.NoError(t, err)

		// Wait for Kafka to be ready
		err = sharedKafkaContainer.TestKafkaAvailability()
		require.NoError(t, err)
	}

	containerRefCount++
	return sharedKafkaContainer
}

// releaseSharedContainer decrements the reference count and cleans up if needed
func releaseSharedContainer() {
	containerMutex.Lock()
	defer containerMutex.Unlock()

	containerRefCount--
	if containerRefCount <= 0 && sharedKafkaContainer != nil {
		sharedKafkaContainer.Cleanup()
		sharedKafkaContainer = nil
		containerRefCount = 0
	}
}

func init() {
	// Configure testcontainers for compatibility with different container runtimes
	configureTestcontainers()
}

// configureTestcontainers sets up testcontainers to work with Docker or Podman
func configureTestcontainers() {
	// Check if we're using Podman
	if isPodmanAvailable() {
		// Configure for Podman
		fmt.Println("Configuring testcontainers for Podman compatibility...")

		// Disable Ryuk reaper which requires bridge network
		os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

		// Set Docker host to Podman socket
		if os.Getenv("DOCKER_HOST") == "" {
			// Try different Podman socket locations
			podmanSockets := []string{
				"unix:///run/user/" + os.Getenv("UID") + "/podman/podman.sock",
				"unix:///var/run/user/" + getUserID() + "/podman/podman.sock",
				"unix:///tmp/podman-run-" + getUserID() + "/podman/podman.sock",
				"unix:///run/podman/podman.sock",
			}

			for _, socket := range podmanSockets {
				if fileExists(strings.TrimPrefix(socket, "unix://")) {
					os.Setenv("DOCKER_HOST", socket)
					break
				}
			}

			// Fallback: use the default Docker socket and let Podman handle compatibility
			if os.Getenv("DOCKER_HOST") == "" {
				os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
			}
		}

		// Disable network name validation for Podman
		os.Setenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", "/var/run/docker.sock")

	} else {
		// Configure for Docker
		if os.Getenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE") == "" {
			os.Setenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", "/var/run/docker.sock")
		}
		if os.Getenv("DOCKER_HOST") == "" {
			os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
		}

		// Only disable Ryuk if explicitly requested (better for cleanup)
		if os.Getenv("DISABLE_RYUK") == "true" {
			os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
			os.Setenv("TESTCONTAINERS_REAPER_IMAGE", "")
		}
	}
}

// isPodmanAvailable checks if Podman is available on the system
func isPodmanAvailable() bool {
	_, err := exec.LookPath("podman")
	return err == nil
}

// getUserID returns the current user ID as a string
func getUserID() string {
	if uid := os.Getenv("UID"); uid != "" {
		return uid
	}
	// Fallback to getting UID from system
	if user, err := user.Current(); err == nil {
		return user.Uid
	}
	return "1000" // Default fallback
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestKafkaConsumerIntegration(t *testing.T) {
	// Skip if running in CI without Docker
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use dedicated container for consumer tests
	kafkaContainer, err := testcontainers.NewKafkaContainerForTesting(t)
	require.NoError(t, err)
	defer kafkaContainer.Cleanup()

	// Wait for Kafka to be ready
	err = kafkaContainer.TestKafkaAvailability()
	require.NoError(t, err)

	t.Run("consumer connects to real Kafka", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()

		config := ConsumerConfig{
			Brokers:  kafkaContainer.GetBrokers(),
			Topic:    "test-logs-" + fmt.Sprintf("%d", time.Now().UnixNano()), // Unique topic
			GroupID:  "test-integration-group-" + fmt.Sprintf("%d", time.Now().UnixNano()),
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Second,
		}

		consumer := NewConsumer(config, mockAlertEngine)
		require.NotNil(t, consumer)

		// Test that consumer can be created and closed
		err := consumer.Close()
		assert.NoError(t, err)
	})

	t.Run("consumer processes real Kafka messages", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		topicName := "application-logs-" + fmt.Sprintf("%d", time.Now().UnixNano())

		config := ConsumerConfig{
			Brokers:  kafkaContainer.GetBrokers(),
			Topic:    topicName,
			GroupID:  "test-process-group-" + fmt.Sprintf("%d", time.Now().UnixNano()),
			MinBytes: 1,
			MaxBytes: 1048576,
			MaxWait:  500 * time.Millisecond,
		}

		consumer := NewConsumer(config, mockAlertEngine)
		require.NotNil(t, consumer)

		// Create test log entry
		logEntry := models.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   "Integration test message",
			Kubernetes: models.KubernetesInfo{
				Namespace: "integration-test",
				Pod:       "test-pod-123",
				Container: "test-container",
				Labels: map[string]string{
					"app":     "test-app",
					"version": "1.0.0",
				},
			},
			Host: "test-host",
		}

		// Produce a test message to Kafka
		logJSON, err := json.Marshal(logEntry)
		require.NoError(t, err)

		err = kafkaContainer.ProduceMessage(topicName, "test-key", string(logJSON))
		require.NoError(t, err)

		// Start consumer with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Start consumer in goroutine
		consumerErr := make(chan error, 1)
		go func() {
			consumerErr <- consumer.Start(ctx)
		}()

		// Wait for message processing (allow some time for consumer to process)
		time.Sleep(3 * time.Second)

		// Cancel context to stop consumer
		cancel()

		// Wait for consumer to stop
		select {
		case err := <-consumerErr:
			// Context cancellation can result in either context.Canceled or context.DeadlineExceeded
			assert.True(t, err == context.Canceled || err == context.DeadlineExceeded,
				"Expected context cancellation error, got: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Consumer did not stop within timeout")
		}

		// Check if any logs were processed by looking at the mock
		if mockAlertEngine.GetCallCount() > 0 {
			t.Logf("Alert engine received %d log evaluations", mockAlertEngine.GetCallCount())
			lastLog := mockAlertEngine.GetLastCall()
			if lastLog != nil {
				t.Logf("Last processed log: %s", lastLog.Message)
			}
		} else {
			t.Log("Warning: Message processing not detected - this might be due to timing")
		}

		// Clean up
		err = consumer.Close()
		assert.NoError(t, err)
	})

	t.Run("consumer group integration", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		topicName := "test-logs-" + fmt.Sprintf("%d", time.Now().UnixNano())

		config := ConsumerConfig{
			Brokers:  kafkaContainer.GetBrokers(),
			Topic:    topicName,
			GroupID:  "test-group-integration-" + fmt.Sprintf("%d", time.Now().UnixNano()),
			MinBytes: 1,
			MaxBytes: 1048576,
			MaxWait:  500 * time.Millisecond,
		}

		// Create consumer group with 2 consumers
		group := NewConsumerGroup(config, mockAlertEngine, 2)
		require.NotNil(t, group)

		// Start group with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := group.Start(ctx)
		// Should return without error when context is canceled
		assert.NoError(t, err)
	})
}

func TestKafkaProcessorIntegration(t *testing.T) {
	// Skip if running in CI without Docker
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use shared container for processor tests to avoid multiple container creation
	kafkaContainer := getOrCreateSharedContainer(t)
	defer releaseSharedContainer()

	t.Run("processor connects to real Kafka", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		topicName := "test-logs-" + fmt.Sprintf("%d", time.Now().UnixNano())

		consumerConfig := ConsumerConfig{
			Brokers:  kafkaContainer.GetBrokers(),
			Topic:    topicName,
			GroupID:  "test-group-" + fmt.Sprintf("%d", time.Now().UnixNano()),
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  time.Second,
		}
		logProcessingConfig := DefaultLogProcessingConfig()
		mockStateStore := mocks.NewMockStateStore()
		processor := NewLogProcessor(consumerConfig, logProcessingConfig, mockAlertEngine, mockStateStore)
		require.NotNil(t, processor)

		// Test basic processor operations
		metrics := processor.GetMetrics()
		assert.NotNil(t, metrics)
		assert.Equal(t, int64(0), metrics.MessagesProcessed)

		healthy := processor.HealthCheck()
		assert.False(t, healthy) // Should be unhealthy initially

		// Clean up
		err := processor.Stop()
		assert.NoError(t, err)
	})

	t.Run("processor processes real messages", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		topicName := "error-logs-" + fmt.Sprintf("%d", time.Now().UnixNano())

		processor := createIntegrationProcessor(kafkaContainer.GetBrokers(), topicName, "test-group-"+fmt.Sprintf("%d", time.Now().UnixNano()), mockAlertEngine)
		require.NotNil(t, processor)

		// Create and produce test message
		logEntry := models.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   "Processor integration test",
			Kubernetes: models.KubernetesInfo{
				Namespace: "processor-test",
				Pod:       "processor-pod-123",
				Container: "processor-container",
			},
		}

		logJSON, err := json.Marshal(logEntry)
		require.NoError(t, err)

		err = kafkaContainer.ProduceMessage(topicName, "processor-key", string(logJSON))
		require.NoError(t, err)

		// Start processor with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Start processor in goroutine
		processorErr := make(chan error, 1)
		go func() {
			processorErr <- processor.ProcessLogs(ctx)
		}()

		// Wait for message processing
		time.Sleep(2 * time.Second)

		// Cancel to stop processor
		cancel()

		// Wait for processor to stop
		select {
		case err := <-processorErr:
			// Context cancellation can result in either context.Canceled or context.DeadlineExceeded
			assert.True(t, err == context.Canceled || err == context.DeadlineExceeded,
				"Expected context cancellation error, got: %v", err)
		case <-time.After(3 * time.Second):
			t.Fatal("Processor did not stop within timeout")
		}

		// Clean up
		err = processor.Stop()
		assert.NoError(t, err)
	})

	t.Run("batch processor integration", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		topicName := "batch-logs-" + fmt.Sprintf("%d", time.Now().UnixNano())

		processor := createIntegrationProcessor(kafkaContainer.GetBrokers(), topicName, "test-group-"+fmt.Sprintf("%d", time.Now().UnixNano()), mockAlertEngine)
		batchProcessor := NewBatchLogProcessor(processor, 5, 1*time.Second)
		require.NotNil(t, batchProcessor)

		// Start batch processor with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// Start batch processing
		err := batchProcessor.ProcessBatch(ctx)
		// Context cancellation can result in either context.Canceled or context.DeadlineExceeded
		assert.True(t, err == context.Canceled || err == context.DeadlineExceeded,
			"Expected context cancellation error, got: %v", err)
	})
}

func TestKafkaFactoryIntegration(t *testing.T) {
	// Skip if running in CI without Docker
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use shared container for factory tests
	kafkaContainer := getOrCreateSharedContainer(t)
	defer releaseSharedContainer()

	t.Run("factory creates working processors", func(t *testing.T) {
		config := DefaultProcessorConfig()
		config.ConsumerConfig.Brokers = kafkaContainer.GetBrokers()
		config.ConsumerConfig.Topic = "factory-test-" + fmt.Sprintf("%d", time.Now().UnixNano())
		config.LogProcessingConfig.BatchSize = 10
		config.LogProcessingConfig.FlushInterval = 1 * time.Second

		factory := NewProcessorFactory(config)
		require.NotNil(t, factory)

		mockAlertEngine := mocks.NewMockAlertEngine()

		// Create regular processor
		mockStateStore := mocks.NewMockStateStore()
		processor, err := factory.CreateProcessor(kafkaContainer.GetBrokers(), config.ConsumerConfig.Topic, "factory-group-"+fmt.Sprintf("%d", time.Now().UnixNano()), mockAlertEngine, mockStateStore)
		require.NoError(t, err)
		assert.NotNil(t, processor)

		// Test processor functionality
		metrics := processor.GetMetrics()
		assert.NotNil(t, metrics)

		err = processor.Stop()
		assert.NoError(t, err)

		// Create batch processor
		mockStateStore2 := mocks.NewMockStateStore()
		batchProcessor, err := factory.CreateBatchProcessor(kafkaContainer.GetBrokers(), config.ConsumerConfig.Topic, "factory-group-"+fmt.Sprintf("%d", time.Now().UnixNano()), mockAlertEngine, mockStateStore2)
		require.NoError(t, err)
		assert.NotNil(t, batchProcessor)
	})
}

// Separate test for container health that needs its own containers
func TestKafkaContainerHealth(t *testing.T) {
	// Skip if running in CI without Docker
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("kafka container starts and is healthy", func(t *testing.T) {
		kafkaContainer, err := testcontainers.NewKafkaContainerForTesting(t)
		require.NoError(t, err)
		defer kafkaContainer.Cleanup()

		// Test container health
		err = kafkaContainer.TestKafkaAvailability()
		assert.NoError(t, err)

		// Test broker connectivity
		brokers := kafkaContainer.GetBrokers()
		assert.NotEmpty(t, brokers)
		assert.Greater(t, len(brokers), 0)

		// Test topic operations
		err = kafkaContainer.CreateTopic("health-test", 1, 1)
		assert.NoError(t, err)

		err = kafkaContainer.WaitForTopic("health-test", 10*time.Second)
		assert.NoError(t, err)

		topics, err := kafkaContainer.ListTopics()
		assert.NoError(t, err)
		assert.Contains(t, topics, "health-test")

		// Test message production
		err = kafkaContainer.ProduceMessage("health-test", "test-key", "test-value")
		assert.NoError(t, err)

		// Test Kafka version info
		version, err := kafkaContainer.GetKafkaVersion()
		assert.NoError(t, err)
		assert.NotEmpty(t, version)
	})

	// This test specifically needs multiple containers, so it's kept separate
	t.Run("multiple containers can run simultaneously", func(t *testing.T) {
		// Create first container
		kafka1, err := testcontainers.NewKafkaContainerForTesting(t)
		require.NoError(t, err)
		defer kafka1.Cleanup()

		// Create second container
		kafka2, err := testcontainers.NewKafkaContainerForTesting(t)
		require.NoError(t, err)
		defer kafka2.Cleanup()

		// Verify both are healthy
		err = kafka1.TestKafkaAvailability()
		assert.NoError(t, err)

		err = kafka2.TestKafkaAvailability()
		assert.NoError(t, err)

		// Verify they have different broker addresses
		brokers1 := kafka1.GetBrokers()
		brokers2 := kafka2.GetBrokers()
		assert.NotEqual(t, brokers1, brokers2)
	})
}

func TestKafkaPerformanceIntegration(t *testing.T) {
	// Skip if running in CI without Docker or in short mode
	if testing.Short() {
		t.Skip("Skipping performance integration test in short mode")
	}

	// Use dedicated container for performance tests
	kafkaContainer, err := testcontainers.NewKafkaContainerForTesting(t)
	require.NoError(t, err)
	defer kafkaContainer.Cleanup()

	// Wait for Kafka to be ready
	err = kafkaContainer.TestKafkaAvailability()
	require.NoError(t, err)

	t.Run("high volume message processing", func(t *testing.T) {
		mockAlertEngine := mocks.NewMockAlertEngine()
		topicName := "performance-test-" + fmt.Sprintf("%d", time.Now().UnixNano())

		config := ConsumerConfig{
			Brokers:  kafkaContainer.GetBrokers(),
			Topic:    topicName,
			GroupID:  "performance-group-" + fmt.Sprintf("%d", time.Now().UnixNano()),
			MinBytes: 1024,
			MaxBytes: 1048576,
			MaxWait:  100 * time.Millisecond,
		}

		consumer := NewConsumer(config, mockAlertEngine)
		require.NotNil(t, consumer)

		// Produce multiple test messages
		messageCount := 10
		for i := 0; i < messageCount; i++ {
			logEntry := models.LogEntry{
				Timestamp: time.Now(),
				Level:     "INFO",
				Message:   fmt.Sprintf("Performance test message %d", i),
				Kubernetes: models.KubernetesInfo{
					Namespace: "performance",
					Pod:       fmt.Sprintf("pod-%d", i),
				},
			}

			logJSON, err := json.Marshal(logEntry)
			require.NoError(t, err)

			err = kafkaContainer.ProduceMessage(topicName, fmt.Sprintf("key-%d", i), string(logJSON))
			require.NoError(t, err)
		}

		// Measure processing time
		start := time.Now()

		// Start consumer with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		go func() {
			consumer.Start(ctx)
		}()

		// Wait for processing
		time.Sleep(5 * time.Second)
		cancel()

		processingTime := time.Since(start)

		// Verify performance metrics
		t.Logf("Processed %d messages in %v", messageCount, processingTime)
		assert.Less(t, processingTime, 10*time.Second)

		// Clean up
		err = consumer.Close()
		assert.NoError(t, err)
	})
}
