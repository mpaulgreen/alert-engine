package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/log-monitoring/alert-engine/pkg/models"
	"github.com/segmentio/kafka-go"
)

// AlertEngine interface for processing logs
type AlertEngine interface {
	EvaluateLog(logEntry models.LogEntry)
}

// Consumer represents a Kafka consumer for processing log messages
type Consumer struct {
	reader      *kafka.Reader
	alertEngine AlertEngine
	config      ConsumerConfig
}

// ConsumerConfig holds configuration for the Kafka consumer
type ConsumerConfig struct {
	Brokers     []string      `json:"brokers"`
	Topic       string        `json:"topic"`
	GroupID     string        `json:"group_id"`
	MinBytes    int           `json:"min_bytes"`
	MaxBytes    int           `json:"max_bytes"`
	MaxWait     time.Duration `json:"max_wait"`
	StartOffset int64         `json:"start_offset"`
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(config ConsumerConfig, alertEngine AlertEngine) *Consumer {
	readerConfig := kafka.ReaderConfig{
		Brokers:     config.Brokers,
		Topic:       config.Topic,
		GroupID:     config.GroupID,
		MinBytes:    config.MinBytes,
		MaxBytes:    config.MaxBytes,
		MaxWait:     config.MaxWait,
		StartOffset: config.StartOffset,
	}

	reader := kafka.NewReader(readerConfig)

	return &Consumer{
		reader:      reader,
		alertEngine: alertEngine,
		config:      config,
	}
}

// parseMessageField parses JSON content from the message field to extract nested fields
func (c *Consumer) parseMessageField(logEntry *models.LogEntry) {
	if logEntry.Message == "" {
		return
	}

	// Try to parse the message as JSON to extract nested fields
	var nestedLog map[string]interface{}
	if err := json.Unmarshal([]byte(logEntry.Message), &nestedLog); err != nil {
		// Message is not JSON, keep as-is
		return
	}

	// Extract service field from nested JSON
	if service, ok := nestedLog["service"].(string); ok && service != "" {
		logEntry.Service = service
	}

	// Extract level field from nested JSON if current level is empty or default
	if level, ok := nestedLog["level"].(string); ok && level != "" {
		// Override level if current is empty or default
		if logEntry.Level == "" || logEntry.Level == "INFO" || logEntry.Level == "DEFAULT" {
			logEntry.Level = level
		}
	}

	// Extract actual message content from nested JSON
	if message, ok := nestedLog["message"].(string); ok && message != "" {
		logEntry.Message = message
	}

	// Extract timestamp from nested JSON if current timestamp is zero
	if timestampStr, ok := nestedLog["timestamp"].(string); ok && timestampStr != "" {
		if logEntry.Timestamp.IsZero() {
			if parsedTime, err := time.Parse(time.RFC3339Nano, timestampStr); err == nil {
				logEntry.Timestamp = parsedTime
			}
		}
	}
}

// Start begins consuming messages from Kafka
func (c *Consumer) Start(ctx context.Context) error {
	log.Printf("Starting Kafka consumer for topic: %s | Group: %s | Brokers: %v",
		c.config.Topic, c.config.GroupID, c.config.Brokers)

	startTime := time.Now()
	var messageCount int64
	var errorCount int64

	for {
		select {
		case <-ctx.Done():
			uptime := time.Since(startTime)
			errorRate := float64(0)
			if messageCount+errorCount > 0 {
				errorRate = float64(errorCount) / float64(messageCount+errorCount) * 100
			}
			log.Printf("Kafka consumer shutting down | Uptime: %v | Messages: %d | Errors: %d | Error rate: %.2f%%",
				uptime, messageCount, errorCount, errorRate)
			return ctx.Err()
		default:
			if err := c.processMessage(ctx); err != nil {
				errorCount++
				log.Printf("ERROR: Consumer message processing failed (error %d): %v", errorCount, err)

				// Log warning if error rate gets high
				total := messageCount + errorCount
				if total > 10 && float64(errorCount)/float64(total) > 0.05 { // 5% error rate threshold
					log.Printf("WARNING: Consumer error rate high: %.2f%% (%d errors out of %d total)",
						float64(errorCount)/float64(total)*100, errorCount, total)
				}

				// Continue processing other messages
				continue
			}

			messageCount++

			// Log throughput every 100 messages
			if messageCount%100 == 0 {
				uptime := time.Since(startTime)
				throughput := float64(messageCount) / uptime.Seconds()
				errorRate := float64(errorCount) / float64(messageCount+errorCount) * 100
				log.Printf("CONSUMER THROUGHPUT: %d messages in %v | Rate: %.2f msgs/sec | Error rate: %.2f%%",
					messageCount, uptime, throughput, errorRate)
			}
		}
	}
}

// processMessage processes a single message from Kafka
func (c *Consumer) processMessage(ctx context.Context) error {
	startTime := time.Now()

	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return fmt.Errorf("failed to read message from Kafka: %w", err)
	}

	// Generate correlation ID for this message
	correlationID := fmt.Sprintf("consumer_%d_%d_%d", msg.Partition, msg.Offset, time.Now().Unix())

	log.Printf("[%s] Received message from partition %d at offset %d | Size: %d bytes",
		correlationID, msg.Partition, msg.Offset, len(msg.Value))

	// Parse the log message
	var logEntry models.LogEntry
	if err := json.Unmarshal(msg.Value, &logEntry); err != nil {
		log.Printf("[%s] ERROR: Failed to unmarshal log entry: %v | Raw message: %s",
			correlationID, err, string(msg.Value))
		return fmt.Errorf("unmarshaling failed: %w", err)
	}

	// Store raw message for debugging
	logEntry.Raw = string(msg.Value)

	log.Printf("[%s] INFO: Successfully parsed message | Level: %s | Service: %s | Message preview: %.100s",
		correlationID, logEntry.Level, logEntry.Service, logEntry.Message)

	// Parse JSON message field to extract nested fields (like service)
	c.parseMessageField(&logEntry)

	log.Printf("[%s] INFO: After parsing nested fields | Level: %s | Service: %s | Message preview: %.100s",
		correlationID, logEntry.Level, logEntry.Service, logEntry.Message)

	// Process the log entry through the alert engine
	alertStartTime := time.Now()
	c.alertEngine.EvaluateLog(logEntry)
	alertProcessingTime := time.Since(alertStartTime)

	totalProcessingTime := time.Since(startTime)

	log.Printf("[%s] SUCCESS: Message processed successfully | Total time: %v | Alert evaluation time: %v",
		correlationID, totalProcessingTime, alertProcessingTime)

	return nil
}

// Close closes the Kafka consumer
func (c *Consumer) Close() error {
	log.Println("Closing Kafka consumer")
	return c.reader.Close()
}

// GetStats returns consumer statistics
func (c *Consumer) GetStats() kafka.ReaderStats {
	return c.reader.Stats()
}

// MessageProcessor handles batch processing of messages
type MessageProcessor struct {
	consumer    *Consumer
	batchSize   int
	flushTimer  *time.Timer
	buffer      []models.LogEntry
	alertEngine AlertEngine
}

// NewMessageProcessor creates a new message processor with batching
func NewMessageProcessor(consumer *Consumer, batchSize int, flushInterval time.Duration, alertEngine AlertEngine) *MessageProcessor {
	return &MessageProcessor{
		consumer:    consumer,
		batchSize:   batchSize,
		flushTimer:  time.NewTimer(flushInterval),
		buffer:      make([]models.LogEntry, 0, batchSize),
		alertEngine: alertEngine,
	}
}

// ProcessBatch processes messages in batches
func (mp *MessageProcessor) ProcessBatch(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			// Flush remaining messages before shutting down
			if len(mp.buffer) > 0 {
				mp.flushBuffer()
			}
			return ctx.Err()
		case <-mp.flushTimer.C:
			// Flush buffer on timer
			if len(mp.buffer) > 0 {
				mp.flushBuffer()
			}
		default:
			// Try to read a message
			if err := mp.readAndBuffer(ctx); err != nil {
				if err == context.Canceled {
					return err
				}
				log.Printf("Error reading message: %v", err)
				continue
			}

			// Check if buffer is full
			if len(mp.buffer) >= mp.batchSize {
				mp.flushBuffer()
			}
		}
	}
}

// readAndBuffer reads a message and adds it to the buffer
func (mp *MessageProcessor) readAndBuffer(ctx context.Context) error {
	msg, err := mp.consumer.reader.ReadMessage(ctx)
	if err != nil {
		return err
	}

	var logEntry models.LogEntry
	if err := json.Unmarshal(msg.Value, &logEntry); err != nil {
		log.Printf("Error unmarshaling log entry: %v", err)
		return err
	}

	logEntry.Raw = string(msg.Value)
	mp.buffer = append(mp.buffer, logEntry)

	return nil
}

// flushBuffer processes all messages in the buffer
func (mp *MessageProcessor) flushBuffer() {
	if len(mp.buffer) == 0 {
		return
	}

	log.Printf("Flushing %d messages from buffer", len(mp.buffer))

	for _, logEntry := range mp.buffer {
		mp.alertEngine.EvaluateLog(logEntry)
	}

	// Clear the buffer
	mp.buffer = mp.buffer[:0]

	// Reset the timer
	mp.flushTimer.Reset(time.Minute)
}

// ConsumerGroup manages multiple consumers for better throughput
type ConsumerGroup struct {
	consumers []*Consumer
	config    ConsumerConfig
}

// NewConsumerGroup creates a new consumer group
func NewConsumerGroup(config ConsumerConfig, alertEngine AlertEngine, consumerCount int) *ConsumerGroup {
	consumers := make([]*Consumer, consumerCount)

	for i := 0; i < consumerCount; i++ {
		consumers[i] = NewConsumer(config, alertEngine)
	}

	return &ConsumerGroup{
		consumers: consumers,
		config:    config,
	}
}

// Start starts all consumers in the group
func (cg *ConsumerGroup) Start(ctx context.Context) error {
	log.Printf("Starting consumer group with %d consumers", len(cg.consumers))

	// Start all consumers
	for i, consumer := range cg.consumers {
		go func(idx int, c *Consumer) {
			if err := c.Start(ctx); err != nil {
				log.Printf("Consumer %d error: %v", idx, err)
			}
		}(i, consumer)
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Close all consumers
	for _, consumer := range cg.consumers {
		if err := consumer.Close(); err != nil {
			log.Printf("Error closing consumer: %v", err)
		}
	}

	return nil
}

// GetGroupStats returns statistics for all consumers in the group
func (cg *ConsumerGroup) GetGroupStats() []kafka.ReaderStats {
	stats := make([]kafka.ReaderStats, len(cg.consumers))

	for i, consumer := range cg.consumers {
		stats[i] = consumer.GetStats()
	}

	return stats
}

// HealthCheck checks if the consumer is healthy
func (c *Consumer) HealthCheck() bool {
	stats := c.GetStats()

	// Check if the consumer is lagging significantly
	if stats.Lag > 1000 {
		log.Printf("Consumer lag is high: %d", stats.Lag)
		return false
	}

	return true
}

// DefaultConsumerConfig returns a default consumer configuration
func DefaultConsumerConfig() ConsumerConfig {
	return ConsumerConfig{
		Brokers:     []string{"127.0.0.1:9094"}, // Aligned with E2E test config
		Topic:       "application-logs",
		GroupID:     "alert-engine-e2e-fresh-20250716", // Aligned with E2E test config
		MinBytes:    1024,                              // 1KB - smaller default for better responsiveness
		MaxBytes:    1048576,                           // 1MB - match YAML config defaults
		MaxWait:     2 * time.Second,                   // Match YAML config default
		StartOffset: -2,                                // Match YAML config (-2 = earliest)
	}
}

// DefaultConsumerConfigFromEnv returns consumer config with environment-based defaults
func DefaultConsumerConfigFromEnv() ConsumerConfig {
	config := DefaultConsumerConfig()

	// Override with environment variables if present
	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		config.Brokers = strings.Split(brokers, ",")
	}
	if topic := os.Getenv("KAFKA_TOPIC"); topic != "" {
		config.Topic = topic
	}
	if groupID := os.Getenv("KAFKA_GROUP_ID"); groupID != "" {
		config.GroupID = groupID
	}

	return config
}
