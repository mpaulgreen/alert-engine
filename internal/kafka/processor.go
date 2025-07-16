package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/log-monitoring/alert-engine/pkg/models"
)

// StateStore interface for log statistics
type StateStore interface {
	SaveLogStats(stats models.LogStats) error
	GetLogStats() (*models.LogStats, error)
}

// LogProcessor processes log messages from Kafka
type LogProcessor struct {
	consumer    *Consumer
	alertEngine AlertEngine
	stateStore  StateStore
	config      ProcessorConfig
	metrics     *ProcessorMetrics
	logStats    *models.LogStats
}

// LogProcessingConfig holds log processing specific configuration
type LogProcessingConfig struct {
	BatchSize       int           `json:"batch_size"`
	FlushInterval   time.Duration `json:"flush_interval"`
	RetryAttempts   int           `json:"retry_attempts"`
	RetryDelay      time.Duration `json:"retry_delay"`
	EnableMetrics   bool          `json:"enable_metrics"`
	DefaultLogLevel string        `json:"default_log_level"` // Default level for missing log levels
}

// ProcessorConfig holds configuration for the log processor
type ProcessorConfig struct {
	ConsumerConfig      ConsumerConfig      `json:"consumer_config"`
	LogProcessingConfig LogProcessingConfig `json:"log_processing_config"`
}

// ProcessorMetrics tracks processing metrics
type ProcessorMetrics struct {
	MessagesProcessed int64         `json:"messages_processed"`
	MessagesFailure   int64         `json:"messages_failure"`
	ProcessingTime    time.Duration `json:"processing_time"`
	LastProcessed     time.Time     `json:"last_processed"`
	ErrorRate         float64       `json:"error_rate"`
}

// NewLogProcessor creates a new log processor with log processing configuration
func NewLogProcessor(consumerConfig ConsumerConfig, logProcessingConfig LogProcessingConfig, alertEngine AlertEngine, stateStore StateStore) *LogProcessor {
	config := ProcessorConfig{
		ConsumerConfig:      consumerConfig,
		LogProcessingConfig: logProcessingConfig,
	}

	consumer := NewConsumer(config.ConsumerConfig, alertEngine)

	// Initialize log stats
	logStats := &models.LogStats{
		TotalLogs:     0,
		LogsByLevel:   make(map[string]int64),
		LogsByService: make(map[string]int64),
		LastUpdated:   time.Now(),
	}

	return &LogProcessor{
		consumer:    consumer,
		alertEngine: alertEngine,
		stateStore:  stateStore,
		config:      config,
		metrics:     &ProcessorMetrics{},
		logStats:    logStats,
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
	if err := lp.validateLogEntry(&logEntry); err != nil {
		log.Printf("Invalid log entry: %v", err)
		return err
	}

	// Process through alert engine
	lp.alertEngine.EvaluateLog(logEntry)

	// Update log statistics
	lp.updateLogStats(logEntry)

	// Update metrics
	lp.metrics.ProcessingTime = time.Since(startTime)

	return nil
}

// validateLogEntry validates a log entry
func (lp *LogProcessor) validateLogEntry(logEntry *models.LogEntry) error {
	if logEntry.Timestamp.IsZero() {
		// Use @timestamp if timestamp is not set
		if !logEntry.AtTimestamp.IsZero() {
			logEntry.Timestamp = logEntry.AtTimestamp
		} else {
			logEntry.Timestamp = time.Now()
		}
	}

	if logEntry.Level == "" {
		// Use configured default log level instead of hardcoded "INFO"
		defaultLevel := lp.config.LogProcessingConfig.DefaultLogLevel
		if defaultLevel == "" {
			defaultLevel = "INFO"
		}
		logEntry.Level = defaultLevel
	}

	if logEntry.Message == "" {
		return fmt.Errorf("log entry message is empty")
	}

	// Use the new GetNamespace() method to check for namespace
	if logEntry.GetNamespace() == "" {
		return fmt.Errorf("log entry missing kubernetes namespace")
	}

	return nil
}

// updateErrorRate calculates and updates the error rate
func (lp *LogProcessor) updateErrorRate() {
	total := lp.metrics.MessagesProcessed + lp.metrics.MessagesFailure
	if total > 0 {
		lp.metrics.ErrorRate = float64(lp.metrics.MessagesFailure) / float64(total)
	}
}

// updateLogStats updates log processing statistics
func (lp *LogProcessor) updateLogStats(logEntry models.LogEntry) {
	// Update total logs
	lp.logStats.TotalLogs++

	// Update logs by level
	if lp.logStats.LogsByLevel == nil {
		lp.logStats.LogsByLevel = make(map[string]int64)
	}
	lp.logStats.LogsByLevel[logEntry.Level]++

	// Update logs by service
	if lp.logStats.LogsByService == nil {
		lp.logStats.LogsByService = make(map[string]int64)
	}
	if logEntry.Service != "" {
		lp.logStats.LogsByService[logEntry.Service]++
	}

	// Update timestamp
	lp.logStats.LastUpdated = time.Now()

	// Save to store every 100 logs to avoid too frequent writes
	if lp.logStats.TotalLogs%100 == 0 {
		if err := lp.stateStore.SaveLogStats(*lp.logStats); err != nil {
			log.Printf("Error saving log stats: %v", err)
		}
	}
}

// GetMetrics returns processor metrics
func (lp *LogProcessor) GetMetrics() *ProcessorMetrics {
	return lp.metrics
}

// GetAlertEngine returns the alert engine
func (lp *LogProcessor) GetAlertEngine() AlertEngine {
	return lp.alertEngine
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
	if err := blp.processor.validateLogEntry(&logEntry); err != nil {
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
func (pf *ProcessorFactory) CreateProcessor(brokers []string, topic string, groupID string, alertEngine AlertEngine, stateStore StateStore) (*LogProcessor, error) {
	consumerConfig := ConsumerConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    pf.config.ConsumerConfig.MinBytes,
		MaxBytes:    pf.config.ConsumerConfig.MaxBytes,
		MaxWait:     pf.config.ConsumerConfig.MaxWait,
		StartOffset: pf.config.ConsumerConfig.StartOffset,
	}
	return NewLogProcessor(consumerConfig, pf.config.LogProcessingConfig, alertEngine, stateStore), nil
}

// CreateBatchProcessor creates a batch processor
func (pf *ProcessorFactory) CreateBatchProcessor(brokers []string, topic string, groupID string, alertEngine AlertEngine, stateStore StateStore) (*BatchLogProcessor, error) {
	processor, err := pf.CreateProcessor(brokers, topic, groupID, alertEngine, stateStore)
	if err != nil {
		return nil, err
	}

	return NewBatchLogProcessor(processor, pf.config.LogProcessingConfig.BatchSize, pf.config.LogProcessingConfig.FlushInterval), nil
}

// DefaultLogProcessingConfig returns default log processing configuration
func DefaultLogProcessingConfig() LogProcessingConfig {
	return LogProcessingConfig{
		BatchSize:       50,               // Better default for testing
		FlushInterval:   10 * time.Second, // Better default for testing
		RetryAttempts:   3,
		RetryDelay:      1 * time.Second,
		EnableMetrics:   true,
		DefaultLogLevel: "INFO", // Configurable default log level
	}
}

// DefaultProcessorConfig returns default processor configuration
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		ConsumerConfig:      DefaultConsumerConfig(),
		LogProcessingConfig: DefaultLogProcessingConfig(),
	}
}
