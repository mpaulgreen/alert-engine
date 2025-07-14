# 50 Natural Language Alert Patterns for Log Monitoring

## **Category 1: Basic Threshold Alerts (Count-based)**

### **Error Rate Patterns**
1. "Page me if there are more than 10 ERROR logs from the payment service in 5 minutes"
2. "Alert when user-service shows over 50 WARN messages per hour"
3. "Notify if database service has any FATAL errors in the last 10 minutes"
4. "Send critical alert if authentication-api exceeds 25 ERROR logs in 2 minutes"
5. "Page on-call if more than 100 4xx responses from api-gateway in 15 minutes"

### **Service-Specific Patterns**
6. "Alert if checkout-service logs contain more than 5 'payment failed' messages in 10 minutes"
7. "Page me when inventory-service shows over 20 'stock unavailable' errors per hour"
8. "Notify team if email-service has more than 3 'SMTP connection failed' in 5 minutes"
9. "Critical page if redis-cache service logs 'connection refused' more than 2 times in 1 minute"
10. "Alert when message-queue shows over 15 'queue full' warnings in 30 minutes"

## **Category 2: Content Pattern Matching**

### **Keyword and Phrase Detection**
11. "Page me if any pod shows 'OutOfMemory' errors during business hours (9 AM - 6 PM)"
12. "Alert on-call if database connection failures spike above normal baseline"
13. "Notify team lead if authentication service shows 'token expired' more than 50 times in 1 hour"
14. "Critical alert if any service logs 'security breach' or 'unauthorized access'"
15. "Page if SSL certificate errors appear in any ingress controller logs"

### **Performance-Related Content**
16. "Alert when any service logs 'timeout' more than 10 times in 5 minutes"
17. "Page me if 'slow query' appears in database logs more than 5 times per hour"
18. "Notify if 'circuit breaker open' shows up in any microservice logs"
19. "Alert when 'rate limit exceeded' appears more than 20 times in 10 minutes"
20. "Critical page if 'deadlock detected' appears in any database service"

## **Category 3: Temporal and Conditional Logic**

### **Time-Based Conditions**
21. "Alert only during weekends if backup-service shows any ERROR logs"
22. "Page me between 2 AM - 4 AM if maintenance-job fails"
23. "Notify during business hours if customer-service response time exceeds 5 seconds"
24. "Alert outside business hours if security-scanner finds critical vulnerabilities"
25. "Page immediately if payment processing fails during Black Friday (Nov 24-25)"

### **Multi-Condition Logic**
26. "Critical page if database connection errors AND API response time spikes above 2 seconds"
27. "Alert if memory usage > 85% AND garbage collection logs show 'full GC' more than 3 times in 5 minutes"
28. "Page when disk space < 10% OR I/O errors appear in storage service logs"
29. "Notify if CPU usage > 90% AND application logs show 'thread pool exhausted'"
30. "Alert if both primary and secondary database show connection failures within 1 minute"

## **Category 4: Deployment and Change-Related**

### **Deployment Awareness**
31. "Skip alerts for payment-service during deployment windows (tagged with 'maintenance')"
32. "Alert only if ERROR logs appear 10 minutes after deployment completion"
33. "Page if rollback occurs AND new ERROR patterns emerge in any service"
34. "Notify if post-deployment health checks fail in any critical service"
35. "Critical alert if canary deployment shows 5x more errors than stable version"

### **Configuration Changes**
36. "Alert when configuration reload fails in any service"
37. "Page if 'config validation error' appears after any ConfigMap update"
38. "Notify if feature flag changes cause increase in error rates by >200%"
39. "Alert when environment variable changes trigger service restarts >3 times in 1 hour"

## **Category 5: Cross-Service and Correlation**

### **Service Dependency Patterns**
40. "Page if payment-service AND order-service show errors simultaneously"
41. "Alert when downstream service failures cascade to more than 2 dependent services"
42. "Notify if external API failures affect more than 3 internal microservices"
43. "Critical page if user-authentication fails AND affects checkout, profile, AND support services"

### **Business Impact Correlation**
44. "Alert if shopping cart abandonment logs increase by >50% during peak hours"
45. "Page when user registration failures spike during marketing campaign periods"
46. "Notify if search service errors correlate with >25% drop in product page views"

## **Category 6: Resource and Infrastructure**

### **Resource Exhaustion**
47. "Critical page if any pod shows 'OOMKilled' status in logs"
48. "Alert when persistent volume claims show 'disk full' errors"
49. "Page if container restart count exceeds 5 times in 10 minutes for any critical service"
50. "Notify if node resource pressure causes pod evictions in production namespace"

---

## **Pattern Templates for NLP Engine**

### **Template Structure Examples:**

```
BASIC_THRESHOLD:
"[ACTION] if [SERVICE] has [COMPARATOR] [NUMBER] [LOG_LEVEL] logs in [TIME_PERIOD]"

CONTENT_MATCHING:
"[ACTION] when [SERVICE] logs contain '[CONTENT_PATTERN]' [COMPARATOR] [NUMBER] times in [TIME_PERIOD]"

TEMPORAL_CONDITIONAL:
"[ACTION] [TIME_CONDITION] if [SERVICE] shows [CONDITION]"

MULTI_CONDITION:
"[ACTION] if [CONDITION_1] [LOGIC_OPERATOR] [CONDITION_2]"

SERVICE_CORRELATION:
"[ACTION] if [SERVICE_1] [CONDITION_1] AND [SERVICE_2] [CONDITION_2]"
```

### **Variable Extraction Patterns:**

**Actions:** `[alert, page, notify, critical page, send notification]`

**Services:** `[payment-service, user-service, database, api-gateway, {service-name}]`

**Comparators:** `[more than, over, exceeds, above, less than, below, exactly]`

**Numbers:** `[\d+]` (regex for any number)

**Log Levels:** `[ERROR, WARN, INFO, DEBUG, FATAL, CRITICAL]`

**Time Periods:** `[minutes, hours, seconds, days]` with number prefix

**Logic Operators:** `[AND, OR, BUT NOT]`

**Content Patterns:** `[quoted strings, regex patterns, keyword lists]`

**Time Conditions:** `[during business hours, between X-Y, on weekends, during {event}]`

---

## **Implementation Strategy for Pattern Recognition**

### **Phase 1: Template Matching**
- Use regex patterns to match basic structures
- Extract variables using named capture groups  
- Validate extracted parameters against known services/metrics

### **Phase 2: Enhanced Entity Recognition**
- Use spaCy NER for service names and metrics
- Build custom entity models for domain-specific terms
- Add fuzzy matching for service name variations

### **Phase 3: Semantic Understanding**
- Implement intent classification (threshold vs pattern vs correlation)
- Add context awareness (deployment state, time zones, business context)
- Support natural language variations and synonyms

### **Validation Framework:**
Each pattern should generate test cases like:
```json
{
  "input": "Page me if payment-service has more than 10 ERROR logs in 5 minutes",
  "expected_rule": {
    "service": "payment-service",
    "log_level": "ERROR", 
    "threshold": 10,
    "time_window": "5m",
    "action": "page",
    "severity": "high"
  }
}
```