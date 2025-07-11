package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// MockKafkaReader is a mock implementation of the Kafka reader interface
type MockKafkaReader struct {
	mu              sync.RWMutex
	messages        []kafka.Message
	currentIndex    int
	shouldFail      bool
	failureMsg      string
	stats           kafka.ReaderStats
	closed          bool
	readDelay       time.Duration
	contextCanceled bool
}

// NewMockKafkaReader creates a new mock Kafka reader
func NewMockKafkaReader() *MockKafkaReader {
	return &MockKafkaReader{
		messages:     make([]kafka.Message, 0),
		currentIndex: 0,
		shouldFail:   false,
		stats: kafka.ReaderStats{
			Dials:      0,
			Fetches:    0,
			Messages:   0,
			Bytes:      0,
			Rebalances: 0,
			Timeouts:   0,
			Errors:     0,
		},
	}
}

// SetShouldFail configures the mock to fail operations
func (m *MockKafkaReader) SetShouldFail(shouldFail bool, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
	m.failureMsg = message
}

// AddMessage adds a message to the mock reader queue
func (m *MockKafkaReader) AddMessage(message kafka.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, message)
}

// AddMessages adds multiple messages to the mock reader queue
func (m *MockKafkaReader) AddMessages(messages []kafka.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, messages...)
}

// ReadMessage reads a message from the mock queue
func (m *MockKafkaReader) ReadMessage(ctx context.Context) (kafka.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simulate read delay if set
	if m.readDelay > 0 {
		time.Sleep(m.readDelay)
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		m.contextCanceled = true
		return kafka.Message{}, ctx.Err()
	default:
	}

	if m.shouldFail {
		m.stats.Errors++
		return kafka.Message{}, fmt.Errorf("mock error: %s", m.failureMsg)
	}

	if m.closed {
		return kafka.Message{}, fmt.Errorf("reader is closed")
	}

	if m.currentIndex >= len(m.messages) {
		// No more messages, block or return timeout
		if m.contextCanceled {
			return kafka.Message{}, context.Canceled
		}
		// Simulate waiting for new messages
		time.Sleep(100 * time.Millisecond)
		return kafka.Message{}, fmt.Errorf("no messages available")
	}

	message := m.messages[m.currentIndex]
	m.currentIndex++

	// Update stats
	m.stats.Messages++
	m.stats.Bytes += int64(len(message.Value))
	m.stats.Fetches++

	return message, nil
}

// Close closes the mock reader
func (m *MockKafkaReader) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		return fmt.Errorf("mock error: %s", m.failureMsg)
	}

	m.closed = true
	return nil
}

// Stats returns mock statistics
func (m *MockKafkaReader) Stats() kafka.ReaderStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// Config returns mock configuration
func (m *MockKafkaReader) Config() kafka.ReaderConfig {
	return kafka.ReaderConfig{
		Brokers: []string{"mock-broker:9092"},
		Topic:   "mock-topic",
		GroupID: "mock-group",
	}
}

// Reset clears all messages and resets the mock state
func (m *MockKafkaReader) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = make([]kafka.Message, 0)
	m.currentIndex = 0
	m.shouldFail = false
	m.failureMsg = ""
	m.closed = false
	m.readDelay = 0
	m.contextCanceled = false
	m.stats = kafka.ReaderStats{}
}

// SetReadDelay sets a delay for read operations (for testing timing)
func (m *MockKafkaReader) SetReadDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readDelay = delay
}

// GetMessagesCount returns the number of messages in the queue
func (m *MockKafkaReader) GetMessagesCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.messages)
}

// GetCurrentIndex returns the current read index
func (m *MockKafkaReader) GetCurrentIndex() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentIndex
}

// IsClosed returns whether the reader is closed
func (m *MockKafkaReader) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

// IsContextCanceled returns whether context was canceled during read
func (m *MockKafkaReader) IsContextCanceled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.contextCanceled
}

// CreateTestMessage creates a test Kafka message
func CreateTestMessage(partition int, offset int64, key, value string) kafka.Message {
	return kafka.Message{
		Topic:     "test-topic",
		Partition: partition,
		Offset:    offset,
		Key:       []byte(key),
		Value:     []byte(value),
		Time:      time.Now(),
	}
}

// CreateTestMessages creates multiple test messages
func CreateTestMessages(count int) []kafka.Message {
	messages := make([]kafka.Message, count)
	for i := 0; i < count; i++ {
		messages[i] = CreateTestMessage(0, int64(i), fmt.Sprintf("key-%d", i), fmt.Sprintf(`{"test": "message-%d"}`, i))
	}
	return messages
}
