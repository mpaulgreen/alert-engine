package models

import (
    "time"
)

// LogEntry represents a structured log entry from OpenShift applications
type LogEntry struct {
    Timestamp  time.Time      `json:"timestamp"`
    Level      string         `json:"level"`
    Message    string         `json:"message"`
    Kubernetes KubernetesInfo `json:"kubernetes"`
    Host       string         `json:"host"`
    Raw        string         `json:"raw,omitempty"`
}

// KubernetesInfo contains Kubernetes-specific metadata added by Vector
type KubernetesInfo struct {
    Namespace string            `json:"namespace"`
    Pod       string            `json:"pod"`
    Container string            `json:"container"`
    Labels    map[string]string `json:"labels"`
}

// LogFilter represents filtering criteria for log queries
type LogFilter struct {
    Namespace  string    `json:"namespace,omitempty"`
    Service    string    `json:"service,omitempty"`
    LogLevel   string    `json:"log_level,omitempty"`
    StartTime  time.Time `json:"start_time,omitempty"`
    EndTime    time.Time `json:"end_time,omitempty"`
    Keywords   []string  `json:"keywords,omitempty"`
    Limit      int       `json:"limit,omitempty"`
}

// LogStats represents statistics about log processing
type LogStats struct {
    TotalLogs     int64              `json:"total_logs"`
    LogsByLevel   map[string]int64   `json:"logs_by_level"`
    LogsByService map[string]int64   `json:"logs_by_service"`
    LastUpdated   time.Time          `json:"last_updated"`
}

// TimeWindow represents a time window for aggregation
type TimeWindow struct {
    Start time.Time `json:"start"`
    End   time.Time `json:"end"`
    Count int       `json:"count"`
} 