package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/log-monitoring/alert-engine/pkg/models"
	"github.com/segmentio/kafka-go"
)

// LogProcessor processes log messages from Kafka
type LogProcessor struct {
	consumer    *Consumer
	alertEngine AlertEngine
	config      ProcessorConfig
	metrics     *ProcessorMetrics
}

// ProcessorConfig holds configuration for the log processor
type ProcessorConfig struct {
	ConsumerConfig ConsumerConfig `json:"consumer_config"`
	BatchSize      int            `json:"batch_size"`
	FlushInterval  time.Duration  `json:"flush_interval"`
	RetryAttempts  int            `json:"retry_attempts"`
	RetryDelay     time.Duration  `json:"retry_delay"`
	EnableMetrics  bool           `json:"enable_metrics"`
}

// ProcessorMetrics tracks processing metrics
type ProcessorMetrics struct {
	MessagesProcessed int64         `json:"messages_processed"`
	MessagesFailure   int64         `json:"messages_failure"`
	ProcessingTime    time.Duration `json:"processing_time"`
	LastProcessed     time.Time     `json:"last_processed"`
	ErrorRate         float64       `json:"error_rate"`
}

// NewLogProcessor creates a new log processor
func NewLogProcessor(brokers []string, topic string, alertEngine AlertEngine) *LogProcessor {
	config := ProcessorConfig{
		ConsumerConfig: ConsumerConfig{
			Brokers:     brokers,
			Topic:       topic,
			GroupID:     "log-monitoring-group",
			MinBytes:    10e3, // 10KB
			MaxBytes:    10e6, // 10MB
			MaxWait:     1 * time.Second,
			StartOffset: kafka.LastOffset,
		},
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
		EnableMetrics: true,
	}

	consumer := NewConsumer(config.ConsumerConfig, alertEngine)

	return &LogProcessor{
		consumer:    consumer,
		alertEngine: alertEngine,
		config:      config,
		metrics:     &ProcessorMetrics{},
	}
}

// ProcessLogs starts processing log messages
func (lp *LogProcessor) ProcessLogs(ctx context.Context) error {
	log.Printf("Starting log processor for topic: %s", lp.config.ConsumerConfig.Topic)

	for {
		select {
		case <-ctx.Done():
			log.Println("Log processor shutting down")
			return ctx.Err()
		default:
			if err := lp.processMessage(ctx); err != nil {
				lp.metrics.MessagesFailure++
				log.Printf("Error processing message: %v", err)

				// Retry logic
				if lp.config.RetryAttempts > 0 {
					if err := lp.retryProcessing(ctx); err != nil {
						log.Printf("Retry failed: %v", err)
					}
				}

				continue
			}

			lp.metrics.MessagesProcessed++
			lp.metrics.LastProcessed = time.Now()
			lp.updateErrorRate()
		}
	}
}

// processMessage processes a single log message
func (lp *LogProcessor) processMessage(ctx context.Context) error {
	startTime := time.Now()

	msg, err := lp.consumer.reader.ReadMessage(ctx)
	if err != nil {
		return err
	}

	// Parse the log message
	var logEntry models.LogEntry
	if err := json.Unmarshal(msg.Value, &logEntry); err != nil {
		log.Printf("Error unmarshaling log entry: %v, raw message: %s", err, string(msg.Value))
		return err
	}

	// Store raw message for debugging
	logEntry.Raw = string(msg.Value)

	// Validate log entry
	if err := lp.validateLogEntry(logEntry); err != nil {
		log.Printf("Invalid log entry: %v", err)
		return err
	}

	// Process through alert engine
	lp.alertEngine.EvaluateLog(logEntry)

	// Update metrics
	lp.metrics.ProcessingTime = time.Since(startTime)

	return nil
}

// validateLogEntry validates a log entry
func (lp *LogProcessor) validateLogEntry(logEntry models.LogEntry) error {
	if logEntry.Timestamp.IsZero() {
		logEntry.Timestamp = time.Now()
	}

	if logEntry.Level == "" {
		logEntry.Level = "INFO"
	}

	if logEntry.Message == "" {
		return fmt.Errorf("log entry message is empty")
	}

	if logEntry.Kubernetes.Namespace == "" {
		return fmt.Errorf("log entry missing kubernetes namespace")
	}

	return nil
}

// retryProcessing implements retry logic for failed messages
func (lp *LogProcessor) retryProcessing(ctx context.Context) error {
	for attempt := 1; attempt <= lp.config.RetryAttempts; attempt++ {
		log.Printf("Retry attempt %d/%d", attempt, lp.config.RetryAttempts)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(lp.config.RetryDelay * time.Duration(attempt)):
			if err := lp.processMessage(ctx); err == nil {
				log.Printf("Retry successful on attempt %d", attempt)
				return nil
			}
		}
	}

	return fmt.Errorf("all retry attempts failed")
}

// updateErrorRate calculates and updates the error rate
func (lp *LogProcessor) updateErrorRate() {
	total := lp.metrics.MessagesProcessed + lp.metrics.MessagesFailure
	if total > 0 {
		lp.metrics.ErrorRate = float64(lp.metrics.MessagesFailure) / float64(total)
	}
}

// GetMetrics returns processor metrics
func (lp *LogProcessor) GetMetrics() *ProcessorMetrics {
	return lp.metrics
}

// HealthCheck checks if the processor is healthy
func (lp *LogProcessor) HealthCheck() bool {
	// Check if error rate is too high
	if lp.metrics.ErrorRate > 0.1 { // 10% error rate threshold
		log.Printf("Processor error rate is high: %.2f%%", lp.metrics.ErrorRate*100)
		return false
	}

	// Check if consumer is healthy
	if !lp.consumer.HealthCheck() {
		return false
	}

	// Check if processing is recent
	if time.Since(lp.metrics.LastProcessed) > 5*time.Minute {
		log.Printf("No messages processed in the last 5 minutes")
		return false
	}

	return true
}

// Stop gracefully stops the processor
func (lp *LogProcessor) Stop() error {
	log.Println("Stopping log processor")
	return lp.consumer.Close()
}

// BatchLogProcessor processes logs in batches for better performance
type BatchLogProcessor struct {
	processor     *LogProcessor
	batchBuffer   []models.LogEntry
	batchSize     int
	flushInterval time.Duration
	flushTimer    *time.Timer
}

// NewBatchLogProcessor creates a new batch processor
func NewBatchLogProcessor(processor *LogProcessor, batchSize int, flushInterval time.Duration) *BatchLogProcessor {
	return &BatchLogProcessor{
		processor:     processor,
		batchBuffer:   make([]models.LogEntry, 0, batchSize),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		flushTimer:    time.NewTimer(flushInterval),
	}
}

// ProcessBatch processes logs in batches
func (blp *BatchLogProcessor) ProcessBatch(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			// Flush remaining messages
			if len(blp.batchBuffer) > 0 {
				blp.flushBatch()
			}
			return ctx.Err()
		case <-blp.flushTimer.C:
			// Flush on timer
			if len(blp.batchBuffer) > 0 {
				blp.flushBatch()
			}
		default:
			// Try to read and buffer a message
			if err := blp.readAndBuffer(ctx); err != nil {
				if err == context.Canceled {
					return err
				}
				log.Printf("Error reading message: %v", err)
				continue
			}

			// Check if batch is full
			if len(blp.batchBuffer) >= blp.batchSize {
				blp.flushBatch()
			}
		}
	}
}

// readAndBuffer reads a message and adds it to the batch buffer
func (blp *BatchLogProcessor) readAndBuffer(ctx context.Context) error {
	msg, err := blp.processor.consumer.reader.ReadMessage(ctx)
	if err != nil {
		return err
	}

	var logEntry models.LogEntry
	if err := json.Unmarshal(msg.Value, &logEntry); err != nil {
		log.Printf("Error unmarshaling log entry: %v", err)
		return err
	}

	// Validate log entry
	if err := blp.processor.validateLogEntry(logEntry); err != nil {
		log.Printf("Invalid log entry: %v", err)
		return err
	}

	logEntry.Raw = string(msg.Value)
	blp.batchBuffer = append(blp.batchBuffer, logEntry)

	return nil
}

// flushBatch processes all messages in the batch buffer
func (blp *BatchLogProcessor) flushBatch() {
	if len(blp.batchBuffer) == 0 {
		return
	}

	log.Printf("Processing batch of %d messages", len(blp.batchBuffer))

	for _, logEntry := range blp.batchBuffer {
		blp.processor.alertEngine.EvaluateLog(logEntry)
		blp.processor.metrics.MessagesProcessed++
	}

	// Clear the buffer
	blp.batchBuffer = blp.batchBuffer[:0]

	// Reset the timer
	blp.flushTimer.Reset(blp.flushInterval)

	// Update metrics
	blp.processor.metrics.LastProcessed = time.Now()
	blp.processor.updateErrorRate()
}

// ProcessorFactory creates different types of processors
type ProcessorFactory struct {
	config ProcessorConfig
}

// NewProcessorFactory creates a new processor factory
func NewProcessorFactory(config ProcessorConfig) *ProcessorFactory {
	return &ProcessorFactory{
		config: config,
	}
}

// CreateProcessor creates a processor based on configuration
func (pf *ProcessorFactory) CreateProcessor(brokers []string, topic string, alertEngine AlertEngine) (*LogProcessor, error) {
	config := pf.config
	config.ConsumerConfig.Brokers = brokers
	config.ConsumerConfig.Topic = topic

	consumer := NewConsumer(config.ConsumerConfig, alertEngine)

	return &LogProcessor{
		consumer:    consumer,
		alertEngine: alertEngine,
		config:      config,
		metrics:     &ProcessorMetrics{},
	}, nil
}

// CreateBatchProcessor creates a batch processor
func (pf *ProcessorFactory) CreateBatchProcessor(brokers []string, topic string, alertEngine AlertEngine) (*BatchLogProcessor, error) {
	processor, err := pf.CreateProcessor(brokers, topic, alertEngine)
	if err != nil {
		return nil, err
	}

	return NewBatchLogProcessor(processor, pf.config.BatchSize, pf.config.FlushInterval), nil
}

// DefaultProcessorConfig returns default processor configuration
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		ConsumerConfig: DefaultConsumerConfig(),
		BatchSize:      100,
		FlushInterval:  5 * time.Second,
		RetryAttempts:  3,
		RetryDelay:     1 * time.Second,
		EnableMetrics:  true,
	}
}
