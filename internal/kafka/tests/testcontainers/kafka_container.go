package testcontainers

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/wait"
)

// KafkaContainer wraps the testcontainers Kafka container
type KafkaContainer struct {
	Container *kafka.KafkaContainer
	Brokers   []string
	ctx       context.Context
}

// NewKafkaContainer creates and starts a new Kafka test container
func NewKafkaContainer(ctx context.Context, t *testing.T) (*KafkaContainer, error) {
	kafkaContainer, err := kafka.RunContainer(ctx,
		kafka.WithClusterID("test-cluster"),
		testcontainers.WithImage("confluentinc/cp-kafka:7.4.0"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Kafka Server started").
				WithOccurrence(1).
				WithStartupTimeout(90*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start Kafka container: %w", err)
	}

	// Get the broker addresses
	brokers, err := kafkaContainer.Brokers(ctx)
	if err != nil {
		kafkaContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to get Kafka brokers: %w", err)
	}

	return &KafkaContainer{
		Container: kafkaContainer,
		Brokers:   brokers,
		ctx:       ctx,
	}, nil
}

// GetBrokers returns the Kafka broker addresses
func (kc *KafkaContainer) GetBrokers() []string {
	return kc.Brokers
}

// GetBrokerAddress returns the first broker address (for single broker tests)
func (kc *KafkaContainer) GetBrokerAddress() string {
	if len(kc.Brokers) > 0 {
		return kc.Brokers[0]
	}
	return ""
}

// GetHost returns the container host
func (kc *KafkaContainer) GetHost() string {
	host, err := kc.Container.Host(kc.ctx)
	if err != nil {
		return "localhost"
	}
	return host
}

// GetPort returns the mapped Kafka port
func (kc *KafkaContainer) GetPort() int {
	port, err := kc.Container.MappedPort(kc.ctx, "9093/tcp")
	if err != nil {
		return 9092
	}

	portInt, err := strconv.Atoi(port.Port())
	if err != nil {
		return 9092
	}
	return portInt
}

// CreateTopic creates a topic in the Kafka container
func (kc *KafkaContainer) CreateTopic(topicName string, partitions int, replicationFactor int) error {
	if partitions <= 0 {
		partitions = 1
	}
	if replicationFactor <= 0 {
		replicationFactor = 1
	}

	// Execute kafka-topics command inside the container
	cmd := []string{
		"kafka-topics",
		"--create",
		"--topic", topicName,
		"--partitions", strconv.Itoa(partitions),
		"--replication-factor", strconv.Itoa(replicationFactor),
		"--bootstrap-server", "localhost:9092",
	}

	_, _, err := kc.Container.Exec(kc.ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to create topic %s: %w", topicName, err)
	}

	return nil
}

// ListTopics lists all topics in the Kafka container
func (kc *KafkaContainer) ListTopics() ([]string, error) {
	cmd := []string{
		"kafka-topics",
		"--list",
		"--bootstrap-server", "localhost:9092",
	}

	_, output, err := kc.Container.Exec(kc.ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list topics: %w", err)
	}

	// Parse the output to extract topic names
	topics := []string{}
	if output != nil {
		// Read all output from the reader
		outputBytes, err := io.ReadAll(output)
		if err != nil {
			return nil, fmt.Errorf("failed to read topics output: %w", err)
		}

		outputStr := string(outputBytes)
		if len(outputStr) > 0 {
			// Split by newlines and filter out empty strings
			lines := strings.Split(outputStr, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "_") { // Skip internal topics
					topics = append(topics, line)
				}
			}
		}
	}

	return topics, nil
}

// ProduceMessage produces a test message to the specified topic
func (kc *KafkaContainer) ProduceMessage(topicName, key, value string) error {
	cmd := []string{
		"sh", "-c",
		fmt.Sprintf(`echo "%s" | kafka-console-producer --broker-list localhost:9092 --topic %s --property "key.separator=:" --property "parse.key=true" <<< "%s:%s"`,
			value, topicName, key, value),
	}

	_, _, err := kc.Container.Exec(kc.ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to produce message to topic %s: %w", topicName, err)
	}

	return nil
}

// WaitForTopic waits for a topic to be available
func (kc *KafkaContainer) WaitForTopic(topicName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		topics, err := kc.ListTopics()
		if err == nil {
			for _, topic := range topics {
				if topic == topicName {
					return nil
				}
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for topic %s to be available", topicName)
}

// Cleanup terminates the Kafka container
func (kc *KafkaContainer) Cleanup() error {
	if kc.Container != nil {
		return kc.Container.Terminate(kc.ctx)
	}
	return nil
}

// NewKafkaContainerForTesting creates a Kafka container specifically for testing
// This is a convenience function that sets up commonly used testing configurations
func NewKafkaContainerForTesting(t *testing.T) (*KafkaContainer, error) {
	ctx := context.Background()

	// Create the container
	kafkaContainer, err := NewKafkaContainer(ctx, t)
	if err != nil {
		return nil, err
	}

	// Create common test topics
	testTopics := []string{
		"test-logs",
		"application-logs",
		"error-logs",
		"batch-logs",
	}

	for _, topic := range testTopics {
		if err := kafkaContainer.CreateTopic(topic, 3, 1); err != nil {
			t.Logf("Warning: failed to create topic %s: %v", topic, err)
			// Don't fail the test, topics might auto-create
		}
	}

	// Wait a bit for topics to be ready
	time.Sleep(2 * time.Second)

	return kafkaContainer, nil
}

// TestKafkaAvailability tests if Kafka container is ready for testing
func (kc *KafkaContainer) TestKafkaAvailability() error {
	// Try to list topics as a health check
	_, err := kc.ListTopics()
	if err != nil {
		return fmt.Errorf("Kafka container not ready: %w", err)
	}
	return nil
}

// GetKafkaVersion returns the Kafka version from the container
func (kc *KafkaContainer) GetKafkaVersion() (string, error) {
	cmd := []string{
		"kafka-broker-api-versions",
		"--bootstrap-server", "localhost:9092",
	}

	_, output, err := kc.Container.Exec(kc.ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to get Kafka version: %w", err)
	}

	// Parse version from output
	var outputStr string
	if output != nil {
		outputBytes, err := io.ReadAll(output)
		if err != nil {
			return "", fmt.Errorf("failed to read version output: %w", err)
		}
		outputStr = string(outputBytes)
	}

	if len(outputStr) > 0 {
		// Extract version info from the first few lines
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "id:") {
				return strings.TrimSpace(line), nil
			}
		}
	}

	return "unknown", nil
}

// ConfigureForTesting applies testing-specific configurations
func (kc *KafkaContainer) ConfigureForTesting() error {
	// Set shorter retention and other test-friendly settings
	configs := map[string]string{
		"log.retention.hours":            "1",
		"log.segment.bytes":              "1048576", // 1MB
		"auto.create.topics.enable":      "true",
		"num.partitions":                 "3",
		"default.replication.factor":     "1",
		"min.insync.replicas":            "1",
		"unclean.leader.election.enable": "true",
	}

	for key, value := range configs {
		cmd := []string{
			"kafka-configs",
			"--bootstrap-server", "localhost:9092",
			"--alter",
			"--entity-type", "brokers",
			"--entity-name", "1",
			"--add-config", fmt.Sprintf("%s=%s", key, value),
		}

		_, _, err := kc.Container.Exec(kc.ctx, cmd)
		if err != nil {
			// Log warning but don't fail - some configs might not be applicable
			fmt.Printf("Warning: failed to set config %s=%s: %v\n", key, value, err)
		}
	}

	return nil
}
