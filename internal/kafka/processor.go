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

	processingStartTime := time.Now()
	var messageCount int64

	for {
		select {
		case <-ctx.Done():
			uptime := time.Since(processingStartTime)
			log.Printf("Log processor shutting down | Uptime: %v | Messages processed: %d | Messages failed: %d",
				uptime, lp.metrics.MessagesProcessed, lp.metrics.MessagesFailure)
			return ctx.Err()
		default:
			if err := lp.processMessage(ctx); err != nil {
				lp.metrics.MessagesFailure++
				log.Printf("ERROR: Message processing failed (attempt %d): %v", lp.metrics.MessagesFailure, err)
				lp.updateErrorRate()

				// Log a warning if error rate is getting high
				if lp.metrics.ErrorRate > 0.05 { // 5% error rate threshold
					log.Printf("WARNING: High error rate detected: %.2f%% (%d failures out of %d total)",
						lp.metrics.ErrorRate*100, lp.metrics.MessagesFailure, lp.metrics.MessagesProcessed+lp.metrics.MessagesFailure)
				}
				continue
			}

			lp.metrics.MessagesProcessed++
			lp.metrics.LastProcessed = time.Now()
			lp.updateErrorRate()
			messageCount++

			// Log throughput summary every 100 messages
			if messageCount%100 == 0 {
				uptime := time.Since(processingStartTime)
				throughput := float64(messageCount) / uptime.Seconds()
				log.Printf("THROUGHPUT: Processed %d messages in %v | Rate: %.2f msgs/sec | Error rate: %.2f%%",
					messageCount, uptime, throughput, lp.metrics.ErrorRate*100)
			}
		}
	}
}

// processMessage processes a single log message
func (lp *LogProcessor) processMessage(ctx context.Context) error {
	startTime := time.Now()

	msg, err := lp.consumer.reader.ReadMessage(ctx)
	if err != nil {
		return fmt.Errorf("failed to read message from Kafka: %w", err)
	}

	// Generate correlation ID for this message
	correlationID := fmt.Sprintf("msg_%d_%d_%d", msg.Partition, msg.Offset, time.Now().Unix())

	log.Printf("[%s] Received message from partition %d at offset %d", correlationID, msg.Partition, msg.Offset)

	// Parse the log message
	var logEntry models.LogEntry
	if err := json.Unmarshal(msg.Value, &logEntry); err != nil {
		log.Printf("[%s] ERROR: Failed to unmarshal log entry: %v | Raw message: %s", correlationID, err, string(msg.Value))
		return fmt.Errorf("unmarshaling failed: %w", err)
	}

	// Store raw message for debugging
	logEntry.Raw = string(msg.Value)

	log.Printf("[%s] INFO: Successfully parsed message | Level: %s | Service: %s | Message preview: %.100s",
		correlationID, logEntry.Level, logEntry.Service, logEntry.Message)

	// Parse JSON message field to extract nested fields (like service)
	lp.parseMessageField(&logEntry)

	log.Printf("[%s] INFO: After parsing nested fields | Level: %s | Service: %s | Message preview: %.100s",
		correlationID, logEntry.Level, logEntry.Service, logEntry.Message)

	// Validate log entry
	if err := lp.validateLogEntry(&logEntry); err != nil {
		log.Printf("[%s] ERROR: Validation failed: %v | Entry details - Level: %s, Service: %s, Namespace: %s, Message length: %d",
			correlationID, err, logEntry.Level, logEntry.Service, logEntry.GetNamespace(), len(logEntry.Message))
		return fmt.Errorf("validation failed: %w", err)
	}

	log.Printf("[%s] INFO: Validation passed | Namespace: %s | Level: %s | Service: %s",
		correlationID, logEntry.GetNamespace(), logEntry.Level, logEntry.Service)

	// Process through alert engine
	alertStartTime := time.Now()
	lp.alertEngine.EvaluateLog(logEntry)
	alertProcessingTime := time.Since(alertStartTime)

	// Update log statistics
	lp.updateLogStats(logEntry)

	// Update metrics
	totalProcessingTime := time.Since(startTime)
	lp.metrics.ProcessingTime = totalProcessingTime

	log.Printf("[%s] SUCCESS: Message processed successfully | Total time: %v | Alert evaluation time: %v | Stats updated",
		correlationID, totalProcessingTime, alertProcessingTime)

	// Log periodic summary every 50 successful messages
	if lp.metrics.MessagesProcessed%50 == 0 {
		lp.logProcessingSummary()
	}

	return nil
}

// parseMessageField parses JSON content from the message field to extract nested fields
func (lp *LogProcessor) parseMessageField(logEntry *models.LogEntry) {
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

	// Update logs by service (FIXED VERSION)
	if lp.logStats.LogsByService == nil {
		lp.logStats.LogsByService = make(map[string]int64)
	}
	serviceName := lp.getServiceName(logEntry)
	if serviceName != "unknown" {
		lp.logStats.LogsByService[serviceName]++
	}

	// Update timestamp
	lp.logStats.LastUpdated = time.Now()

	// TODO: Make save frequency configurable via config (currently every 5 logs)
	// Save to store every 5 logs to ensure timely updates for testing
	if lp.logStats.TotalLogs%5 == 0 {
		if err := lp.stateStore.SaveLogStats(*lp.logStats); err != nil {
			log.Printf("Error saving log stats: %v", err)
		}
	}
}

// getServiceName extracts service name from log entry using robust logic
func (lp *LogProcessor) getServiceName(logEntry models.LogEntry) string {
	// First check top-level service field
	if logEntry.Service != "" {
		return logEntry.Service
	}
	// Fallback to Kubernetes app label (like Alert Engine does)
	if appLabel, exists := logEntry.Kubernetes.Labels["app"]; exists {
		return appLabel
	}
	// Fallback to Kubernetes service label
	if serviceLabel, exists := logEntry.Kubernetes.Labels["service"]; exists {
		return serviceLabel
	}
	return "unknown"
}

// logProcessingSummary logs a summary of processing statistics
func (lp *LogProcessor) logProcessingSummary() {
	errorRate := float64(0)
	total := lp.metrics.MessagesProcessed + lp.metrics.MessagesFailure
	if total > 0 {
		errorRate = float64(lp.metrics.MessagesFailure) / float64(total) * 100
	}

	log.Printf("=== PROCESSING SUMMARY ===")
	log.Printf("Messages processed: %d | Failed: %d | Error rate: %.2f%%",
		lp.metrics.MessagesProcessed, lp.metrics.MessagesFailure, errorRate)
	log.Printf("Total logs in stats: %d | Last processing time: %v",
		lp.logStats.TotalLogs, lp.metrics.ProcessingTime)
	log.Printf("Top 5 log levels: %v", lp.getTopLogLevels(5))
	log.Printf("Top 5 services: %v", lp.getTopServices(5))
	log.Printf("Last updated: %v", lp.logStats.LastUpdated.Format(time.RFC3339))
}

// getTopLogLevels returns the top N log levels by count
func (lp *LogProcessor) getTopLogLevels(n int) map[string]int64 {
	if len(lp.logStats.LogsByLevel) <= n {
		return lp.logStats.LogsByLevel
	}

	// For simplicity, return first n items (could be enhanced with sorting)
	result := make(map[string]int64)
	count := 0
	for level, cnt := range lp.logStats.LogsByLevel {
		if count >= n {
			break
		}
		result[level] = cnt
		count++
	}
	return result
}

// getTopServices returns the top N services by log count
func (lp *LogProcessor) getTopServices(n int) map[string]int64 {
	if len(lp.logStats.LogsByService) <= n {
		return lp.logStats.LogsByService
	}

	// For simplicity, return first n items (could be enhanced with sorting)
	result := make(map[string]int64)
	count := 0
	for service, cnt := range lp.logStats.LogsByService {
		if count >= n {
			break
		}
		result[service] = cnt
		count++
	}
	return result
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

	batchStartTime := time.Now()
	batchID := fmt.Sprintf("batch_%d_%d", time.Now().Unix(), len(blp.batchBuffer))

	log.Printf("[%s] INFO: Starting batch processing | Messages: %d", batchID, len(blp.batchBuffer))

	successCount := 0
	errorCount := 0

	for i, logEntry := range blp.batchBuffer {
		correlationID := fmt.Sprintf("%s_msg_%d", batchID, i)

		// Validate each log entry in batch
		if err := blp.processor.validateLogEntry(&logEntry); err != nil {
			log.Printf("[%s] ERROR: Batch validation failed: %v | Level: %s | Service: %s",
				correlationID, err, logEntry.Level, logEntry.Service)
			errorCount++
			blp.processor.metrics.MessagesFailure++
			continue
		}

		// Process through alert engine
		alertStartTime := time.Now()
		blp.processor.alertEngine.EvaluateLog(logEntry)
		alertTime := time.Since(alertStartTime)

		blp.processor.metrics.MessagesProcessed++
		successCount++

		// Log individual message success (less verbose for batches)
		if i < 5 || i%10 == 0 { // Log first 5 and every 10th message
			log.Printf("[%s] SUCCESS: Message processed | Alert time: %v | Level: %s | Service: %s",
				correlationID, alertTime, logEntry.Level, logEntry.Service)
		}
	}

	// Clear the buffer
	blp.batchBuffer = blp.batchBuffer[:0]

	// Reset the timer
	blp.flushTimer.Reset(blp.flushInterval)

	// Update metrics
	blp.processor.metrics.LastProcessed = time.Now()
	blp.processor.updateErrorRate()

	batchTime := time.Since(batchStartTime)
	log.Printf("[%s] SUCCESS: Batch completed | Total time: %v | Success: %d | Errors: %d | Rate: %.2f msgs/sec",
		batchID, batchTime, successCount, errorCount, float64(successCount)/batchTime.Seconds())
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
