package kafka

import (
    "context"
    "encoding/json"
    "log"
    "time"

    "github.com/segmentio/kafka-go"
    "github.com/log-monitoring/alert-engine/pkg/models"
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

// Start begins consuming messages from Kafka
func (c *Consumer) Start(ctx context.Context) error {
    log.Printf("Starting Kafka consumer for topic: %s", c.config.Topic)

    for {
        select {
        case <-ctx.Done():
            log.Println("Kafka consumer context cancelled, shutting down")
            return ctx.Err()
        default:
            if err := c.processMessage(ctx); err != nil {
                log.Printf("Error processing message: %v", err)
                // Continue processing other messages
                continue
            }
        }
    }
}

// processMessage processes a single message from Kafka
func (c *Consumer) processMessage(ctx context.Context) error {
    msg, err := c.reader.ReadMessage(ctx)
    if err != nil {
        return err
    }

    log.Printf("Received message from partition %d at offset %d", msg.Partition, msg.Offset)

    // Parse the log message
    var logEntry models.LogEntry
    if err := json.Unmarshal(msg.Value, &logEntry); err != nil {
        log.Printf("Error unmarshaling log entry: %v, raw message: %s", err, string(msg.Value))
        return err
    }

    // Store raw message for debugging
    logEntry.Raw = string(msg.Value)

    // Process the log entry through the alert engine
    c.alertEngine.EvaluateLog(logEntry)

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
        Brokers:     []string{"localhost:9092"},
        Topic:       "application-logs",
        GroupID:     "log-monitoring-group",
        MinBytes:    10e3, // 10KB
        MaxBytes:    10e6, // 10MB
        MaxWait:     1 * time.Second,
        StartOffset: kafka.LastOffset,
    }
} 