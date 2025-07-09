# Kafka Package

The `internal/kafka` package provides comprehensive Kafka integration for consuming log messages and processing them through the alert engine. It's designed with multiple processing patterns, robust error handling, and comprehensive monitoring capabilities.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Components](#components)
  - [Consumer](#consumer)
  - [Processor](#processor)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Usage Examples](#usage-examples)
- [Error Handling](#error-handling)
- [Monitoring & Metrics](#monitoring--metrics)
- [Best Practices](#best-practices)
- [API Reference](#api-reference)

## Overview

This package implements a robust Kafka consumer system that:
- Consumes log messages from Kafka topics
- Validates and processes log entries
- Integrates with the alert engine for real-time alerting
- Provides comprehensive metrics and health monitoring
- Supports multiple processing patterns (single, batch, consumer groups)

## Architecture

```
Kafka Topic → Consumer → Message Validation → Alert Engine → Metrics
                ↓
          Retry Logic (on failure)
                ↓
        Health Monitoring
```

The package is built with two main components:

1. **Consumer** (`consumer.go`): Low-level Kafka message consumption
2. **Processor** (`processor.go`): High-level processing with validation, retry logic, and metrics

## Components

### Consumer

The consumer component provides basic Kafka message consumption functionality:

#### Key Types:
- **`Consumer`**: Main Kafka consumer wrapper
- **`ConsumerConfig`**: Configuration for Kafka connection and behavior
- **`MessageProcessor`**: Batch processing for better performance
- **`ConsumerGroup`**: Multiple consumers for higher throughput

#### Features:
- Continuous message consumption
- JSON deserialization to `models.LogEntry`
- Integration with alert engine
- Graceful shutdown handling
- Consumer group management

### Processor

The processor component adds advanced processing capabilities:

#### Key Types:
- **`LogProcessor`**: Enhanced message processing with validation
- **`ProcessorConfig`**: Extended configuration with retry and metrics
- **`ProcessorMetrics`**: Comprehensive processing statistics
- **`BatchLogProcessor`**: Batch processing for high throughput

#### Features:
- Message validation and sanitization
- Automatic retry logic with exponential backoff
- Comprehensive metrics collection
- Health monitoring and circuit breaker patterns
- Factory pattern for different processor types

## Quick Start

### Basic Consumer Setup

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/log-monitoring/alert-engine/internal/kafka"
    "github.com/log-monitoring/alert-engine/internal/alerting"
)

func main() {
    // Create alert engine
    alertEngine := alerting.NewEngine()
    
    // Create consumer with default config
    config := kafka.DefaultConsumerConfig()
    config.Brokers = []string{"localhost:9092"}
    config.Topic = "application-logs"
    
    consumer := kafka.NewConsumer(config, alertEngine)
    
    // Start consuming
    ctx := context.Background()
    if err := consumer.Start(ctx); err != nil {
        log.Fatal(err)
    }
}
```

### Advanced Processor Setup

```go
package main

import (
    "context"
    "log"
    
    "github.com/log-monitoring/alert-engine/internal/kafka"
    "github.com/log-monitoring/alert-engine/internal/alerting"
)

func main() {
    // Create alert engine
    alertEngine := alerting.NewEngine()
    
    // Create processor
    brokers := []string{"localhost:9092"}
    topic := "application-logs"
    
    processor := kafka.NewLogProcessor(brokers, topic, alertEngine)
    
    // Start processing with monitoring
    ctx := context.Background()
    go func() {
        if err := processor.ProcessLogs(ctx); err != nil {
            log.Printf("Processor error: %v", err)
        }
    }()
    
    // Monitor health
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        if !processor.HealthCheck() {
            log.Println("Processor is unhealthy!")
        }
        
        metrics := processor.GetMetrics()
        log.Printf("Processed: %d, Failed: %d, Error Rate: %.2f%%",
            metrics.MessagesProcessed,
            metrics.MessagesFailure,
            metrics.ErrorRate*100)
    }
}
```

## Configuration

### ConsumerConfig

```go
type ConsumerConfig struct {
    Brokers     []string      `json:"brokers"`       // Kafka broker addresses
    Topic       string        `json:"topic"`         // Topic to consume from
    GroupID     string        `json:"group_id"`      // Consumer group ID
    MinBytes    int           `json:"min_bytes"`     // Minimum bytes to fetch
    MaxBytes    int           `json:"max_bytes"`     // Maximum bytes to fetch
    MaxWait     time.Duration `json:"max_wait"`      // Maximum wait time
    StartOffset int64         `json:"start_offset"`  // Starting offset
}
```

### ProcessorConfig

```go
type ProcessorConfig struct {
    ConsumerConfig ConsumerConfig    `json:"consumer_config"`  // Base consumer config
    BatchSize      int               `json:"batch_size"`       // Batch processing size
    FlushInterval  time.Duration     `json:"flush_interval"`   // Batch flush interval
    RetryAttempts  int               `json:"retry_attempts"`   // Max retry attempts
    RetryDelay     time.Duration     `json:"retry_delay"`      // Base retry delay
    EnableMetrics  bool              `json:"enable_metrics"`   // Enable metrics collection
}
```

### Default Configurations

```go
// Default consumer configuration
config := kafka.DefaultConsumerConfig()
// Brokers: ["localhost:9092"]
// Topic: "application-logs"
// GroupID: "log-monitoring-group"
// MinBytes: 10KB, MaxBytes: 10MB
// MaxWait: 1 second

// Default processor configuration
procConfig := kafka.DefaultProcessorConfig()
// BatchSize: 100
// FlushInterval: 5 seconds
// RetryAttempts: 3
// RetryDelay: 1 second
```

## Usage Examples

### Single Consumer

```go
// Create and start a single consumer
config := kafka.DefaultConsumerConfig()
consumer := kafka.NewConsumer(config, alertEngine)
consumer.Start(ctx)
```

### Consumer Group (Multiple Consumers)

```go
// Create consumer group for higher throughput
config := kafka.DefaultConsumerConfig()
group := kafka.NewConsumerGroup(config, alertEngine, 3) // 3 consumers
group.Start(ctx)
```

### Batch Processing

```go
// Create batch processor for high-throughput scenarios
factory := kafka.NewProcessorFactory(kafka.DefaultProcessorConfig())
batchProcessor, _ := factory.CreateBatchProcessor(brokers, topic, alertEngine)
batchProcessor.ProcessBatch(ctx)
```

### Custom Message Processing

```go
// Create processor with custom configuration
config := kafka.ProcessorConfig{
    ConsumerConfig: kafka.ConsumerConfig{
        Brokers: []string{"broker1:9092", "broker2:9092"},
        Topic:   "custom-logs",
        GroupID: "custom-group",
    },
    BatchSize:     200,
    FlushInterval: 10 * time.Second,
    RetryAttempts: 5,
    RetryDelay:    2 * time.Second,
    EnableMetrics: true,
}

factory := kafka.NewProcessorFactory(config)
processor, _ := factory.CreateProcessor(config.ConsumerConfig.Brokers, 
                                       config.ConsumerConfig.Topic, 
                                       alertEngine)
```

## Error Handling

The package implements a multi-layered error handling strategy:

### 1. Message-Level Errors
- Invalid JSON messages are logged and skipped
- Processing continues with the next message
- Error metrics are updated

### 2. Retry Logic
- Configurable retry attempts with exponential backoff
- Automatic retry for transient failures
- Circuit breaker pattern for persistent failures

### 3. Health Monitoring
- Continuous health checks based on error rates
- Lag monitoring for consumer performance
- Automatic recovery mechanisms

### 4. Graceful Shutdown
- Context-based cancellation
- Proper resource cleanup
- Flush remaining messages before shutdown

## Monitoring & Metrics

### ProcessorMetrics

```go
type ProcessorMetrics struct {
    MessagesProcessed int64         // Total messages processed
    MessagesFailure   int64         // Total failed messages
    ProcessingTime    time.Duration // Average processing time
    LastProcessed     time.Time     // Last processed message timestamp
    ErrorRate         float64       // Current error rate (0.0-1.0)
}
```

### Health Checks

```go
// Check processor health
if processor.HealthCheck() {
    log.Println("Processor is healthy")
} else {
    log.Println("Processor is unhealthy - check logs")
}

// Get detailed metrics
metrics := processor.GetMetrics()
log.Printf("Error rate: %.2f%%", metrics.ErrorRate*100)
```

### Consumer Statistics

```go
// Get Kafka consumer statistics
stats := consumer.GetStats()
log.Printf("Consumer lag: %d", stats.Lag)
log.Printf("Messages consumed: %d", stats.Messages)
```

## Best Practices

### 1. Configuration
- Use environment variables for broker addresses and credentials
- Set appropriate batch sizes based on your throughput requirements
- Configure consumer groups for horizontal scaling

### 2. Error Handling
- Monitor error rates and set up alerts
- Implement proper retry logic for transient failures
- Use circuit breakers for cascading failure prevention

### 3. Performance
- Use batch processing for high-throughput scenarios
- Tune consumer configuration (MinBytes, MaxBytes, MaxWait)
- Consider consumer groups for parallel processing

### 4. Monitoring
- Implement comprehensive logging
- Set up metrics collection and alerting
- Monitor consumer lag and processing rates

### 5. Testing
- Use dependency injection for easier testing
- Mock the AlertEngine interface for unit tests
- Test with various message formats and error scenarios

## API Reference

### Core Functions

#### NewConsumer
```go
func NewConsumer(config ConsumerConfig, alertEngine AlertEngine) *Consumer
```

#### NewLogProcessor
```go
func NewLogProcessor(brokers []string, topic string, alertEngine AlertEngine) *LogProcessor
```

#### NewConsumerGroup
```go
func NewConsumerGroup(config ConsumerConfig, alertEngine AlertEngine, consumerCount int) *ConsumerGroup
```

#### NewProcessorFactory
```go
func NewProcessorFactory(config ProcessorConfig) *ProcessorFactory
```

### Interface Requirements

#### AlertEngine Interface
```go
type AlertEngine interface {
    EvaluateLog(logEntry models.LogEntry)
}
```

Your alert engine implementation must satisfy this interface to work with the Kafka consumers and processors.

---

For more detailed examples and advanced usage patterns, refer to the source code and tests in this package. 