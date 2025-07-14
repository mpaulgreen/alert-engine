package models

import (
	"time"
)

// LogEntry represents a structured log entry from OpenShift applications
type LogEntry struct {
	Timestamp     time.Time      `json:"timestamp"`
	AtTimestamp   time.Time      `json:"@timestamp"`
	Level         string         `json:"level"`
	Message       string         `json:"message"`
	Kubernetes    KubernetesInfo `json:"kubernetes"`
	NamespaceName string         `json:"namespace_name"` // Top-level namespace field
	Host          string         `json:"host"`
	Hostname      string         `json:"hostname"`
	LogSource     string         `json:"log_source"`
	LogType       string         `json:"log_type"`
	Raw           string         `json:"raw,omitempty"`
}

// KubernetesInfo contains Kubernetes-specific metadata from OpenShift logs
type KubernetesInfo struct {
	Namespace     string            `json:"namespace"`      // Keep for backward compatibility
	NamespaceName string            `json:"namespace_name"` // Actual field from OpenShift logs
	Pod           string            `json:"pod"`            // Keep for backward compatibility
	PodName       string            `json:"pod_name"`       // Actual field from OpenShift logs
	Container     string            `json:"container"`      // Keep for backward compatibility
	ContainerName string            `json:"container_name"` // Actual field from OpenShift logs
	Labels        map[string]string `json:"labels"`
	Annotations   map[string]string `json:"annotations"`
	ContainerID   string            `json:"container_id"`
	PodIP         string            `json:"pod_ip"`
	PodOwner      string            `json:"pod_owner"`
}

// GetNamespace returns the namespace from the log entry, checking multiple possible fields
func (le *LogEntry) GetNamespace() string {
	// Check top-level namespace_name first
	if le.NamespaceName != "" {
		return le.NamespaceName
	}

	// Check kubernetes.namespace_name
	if le.Kubernetes.NamespaceName != "" {
		return le.Kubernetes.NamespaceName
	}

	// Fallback to kubernetes.namespace for backward compatibility
	if le.Kubernetes.Namespace != "" {
		return le.Kubernetes.Namespace
	}

	return ""
}

// GetPodName returns the pod name from the log entry, checking multiple possible fields
func (le *LogEntry) GetPodName() string {
	if le.Kubernetes.PodName != "" {
		return le.Kubernetes.PodName
	}

	return le.Kubernetes.Pod
}

// GetContainerName returns the container name from the log entry, checking multiple possible fields
func (le *LogEntry) GetContainerName() string {
	if le.Kubernetes.ContainerName != "" {
		return le.Kubernetes.ContainerName
	}

	return le.Kubernetes.Container
}

// LogFilter represents filtering criteria for log queries
type LogFilter struct {
	Namespace string    `json:"namespace,omitempty"`
	Service   string    `json:"service,omitempty"`
	LogLevel  string    `json:"log_level,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Keywords  []string  `json:"keywords,omitempty"`
	Limit     int       `json:"limit,omitempty"`
}

// LogStats represents statistics about log processing
type LogStats struct {
	TotalLogs     int64            `json:"total_logs"`
	LogsByLevel   map[string]int64 `json:"logs_by_level"`
	LogsByService map[string]int64 `json:"logs_by_service"`
	LastUpdated   time.Time        `json:"last_updated"`
}

// TimeWindow represents a time window for aggregation
type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Count int       `json:"count"`
}
