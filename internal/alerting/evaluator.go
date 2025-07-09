package alerting

import (
    "context"
    "fmt"
    "log"
    "strings"
    "time"

    "github.com/log-monitoring/alert-engine/pkg/models"
)

// Evaluator handles the evaluation of log entries against alert rules
type Evaluator struct {
    stateStore StateStore
}

// NewEvaluator creates a new evaluator instance
func NewEvaluator(stateStore StateStore) *Evaluator {
    return &Evaluator{
        stateStore: stateStore,
    }
}

// EvaluateCondition evaluates a single condition against a log entry
func (e *Evaluator) EvaluateCondition(logEntry models.LogEntry, condition models.AlertConditions) (bool, error) {
    // Check log level match
    if condition.LogLevel != "" && logEntry.Level != condition.LogLevel {
        return false, nil
    }

    // Check namespace match
    if condition.Namespace != "" && logEntry.Kubernetes.Namespace != condition.Namespace {
        return false, nil
    }

    // Check service match (from app label)
    if condition.Service != "" {
        if appLabel, exists := logEntry.Kubernetes.Labels["app"]; !exists || appLabel != condition.Service {
            return false, nil
        }
    }

    // Check keyword matches
    if len(condition.Keywords) > 0 {
        if !e.matchesKeywords(logEntry.Message, condition.Keywords) {
            return false, nil
        }
    }

    return true, nil
}

// EvaluateThreshold evaluates if the current count meets the threshold criteria
func (e *Evaluator) EvaluateThreshold(ruleID string, condition models.AlertConditions, timestamp time.Time) (bool, int64, error) {
    // Get current count for the time window
    count, err := e.stateStore.IncrementCounter(ruleID, condition.TimeWindow)
    if err != nil {
        return false, 0, fmt.Errorf("failed to increment counter: %w", err)
    }

    // Check if threshold is met
    thresholdMet := e.compareWithThreshold(count, condition.Threshold, condition.Operator)
    
    return thresholdMet, count, nil
}

// TestRule tests an alert rule against sample log entries
func (e *Evaluator) TestRule(rule models.AlertRule, sampleLogs []models.LogEntry) (*RuleTestResult, error) {
    result := &RuleTestResult{
        RuleID:     rule.ID,
        RuleName:   rule.Name,
        TestTime:   time.Now(),
        Matches:    []LogMatch{},
        Summary:    TestSummary{},
    }

    matchCount := 0
    
    for _, logEntry := range sampleLogs {
        matches, err := e.EvaluateCondition(logEntry, rule.Conditions)
        if err != nil {
            return nil, fmt.Errorf("error evaluating condition: %w", err)
        }

        if matches {
            matchCount++
            result.Matches = append(result.Matches, LogMatch{
                LogEntry:  logEntry,
                Matched:   true,
                Timestamp: logEntry.Timestamp,
            })
        }
    }

    result.Summary = TestSummary{
        TotalLogs:    len(sampleLogs),
        MatchedLogs:  matchCount,
        MatchRate:    float64(matchCount) / float64(len(sampleLogs)),
        WouldTrigger: matchCount >= rule.Conditions.Threshold,
    }

    return result, nil
}

// compareWithThreshold compares count with threshold based on operator
func (e *Evaluator) compareWithThreshold(count int64, threshold int, operator string) bool {
    switch operator {
    case "gt", "":
        return count > int64(threshold)
    case "gte":
        return count >= int64(threshold)
    case "lt":
        return count < int64(threshold)
    case "lte":
        return count <= int64(threshold)
    case "eq":
        return count == int64(threshold)
    default:
        return count > int64(threshold)
    }
}

// matchesKeywords checks if the message contains all required keywords
func (e *Evaluator) matchesKeywords(message string, keywords []string) bool {
    messageUpper := strings.ToUpper(message)
    
    for _, keyword := range keywords {
        if !strings.Contains(messageUpper, strings.ToUpper(keyword)) {
            return false
        }
    }
    
    return true
}

// RuleTestResult represents the result of testing an alert rule
type RuleTestResult struct {
    RuleID     string      `json:"rule_id"`
    RuleName   string      `json:"rule_name"`
    TestTime   time.Time   `json:"test_time"`
    Matches    []LogMatch  `json:"matches"`
    Summary    TestSummary `json:"summary"`
}

// LogMatch represents a log entry that matched a rule
type LogMatch struct {
    LogEntry  models.LogEntry `json:"log_entry"`
    Matched   bool            `json:"matched"`
    Timestamp time.Time       `json:"timestamp"`
}

// TestSummary provides a summary of the test results
type TestSummary struct {
    TotalLogs    int     `json:"total_logs"`
    MatchedLogs  int     `json:"matched_logs"`
    MatchRate    float64 `json:"match_rate"`
    WouldTrigger bool    `json:"would_trigger"`
}

// RulePerformance tracks performance metrics for rule evaluation
type RulePerformance struct {
    RuleID           string        `json:"rule_id"`
    EvaluationCount  int64         `json:"evaluation_count"`
    MatchCount       int64         `json:"match_count"`
    LastEvaluation   time.Time     `json:"last_evaluation"`
    AverageEvalTime  time.Duration `json:"average_eval_time"`
    TotalEvalTime    time.Duration `json:"total_eval_time"`
}

// PerformanceTracker tracks performance metrics for rule evaluations
type PerformanceTracker struct {
    metrics map[string]*RulePerformance
}

// NewPerformanceTracker creates a new performance tracker
func NewPerformanceTracker() *PerformanceTracker {
    return &PerformanceTracker{
        metrics: make(map[string]*RulePerformance),
    }
}

// TrackEvaluation tracks the evaluation of a rule
func (pt *PerformanceTracker) TrackEvaluation(ruleID string, evalTime time.Duration, matched bool) {
    if pt.metrics[ruleID] == nil {
        pt.metrics[ruleID] = &RulePerformance{
            RuleID: ruleID,
        }
    }

    metric := pt.metrics[ruleID]
    metric.EvaluationCount++
    metric.LastEvaluation = time.Now()
    metric.TotalEvalTime += evalTime
    metric.AverageEvalTime = metric.TotalEvalTime / time.Duration(metric.EvaluationCount)

    if matched {
        metric.MatchCount++
    }
}

// GetPerformanceMetrics returns performance metrics for all rules
func (pt *PerformanceTracker) GetPerformanceMetrics() map[string]*RulePerformance {
    return pt.metrics
}

// GetRulePerformance returns performance metrics for a specific rule
func (pt *PerformanceTracker) GetRulePerformance(ruleID string) *RulePerformance {
    return pt.metrics[ruleID]
}

// BatchEvaluator handles batch evaluation of log entries
type BatchEvaluator struct {
    evaluator *Evaluator
    batchSize int
}

// NewBatchEvaluator creates a new batch evaluator
func NewBatchEvaluator(stateStore StateStore, batchSize int) *BatchEvaluator {
    return &BatchEvaluator{
        evaluator: NewEvaluator(stateStore),
        batchSize: batchSize,
    }
}

// EvaluateBatch evaluates a batch of log entries against all rules
func (be *BatchEvaluator) EvaluateBatch(ctx context.Context, logs []models.LogEntry, rules []models.AlertRule) ([]models.Alert, error) {
    var alerts []models.Alert
    
    // Process logs in batches
    for i := 0; i < len(logs); i += be.batchSize {
        end := i + be.batchSize
        if end > len(logs) {
            end = len(logs)
        }
        
        batch := logs[i:end]
        batchAlerts, err := be.evaluateBatch(ctx, batch, rules)
        if err != nil {
            return nil, fmt.Errorf("error evaluating batch: %w", err)
        }
        
        alerts = append(alerts, batchAlerts...)
        
        // Check for context cancellation
        select {
        case <-ctx.Done():
            return alerts, ctx.Err()
        default:
        }
    }
    
    return alerts, nil
}

// evaluateBatch evaluates a single batch of logs
func (be *BatchEvaluator) evaluateBatch(ctx context.Context, logs []models.LogEntry, rules []models.AlertRule) ([]models.Alert, error) {
    var alerts []models.Alert
    
    for _, logEntry := range logs {
        for _, rule := range rules {
            if !rule.Enabled {
                continue
            }
            
            matches, err := be.evaluator.EvaluateCondition(logEntry, rule.Conditions)
            if err != nil {
                log.Printf("Error evaluating condition for rule %s: %v", rule.ID, err)
                continue
            }
            
            if matches {
                thresholdMet, count, err := be.evaluator.EvaluateThreshold(rule.ID, rule.Conditions, logEntry.Timestamp)
                if err != nil {
                    log.Printf("Error evaluating threshold for rule %s: %v", rule.ID, err)
                    continue
                }
                
                if thresholdMet {
                    alert := models.Alert{
                        ID:        fmt.Sprintf("%s-%d", rule.ID, time.Now().Unix()),
                        RuleID:    rule.ID,
                        RuleName:  rule.Name,
                        LogEntry:  logEntry,
                        Timestamp: logEntry.Timestamp,
                        Severity:  rule.Actions.Severity,
                        Status:    "pending",
                        Count:     int(count),
                    }
                    
                    alerts = append(alerts, alert)
                }
            }
        }
    }
    
    return alerts, nil
} 