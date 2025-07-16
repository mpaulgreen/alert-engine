//go:build unit

package models

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test the helper methods for actual code coverage
func TestLogEntry_GetNamespace(t *testing.T) {
	t.Run("returns top-level namespace", func(t *testing.T) {
		logEntry := LogEntry{
			Namespace: "production",
			Kubernetes: KubernetesInfo{
				Namespace:     "staging",
				NamespaceName: "development",
			},
		}
		assert.Equal(t, "production", logEntry.GetNamespace())
	})

	t.Run("returns namespace_name when namespace is empty", func(t *testing.T) {
		logEntry := LogEntry{
			Namespace:     "",
			NamespaceName: "production-v2",
			Kubernetes: KubernetesInfo{
				Namespace:     "staging",
				NamespaceName: "development",
			},
		}
		assert.Equal(t, "production-v2", logEntry.GetNamespace())
	})

	t.Run("returns kubernetes namespace_name when top-level fields are empty", func(t *testing.T) {
		logEntry := LogEntry{
			Namespace:     "",
			NamespaceName: "",
			Kubernetes: KubernetesInfo{
				Namespace:     "staging",
				NamespaceName: "development",
			},
		}
		assert.Equal(t, "development", logEntry.GetNamespace())
	})

	t.Run("returns kubernetes namespace as fallback", func(t *testing.T) {
		logEntry := LogEntry{
			Namespace:     "",
			NamespaceName: "",
			Kubernetes: KubernetesInfo{
				Namespace:     "staging",
				NamespaceName: "",
			},
		}
		assert.Equal(t, "staging", logEntry.GetNamespace())
	})

	t.Run("returns empty string when no namespace found", func(t *testing.T) {
		logEntry := LogEntry{
			Namespace:     "",
			NamespaceName: "",
			Kubernetes: KubernetesInfo{
				Namespace:     "",
				NamespaceName: "",
			},
		}
		assert.Equal(t, "", logEntry.GetNamespace())
	})
}

func TestLogEntry_GetPodName(t *testing.T) {
	t.Run("returns pod_name when available", func(t *testing.T) {
		logEntry := LogEntry{
			Kubernetes: KubernetesInfo{
				Pod:     "legacy-pod-name",
				PodName: "new-pod-name-abc123",
			},
		}
		assert.Equal(t, "new-pod-name-abc123", logEntry.GetPodName())
	})

	t.Run("returns pod as fallback when pod_name is empty", func(t *testing.T) {
		logEntry := LogEntry{
			Kubernetes: KubernetesInfo{
				Pod:     "legacy-pod-name",
				PodName: "",
			},
		}
		assert.Equal(t, "legacy-pod-name", logEntry.GetPodName())
	})

	t.Run("returns empty string when both are empty", func(t *testing.T) {
		logEntry := LogEntry{
			Kubernetes: KubernetesInfo{
				Pod:     "",
				PodName: "",
			},
		}
		assert.Equal(t, "", logEntry.GetPodName())
	})
}

func TestLogEntry_GetContainerName(t *testing.T) {
	t.Run("returns container_name when available", func(t *testing.T) {
		logEntry := LogEntry{
			Kubernetes: KubernetesInfo{
				Container:     "legacy-container",
				ContainerName: "new-container-name",
			},
		}
		assert.Equal(t, "new-container-name", logEntry.GetContainerName())
	})

	t.Run("returns container as fallback when container_name is empty", func(t *testing.T) {
		logEntry := LogEntry{
			Kubernetes: KubernetesInfo{
				Container:     "legacy-container",
				ContainerName: "",
			},
		}
		assert.Equal(t, "legacy-container", logEntry.GetContainerName())
	})

	t.Run("returns empty string when both are empty", func(t *testing.T) {
		logEntry := LogEntry{
			Kubernetes: KubernetesInfo{
				Container:     "",
				ContainerName: "",
			},
		}
		assert.Equal(t, "", logEntry.GetContainerName())
	})
}

func TestLogEntry_JSONMarshaling(t *testing.T) {
	t.Run("successful marshal and unmarshal", func(t *testing.T) {
		original := LogEntry{
			Timestamp: time.Date(2024, 1, 15, 10, 30, 45, 123000000, time.UTC),
			Level:     "ERROR",
			Message:   "Database connection failed: timeout after 30s",
			Kubernetes: KubernetesInfo{
				Namespace: "production",
				Pod:       "user-service-7b6f9c8d4-xyz12",
				Container: "user-service",
				Labels: map[string]string{
					"app":         "user-service",
					"version":     "1.2.3",
					"environment": "production",
				},
			},
			Host: "worker-node-01",
			Raw:  `{"timestamp":"2024-01-15T10:30:45.123Z","level":"ERROR","message":"Database connection failed"}`,
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Unmarshal from JSON
		var restored LogEntry
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		// Verify all fields are correctly preserved
		assert.True(t, original.Timestamp.Equal(restored.Timestamp))
		assert.Equal(t, original.Level, restored.Level)
		assert.Equal(t, original.Message, restored.Message)
		assert.Equal(t, original.Kubernetes.Namespace, restored.Kubernetes.Namespace)
		assert.Equal(t, original.Kubernetes.Pod, restored.Kubernetes.Pod)
		assert.Equal(t, original.Kubernetes.Container, restored.Kubernetes.Container)
		assert.Equal(t, original.Kubernetes.Labels, restored.Kubernetes.Labels)
		assert.Equal(t, original.Host, restored.Host)
		assert.Equal(t, original.Raw, restored.Raw)
	})

	t.Run("unmarshal with missing optional fields", func(t *testing.T) {
		jsonData := `{
			"timestamp": "2024-01-15T10:30:45.123Z",
			"level": "INFO",
			"message": "Service started successfully",
			"kubernetes": {
				"namespace": "production",
				"pod": "user-service-abc123",
				"container": "user-service",
				"labels": {
					"app": "user-service"
				}
			}
		}`

		var logEntry LogEntry
		err := json.Unmarshal([]byte(jsonData), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "INFO", logEntry.Level)
		assert.Equal(t, "Service started successfully", logEntry.Message)
		assert.Equal(t, "production", logEntry.Kubernetes.Namespace)
		assert.Equal(t, "user-service-abc123", logEntry.Kubernetes.Pod)
		assert.Equal(t, "user-service", logEntry.Kubernetes.Container)
		assert.Equal(t, map[string]string{"app": "user-service"}, logEntry.Kubernetes.Labels)
		assert.Equal(t, "", logEntry.Host) // Missing field should be empty
		assert.Equal(t, "", logEntry.Raw)  // Missing field should be empty
	})

	t.Run("unmarshal with empty kubernetes info", func(t *testing.T) {
		jsonData := `{
			"timestamp": "2024-01-15T10:30:45.123Z",
			"level": "DEBUG",
			"message": "Debug message",
			"kubernetes": {
				"namespace": "",
				"pod": "",
				"container": "",
				"labels": {}
			},
			"host": "localhost"
		}`

		var logEntry LogEntry
		err := json.Unmarshal([]byte(jsonData), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "DEBUG", logEntry.Level)
		assert.Equal(t, "Debug message", logEntry.Message)
		assert.Equal(t, "", logEntry.Kubernetes.Namespace)
		assert.Equal(t, "", logEntry.Kubernetes.Pod)
		assert.Equal(t, "", logEntry.Kubernetes.Container)
		assert.Empty(t, logEntry.Kubernetes.Labels)
		assert.Equal(t, "localhost", logEntry.Host)
	})

	t.Run("unmarshal with invalid timestamp", func(t *testing.T) {
		jsonData := `{
			"timestamp": "invalid-timestamp",
			"level": "INFO",
			"message": "Test message"
		}`

		var logEntry LogEntry
		err := json.Unmarshal([]byte(jsonData), &logEntry)
		assert.Error(t, err)
	})
}

func TestKubernetesInfo_JSONMarshaling(t *testing.T) {
	t.Run("successful marshal and unmarshal", func(t *testing.T) {
		original := KubernetesInfo{
			Namespace: "production",
			Pod:       "user-service-7b6f9c8d4-xyz12",
			Container: "user-service",
			Labels: map[string]string{
				"app":         "user-service",
				"version":     "1.2.3",
				"environment": "production",
				"team":        "backend",
			},
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Unmarshal from JSON
		var restored KubernetesInfo
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		assert.Equal(t, original.Namespace, restored.Namespace)
		assert.Equal(t, original.Pod, restored.Pod)
		assert.Equal(t, original.Container, restored.Container)
		assert.Equal(t, original.Labels, restored.Labels)
	})

	t.Run("unmarshal with nil labels", func(t *testing.T) {
		jsonData := `{
			"namespace": "production",
			"pod": "test-pod",
			"container": "test-container",
			"labels": null
		}`

		var kubernetesInfo KubernetesInfo
		err := json.Unmarshal([]byte(jsonData), &kubernetesInfo)
		require.NoError(t, err)

		assert.Equal(t, "production", kubernetesInfo.Namespace)
		assert.Equal(t, "test-pod", kubernetesInfo.Pod)
		assert.Equal(t, "test-container", kubernetesInfo.Container)
		assert.Nil(t, kubernetesInfo.Labels)
	})

	t.Run("unmarshal with empty labels object", func(t *testing.T) {
		jsonData := `{
			"namespace": "production",
			"pod": "test-pod",
			"container": "test-container",
			"labels": {}
		}`

		var kubernetesInfo KubernetesInfo
		err := json.Unmarshal([]byte(jsonData), &kubernetesInfo)
		require.NoError(t, err)

		assert.Equal(t, "production", kubernetesInfo.Namespace)
		assert.Equal(t, "test-pod", kubernetesInfo.Pod)
		assert.Equal(t, "test-container", kubernetesInfo.Container)
		assert.Empty(t, kubernetesInfo.Labels)
		assert.NotNil(t, kubernetesInfo.Labels)
	})
}

func TestLogFilter_JSONMarshaling(t *testing.T) {
	t.Run("successful marshal and unmarshal", func(t *testing.T) {
		original := LogFilter{
			Namespace: "production",
			Service:   "user-service",
			LogLevel:  "ERROR",
			StartTime: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			Keywords:  []string{"error", "failed", "timeout"},
			Limit:     100,
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Unmarshal from JSON
		var restored LogFilter
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		assert.Equal(t, original.Namespace, restored.Namespace)
		assert.Equal(t, original.Service, restored.Service)
		assert.Equal(t, original.LogLevel, restored.LogLevel)
		assert.True(t, original.StartTime.Equal(restored.StartTime))
		assert.True(t, original.EndTime.Equal(restored.EndTime))
		assert.Equal(t, original.Keywords, restored.Keywords)
		assert.Equal(t, original.Limit, restored.Limit)
	})

	t.Run("unmarshal with omitted fields", func(t *testing.T) {
		jsonData := `{
			"namespace": "production",
			"log_level": "ERROR"
		}`

		var filter LogFilter
		err := json.Unmarshal([]byte(jsonData), &filter)
		require.NoError(t, err)

		assert.Equal(t, "production", filter.Namespace)
		assert.Equal(t, "ERROR", filter.LogLevel)
		assert.Equal(t, "", filter.Service)
		assert.True(t, filter.StartTime.IsZero())
		assert.True(t, filter.EndTime.IsZero())
		assert.Empty(t, filter.Keywords)
		assert.Equal(t, 0, filter.Limit)
	})

	t.Run("unmarshal with empty keywords array", func(t *testing.T) {
		jsonData := `{
			"namespace": "production",
			"keywords": []
		}`

		var filter LogFilter
		err := json.Unmarshal([]byte(jsonData), &filter)
		require.NoError(t, err)

		assert.Equal(t, "production", filter.Namespace)
		assert.Empty(t, filter.Keywords)
		assert.NotNil(t, filter.Keywords)
	})
}

func TestLogStats_JSONMarshaling(t *testing.T) {
	t.Run("successful marshal and unmarshal", func(t *testing.T) {
		original := LogStats{
			TotalLogs: 1000,
			LogsByLevel: map[string]int64{
				"DEBUG": 300,
				"INFO":  500,
				"WARN":  150,
				"ERROR": 45,
				"FATAL": 5,
			},
			LogsByService: map[string]int64{
				"user-service":    400,
				"payment-service": 300,
				"order-service":   300,
			},
			LastUpdated: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Unmarshal from JSON
		var restored LogStats
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		assert.Equal(t, original.TotalLogs, restored.TotalLogs)
		assert.Equal(t, original.LogsByLevel, restored.LogsByLevel)
		assert.Equal(t, original.LogsByService, restored.LogsByService)
		assert.True(t, original.LastUpdated.Equal(restored.LastUpdated))
	})

	t.Run("unmarshal with empty maps", func(t *testing.T) {
		jsonData := `{
			"total_logs": 0,
			"logs_by_level": {},
			"logs_by_service": {},
			"last_updated": "2024-01-15T10:30:00Z"
		}`

		var stats LogStats
		err := json.Unmarshal([]byte(jsonData), &stats)
		require.NoError(t, err)

		assert.Equal(t, int64(0), stats.TotalLogs)
		assert.Empty(t, stats.LogsByLevel)
		assert.Empty(t, stats.LogsByService)
		assert.NotNil(t, stats.LogsByLevel)
		assert.NotNil(t, stats.LogsByService)
	})

	t.Run("unmarshal with null maps", func(t *testing.T) {
		jsonData := `{
			"total_logs": 100,
			"logs_by_level": null,
			"logs_by_service": null,
			"last_updated": "2024-01-15T10:30:00Z"
		}`

		var stats LogStats
		err := json.Unmarshal([]byte(jsonData), &stats)
		require.NoError(t, err)

		assert.Equal(t, int64(100), stats.TotalLogs)
		assert.Nil(t, stats.LogsByLevel)
		assert.Nil(t, stats.LogsByService)
	})
}

func TestTimeWindow_JSONMarshaling(t *testing.T) {
	t.Run("successful marshal and unmarshal", func(t *testing.T) {
		original := TimeWindow{
			Start: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			Count: 42,
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Unmarshal from JSON
		var restored TimeWindow
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		assert.True(t, original.Start.Equal(restored.Start))
		assert.True(t, original.End.Equal(restored.End))
		assert.Equal(t, original.Count, restored.Count)
	})

	t.Run("unmarshal with zero times", func(t *testing.T) {
		jsonData := `{
			"start": "0001-01-01T00:00:00Z",
			"end": "0001-01-01T00:00:00Z",
			"count": 0
		}`

		var timeWindow TimeWindow
		err := json.Unmarshal([]byte(jsonData), &timeWindow)
		require.NoError(t, err)

		assert.True(t, timeWindow.Start.IsZero())
		assert.True(t, timeWindow.End.IsZero())
		assert.Equal(t, 0, timeWindow.Count)
	})
}

func TestLogLevel_Validation(t *testing.T) {
	t.Run("valid log levels", func(t *testing.T) {
		validLevels := []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}
		for _, level := range validLevels {
			logEntry := LogEntry{
				Timestamp: time.Now(),
				Level:     level,
				Message:   "Test message",
				Kubernetes: KubernetesInfo{
					Namespace: "test",
					Pod:       "test-pod",
					Container: "test-container",
					Labels:    map[string]string{"app": "test"},
				},
				Host: "test-host",
			}

			// Test that it can be marshaled/unmarshaled without error
			data, err := json.Marshal(logEntry)
			require.NoError(t, err)

			var restored LogEntry
			err = json.Unmarshal(data, &restored)
			require.NoError(t, err)
			assert.Equal(t, level, restored.Level)
		}
	})

	t.Run("case sensitivity", func(t *testing.T) {
		testCases := []struct {
			name     string
			level    string
			expected string
		}{
			{"lowercase debug", "debug", "debug"},
			{"mixed case warn", "WaRn", "WaRn"},
			{"uppercase error", "ERROR", "ERROR"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				logEntry := LogEntry{
					Timestamp: time.Now(),
					Level:     tc.level,
					Message:   "Test message",
					Kubernetes: KubernetesInfo{
						Namespace: "test",
						Pod:       "test-pod",
						Container: "test-container",
						Labels:    map[string]string{"app": "test"},
					},
					Host: "test-host",
				}

				data, err := json.Marshal(logEntry)
				require.NoError(t, err)

				var restored LogEntry
				err = json.Unmarshal(data, &restored)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, restored.Level)
			})
		}
	})
}

func TestTimestamp_Formatting(t *testing.T) {
	t.Run("RFC3339 timestamp formatting", func(t *testing.T) {
		timestamp := time.Date(2024, 1, 15, 10, 30, 45, 123456789, time.UTC)
		logEntry := LogEntry{
			Timestamp: timestamp,
			Level:     "INFO",
			Message:   "Test message",
			Kubernetes: KubernetesInfo{
				Namespace: "test",
				Pod:       "test-pod",
				Container: "test-container",
				Labels:    map[string]string{"app": "test"},
			},
			Host: "test-host",
		}

		data, err := json.Marshal(logEntry)
		require.NoError(t, err)

		// Check that the timestamp is formatted correctly in JSON
		assert.Contains(t, string(data), `"timestamp":"2024-01-15T10:30:45.123456789Z"`)

		var restored LogEntry
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)
		assert.True(t, timestamp.Equal(restored.Timestamp))
	})

	t.Run("different timezone handling", func(t *testing.T) {
		// Test with different timezone
		location, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)

		timestamp := time.Date(2024, 1, 15, 10, 30, 45, 0, location)
		logEntry := LogEntry{
			Timestamp: timestamp,
			Level:     "INFO",
			Message:   "Test message",
			Kubernetes: KubernetesInfo{
				Namespace: "test",
				Pod:       "test-pod",
				Container: "test-container",
				Labels:    map[string]string{"app": "test"},
			},
			Host: "test-host",
		}

		data, err := json.Marshal(logEntry)
		require.NoError(t, err)

		var restored LogEntry
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		// Timestamps should be equal when comparing in UTC
		assert.True(t, timestamp.Equal(restored.Timestamp))
	})
}

func TestLogEdgeCases(t *testing.T) {
	t.Run("empty log entry", func(t *testing.T) {
		empty := LogEntry{}

		data, err := json.Marshal(empty)
		require.NoError(t, err)

		var restored LogEntry
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		assert.True(t, restored.Timestamp.IsZero())
		assert.Equal(t, "", restored.Level)
		assert.Equal(t, "", restored.Message)
		assert.Equal(t, "", restored.Kubernetes.Namespace)
		assert.Equal(t, "", restored.Kubernetes.Pod)
		assert.Equal(t, "", restored.Kubernetes.Container)
		assert.Nil(t, restored.Kubernetes.Labels)
		assert.Equal(t, "", restored.Host)
		assert.Equal(t, "", restored.Raw)
	})

	t.Run("very long message", func(t *testing.T) {
		longMessage := string(make([]byte, 10000))
		for i := range longMessage {
			longMessage = longMessage[:i] + "a" + longMessage[i+1:]
		}

		logEntry := LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   longMessage,
			Kubernetes: KubernetesInfo{
				Namespace: "test",
				Pod:       "test-pod",
				Container: "test-container",
				Labels:    map[string]string{"app": "test"},
			},
			Host: "test-host",
		}

		data, err := json.Marshal(logEntry)
		require.NoError(t, err)

		var restored LogEntry
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)
		assert.Equal(t, longMessage, restored.Message)
	})

	t.Run("special characters in message", func(t *testing.T) {
		specialMessage := "Test message with special chars: \n\t\r\"'\\\\/ and unicode: 🚀 中文"
		logEntry := LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   specialMessage,
			Kubernetes: KubernetesInfo{
				Namespace: "test",
				Pod:       "test-pod",
				Container: "test-container",
				Labels:    map[string]string{"app": "test"},
			},
			Host: "test-host",
		}

		data, err := json.Marshal(logEntry)
		require.NoError(t, err)

		var restored LogEntry
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)
		assert.Equal(t, specialMessage, restored.Message)
	})

	t.Run("large number of labels", func(t *testing.T) {
		labels := make(map[string]string)
		for i := 0; i < 100; i++ {
			labels[fmt.Sprintf("label%d", i)] = fmt.Sprintf("value%d", i)
		}

		kubernetesInfo := KubernetesInfo{
			Namespace: "test",
			Pod:       "test-pod",
			Container: "test-container",
			Labels:    labels,
		}

		data, err := json.Marshal(kubernetesInfo)
		require.NoError(t, err)

		var restored KubernetesInfo
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)
		assert.Equal(t, labels, restored.Labels)
		assert.Equal(t, 100, len(restored.Labels))
	})
}

func TestConcurrentAccess(t *testing.T) {
	t.Run("concurrent marshaling", func(t *testing.T) {
		logEntry := LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   "Test concurrent access",
			Kubernetes: KubernetesInfo{
				Namespace: "test",
				Pod:       "test-pod",
				Container: "test-container",
				Labels:    map[string]string{"app": "test"},
			},
			Host: "test-host",
		}

		const numGoroutines = 10
		done := make(chan bool, numGoroutines)
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer func() { done <- true }()

				data, err := json.Marshal(logEntry)
				if err != nil {
					errors <- err
					return
				}

				var restored LogEntry
				err = json.Unmarshal(data, &restored)
				if err != nil {
					errors <- err
					return
				}

				if restored.Message != logEntry.Message {
					errors <- fmt.Errorf("message mismatch: expected %s, got %s", logEntry.Message, restored.Message)
				}
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Check for errors
		select {
		case err := <-errors:
			t.Fatalf("Concurrent access failed: %v", err)
		default:
			// No errors
		}
	})
}
