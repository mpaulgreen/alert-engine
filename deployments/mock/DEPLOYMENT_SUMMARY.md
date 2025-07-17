# MockLogGenerator OpenShift Deployment - Complete Package

## 📦 Package Contents

This directory contains a complete, production-ready deployment package for the MockLogGenerator on OpenShift, designed to provide continuous log generation for Alert Engine testing.

## 📁 File Structure

```
alert-engine/deployments/mock/
├── 🐳 Container Image Files
│   ├── Dockerfile                    # Multi-stage container build
│   ├── requirements.txt              # Python dependencies
│   └── mock_log_generator.py        # Enhanced application code
│
├── ⚙️ Kubernetes Manifests
│   ├── configmap.yaml               # Application configuration
│   ├── serviceaccount.yaml          # RBAC and permissions
│   ├── deployment.yaml              # Main workload definition
│   ├── networkpolicy.yaml           # Security policies
│   └── kustomization.yaml           # Kustomize configuration
│
├── 🔧 Automation Scripts
│   ├── build.sh                     # Container build automation
│   ├── deploy.sh                    # Complete deployment automation
│   └── DEPLOYMENT_SUMMARY.md        # This file
│
└── 📚 Documentation
    └── README.md                     # Comprehensive deployment guide
```

## 🚀 Quick Start

### Prerequisites
- OpenShift cluster access with cluster-admin privileges
- Alert Engine infrastructure deployed (Kafka, Redis, ClusterLogForwarder)
- Container registry access (Quay.io, Docker Hub, etc.)

### Deploy in 3 Steps

1. **Build & Push Container Image**:
   ```bash
   cd alert-engine/deployments/mock
   ./build.sh --push --registry your-registry.com/your-org
   ```
   
   > **Note**: The build script automatically uses `--platform linux/amd64` for compatibility with x86_64 OpenShift clusters.

2. **Update Image Reference**:
   ```bash
   # Edit deployment.yaml and update the image line:
   # image: your-registry.com/your-org/mock-log-generator:latest
   ```

3. **Deploy to OpenShift**:
   ```bash
   ./deploy.sh deploy --wait --follow-logs
   ```

### One-Command Deployment (Advanced)
```bash
# Complete deployment with custom registry
./deploy.sh deploy --registry your-registry.com/your-org --wait --follow-logs
```

## 📊 What Gets Deployed

| Resource | Name | Purpose |
|----------|------|---------|
| **Deployment** | `mock-log-generator` | Main application workload |
| **ConfigMap** | `mock-log-generator-config` | Environment configuration |
| **ServiceAccount** | `mock-log-generator` | Pod identity and permissions |
| **NetworkPolicy** | `mock-log-generator-netpol` | Network security |
| **ClusterRole** | `mock-log-generator-cluster-role` | Node read permissions |
| **RoleBinding** | `mock-log-generator-rolebinding` | Namespace permissions |
| **ClusterRoleBinding** | `mock-log-generator-cluster-rolebinding` | Cluster permissions |

## 🎯 Alert Patterns Generated

The MockLogGenerator generates 19 distinct alert patterns for comprehensive testing:

### Critical Patterns (High Priority)
1. **critical_namespace_alerts** - Production critical alerts
2. **payment_failures** - Payment processing failures
3. **database_errors** - Database connection issues
4. **authentication_failures** - User auth failures

### Error Patterns (Medium Priority)
5. **high_error_rate** - High volume ERROR logs
6. **service_timeouts** - Service timeout scenarios
7. **cross_service_errors** - Service communication failures
8. **email_smtp_failed** - Email delivery failures
9. **redis_connection_refused** - Cache connection failures

### Warning Patterns (Lower Priority)
10. **high_warn_rate** - High volume WARNING logs
11. **inventory_warnings** - Stock level warnings
12. **notification_failures** - Notification delivery issues
13. **audit_issues** - Security audit violations

### Specialized Patterns
14. **checkout_payment_failed** - E-commerce checkout failures
15. **inventory_stock_unavailable** - Stock availability issues
16. **message_queue_full** - Message queue overflow
17. **timeout_any_service** - General timeout scenarios
18. **slow_query** - Database performance issues
19. **deadlock_detected** - Database deadlock scenarios

### 🧪 E2E Test Mode Alignment

**✅ ALIGNED WITH E2E TESTS**: MockLogGenerator now perfectly matches e2e test expectations:

- **Test Mode** (`LOG_MODE=test`): 
  - Uses `threshold=1` + buffer for reliable alert triggering
  - Generates specific service/log level combinations expected by e2e tests
  - Prioritizes 11 core patterns from `comprehensive_e2e_test_config.json`
  - Ensures exact keyword matching for pattern recognition

- **E2E Test Pattern Mapping**:
  ```
  payment-service    → ERROR logs  (high_error_rate)
  user-service       → WARN logs   (high_warn_rate)  
  database-service   → FATAL logs  (database_errors)
  authentication-api → ERROR logs  (authentication_failures)
  + 8 additional service-specific patterns
  ```

- **Continuous Mode** (`LOG_MODE=continuous`): Production-style simulation

## 🔧 Configuration Options

### Environment Variables (ConfigMap)
```yaml
LOG_MODE: "continuous"  # or "test"
LOG_LEVEL: "INFO"
LOG_GENERATION_INTERVAL: "1.0"  # seconds
BURST_INTERVAL: "10"  # cycles between alert bursts
CLUSTER_NAME: "openshift-cluster"  # optional identifier
```

**📝 Architecture Note**: No Kafka configuration needed - logs output to stdout for Vector/ClusterLogForwarder collection

### Runtime Configuration
```bash
# Adjust log generation speed
oc patch configmap mock-log-generator-config -n mock-logs \
  -p '{"data":{"LOG_GENERATION_INTERVAL":"0.5"}}'

# Change alert pattern frequency
oc patch configmap mock-log-generator-config -n mock-logs \
  -p '{"data":{"BURST_INTERVAL":"5"}}'

# Switch to test mode for e2e testing
oc patch configmap mock-log-generator-config -n mock-logs \
  -p '{"data":{"LOG_MODE":"test"}}'

# Restart to apply changes
oc rollout restart deployment/mock-log-generator -n mock-logs
```

## 🔐 Security Features

### Pod Security
- **Non-root user**: Runs as UID 1001
- **Dropped capabilities**: All Linux capabilities removed
- **Read-only root filesystem**: Where possible
- **Security contexts**: OpenShift SCC compliant

### Network Security
- **NetworkPolicy**: Restricts ingress/egress traffic
- **Minimal connectivity**: Only DNS and container registry access
- **DNS resolution**: Allowed for service discovery  
- **No external internet**: Minimal external access

### RBAC Security
- **Minimal permissions**: Only required access granted
- **Namespace-scoped**: Most permissions limited to mock-logs namespace
- **Node read-only**: Limited cluster-level access for metadata

## 📈 Resource Requirements

### Default Resources
```yaml
requests:
  memory: "128Mi"
  cpu: "100m"
limits:
  memory: "256Mi"
  cpu: "200m"
```

### Scaling Considerations
- **Single replica**: Controlled log generation rate
- **Horizontal scaling**: Not recommended (would multiply log volume)
- **Vertical scaling**: Increase resources if needed for higher throughput

## 🔍 Monitoring & Troubleshooting

### Quick Status Check
```bash
./deploy.sh status
```

### View Logs
```bash
./deploy.sh logs
# or
oc logs -n mock-logs -l app=mock-log-generator -f
```

### Common Issues
1. **Image pull errors**: Verify registry access and image name
2. **Log generation issues**: Check pod logs and application health
3. **RBAC issues**: Verify ServiceAccount and role bindings
4. **Resource limits**: Monitor CPU/memory usage

### Debug Commands
```bash
# Check pod details
oc describe pod -n mock-logs -l app=mock-log-generator

# Test log generation
oc logs -n mock-logs -l app=mock-log-generator --tail 10

# Verify log flow to Vector
oc logs -n openshift-logging -l component=vector | grep mock-logs
```

## 🧹 Cleanup

### Remove Deployment
```bash
./deploy.sh undeploy
```

### Manual Cleanup
```bash
oc delete all,configmap,serviceaccount,networkpolicy \
  -n mock-logs -l app=mock-log-generator
```

## 🔄 Integration Flow

```mermaid
graph LR
    A[MockLogGenerator Pod (mock-logs namespace)] --> B[Stdout Logs]
    B --> C[ClusterLogForwarder/Vector]
    C --> D[Kafka Topic: application-logs]
    D --> E[Alert Engine (alert-engine namespace)]
    E --> F[Alert Processing]
    F --> G[Slack Notifications]
    
    style A fill:#e1f5fe
    style D fill:#fff3e0
    style E fill:#f3e5f5
    style G fill:#e8f5e8
```

## 📞 Support

For issues or questions:
1. Check the [README.md](README.md) for detailed documentation
2. Review [Alert Engine infrastructure setup](../../alert_engine_infra_setup.md)
3. Examine logs using `./deploy.sh logs`
4. Use `./deploy.sh status` for deployment health

---

**🎉 MockLogGenerator Deployment Package v1.0.0**

Complete, production-ready continuous log generator for Alert Engine testing on OpenShift.

**Last Updated**: December 2024  
**Compatibility**: OpenShift 4.16+, Alert Engine v1.0.0 