# Alert Engine Scripts

This directory contains various scripts for building, testing, and deploying the Alert Engine.

## 🎯 Quick Script Reference

| Scenario | Script to Use |
|----------|---------------|
| "I want to set up the infrastructure" | `./setup_openshift_infrastructure.sh` |
| "I want to verify my setup is working" | `./validate_openshift_infrastructure.sh` |
| "I want to see what will be deleted" | `./verify_resources_before_cleanup.sh` |
| "I want to remove everything" | `./cleanup_openshift_infrastructure.sh` |
| "I want to run unit tests" | `./run_unit_tests.sh` |
| "I want to run integration tests" | `./run_integration_tests.sh` |

## 📋 Table of Contents

1. [Workflow Integration](#workflow-integration)
2. [OpenShift Infrastructure Scripts](#openshift-infrastructure-scripts)
3. [Cleanup Scripts](#cleanup-scripts)
4. [Testing Scripts](#testing-scripts)
5. [Development Scripts](#development-scripts)

## 🔄 Workflow Integration

### Complete Setup Workflow
```bash
# 1. Deploy infrastructure (10-15 minutes)
cd alert-engine/scripts
./setup_openshift_infrastructure.sh

# 2. Validate everything is working
./validate_openshift_infrastructure.sh

# 3. Deploy Alert Engine application
# (use connection details from validation output)
```

### Complete Cleanup Workflow  
```bash
# 1. See what exists before cleanup
./verify_resources_before_cleanup.sh

# 2. Remove everything (5-10 minutes)
./cleanup_openshift_infrastructure.sh
```

### Testing Workflows
```bash
# Unit testing
./run_unit_tests.sh

# Integration testing  
./run_integration_tests.sh

# Kafka-specific testing
./run_kafka_integration_tests.sh
```

## OpenShift Infrastructure Scripts

> **Note**: This directory contains scripts for different purposes. The validation script serves post-setup verification, while the cleanup verification script serves pre-cleanup assessment. Both are needed for different workflows and use the shared utility library `openshift_utils.sh`.

### 🚀 setup_openshift_infrastructure.sh

**Purpose**: Automates the complete OpenShift infrastructure setup for Alert Engine deployment.

**What it does**:
- **Step 1**: Installs and configures Red Hat AMQ Streams (Kafka)
- **Step 2**: Deploys a highly available Redis cluster (6 nodes)
- **Step 3**: Sets up OpenShift Logging with ClusterLogForwarder

**Prerequisites**:
- OpenShift 4.16.17+ cluster with cluster-admin access
- `oc` CLI tool installed and configured
- Access to OperatorHub in your OpenShift cluster

**Usage**:
```bash
# Basic usage with default storage class (gp3-csi)
./setup_openshift_infrastructure.sh

# With custom storage class
STORAGE_CLASS=fast-ssd ./setup_openshift_infrastructure.sh
```

**Environment Variables**:
- `STORAGE_CLASS`: Storage class to use (default: `gp3-csi`)

**Expected Runtime**: ~10-15 minutes

**Output**: 
- Complete infrastructure setup with connection details
- Sample `config.yaml` for Alert Engine deployment

### 🔍 validate_openshift_infrastructure.sh

**Purpose**: Validates the OpenShift infrastructure components after setup.

**What it validates**:
- Kafka cluster health and connectivity
- Redis cluster status and operations
- OpenShift Logging operator and ClusterLogForwarder
- Alert Engine namespace and RBAC setup
- End-to-end log flow (Log Generator → Vector → Kafka)

**Usage**:
```bash
# Run complete validation
./validate_openshift_infrastructure.sh

# Check specific component (by editing the script)
# Uncomment specific validation functions in main()
```

**Exit Codes**:
- `0`: All validations passed
- `1`: One or more validations failed

### 🔧 openshift_utils.sh

**Purpose**: Shared utility library for common OpenShift operations.

**What it provides**:
- Common resource checking functions
- Logging utilities with colored output
- Prerequisites validation functions
- Connection testing utilities
- Configuration generation helpers

**Usage**:
```bash
# Source the utility library in other scripts
source ./openshift_utils.sh

# Use shared functions
if resource_exists namespace "my-namespace"; then
    log_success "Namespace exists"
fi
```

**Key Functions**:
- `resource_exists()` - Check if a resource exists
- `get_resource_status()` - Get resource status/condition
- `check_openshift_prerequisites()` - Validate oc CLI and cluster access
- `test_kafka_connectivity()` - Test Kafka cluster connectivity
- `test_redis_connectivity()` - Test Redis cluster health
- `generate_alert_engine_config()` - Generate sample configuration

## Cleanup Scripts

### 🗑️ cleanup_openshift_infrastructure.sh

**Purpose**: Removes all OpenShift infrastructure resources created by the setup script.

**What it cleans up**:
- AMQ Streams / Kafka resources (cluster, topics, operators)
- Redis cluster resources (StatefulSet, services, ConfigMaps)
- OpenShift Logging resources (ClusterLogForwarder, operators)
- Alert Engine test resources (deployments, service accounts)
- Network policies and RBAC resources
- Namespaces and PVCs

**Usage**:
```bash
# Run with confirmation prompts
./cleanup_openshift_infrastructure.sh

# The script will ask for confirmation before proceeding
```

**Features**:
- **Safe deletion** with confirmation prompts
- **Progress tracking** with colored output
- **Graceful cleanup** with proper waiting for resource deletion
- **Final verification** to ensure complete cleanup

**Expected Runtime**: ~5-10 minutes

### 🔍 verify_resources_before_cleanup.sh

**Purpose**: Inventories all Alert Engine infrastructure resources before cleanup.

**What it checks**:
- Lists all namespaces and their resources
- Counts pods, services, PVCs, and other objects
- Identifies operators and CSVs
- Shows cluster-level resources (ClusterRoles, ClusterRoleBindings)
- Provides cleanup summary with resource counts

**Usage**:
```bash
# Run inventory check
./verify_resources_before_cleanup.sh

# Example output shows what exists and total resources to cleanup
```

**Features**:
- **Non-destructive** - only reads and reports
- **Comprehensive inventory** of all related resources
- **Cleanup decision support** - helps determine if cleanup is needed
- **Resource counting** for validation

## Testing Scripts

### 🧪 run_unit_tests.sh

**Purpose**: Executes Go unit tests for all packages.

**Usage**:
```bash
./run_unit_tests.sh
```

**Features**:
- Runs tests with race detection
- Generates coverage reports
- Supports verbose output

### 🔗 run_integration_tests.sh

**Purpose**: Runs integration tests with real dependencies.

**Usage**:
```bash
# Run all integration tests
./run_integration_tests.sh

# Run specific package integration tests
./run_integration_tests.sh ./internal/kafka
```

**Prerequisites**:
- Docker or Podman for testcontainers
- Network access for spinning up test containers

### 📨 run_kafka_integration_tests.sh

**Purpose**: Specialized Kafka integration testing.

**Usage**:
```bash
./run_kafka_integration_tests.sh
```

**Features**:
- Uses testcontainers for isolated Kafka testing
- Tests producer-consumer flows
- Validates alert processing pipeline

## Development Scripts

### 🐳 docker-compose.test.yml

**Purpose**: Docker Compose configuration for local testing environment.

**Usage**:
```bash
# Start test environment
docker-compose -f docker-compose.test.yml up -d

# Run tests against local environment
go test ./...

# Cleanup
docker-compose -f docker-compose.test.yml down
```

**Services**:
- Kafka with Zookeeper
- Redis cluster
- Test log generators

## 📖 Complete Setup Guide

### 1. OpenShift Infrastructure Setup

Follow these steps for a complete Alert Engine infrastructure setup:

```bash
# 1. Login to your OpenShift cluster
oc login --token=YOUR_TOKEN --server=YOUR_CLUSTER_URL

# 2. Verify cluster access and storage classes
oc get storageclass
oc auth can-i '*' '*' --all-namespaces

# 3. Run the infrastructure setup
cd alert-engine/scripts
./setup_openshift_infrastructure.sh

# 4. Wait for completion (~10-15 minutes)
# Script will output connection details and sample config

# 5. Validate the setup
./validate_openshift_infrastructure.sh
```

### 2. Local Development Testing

```bash
# Unit tests
./run_unit_tests.sh

# Integration tests with real components
./run_integration_tests.sh

# Kafka-specific integration tests
./run_kafka_integration_tests.sh
```

### 3. Deploying Alert Engine

After infrastructure setup:

```bash
# Update your config.yaml with the connection details from setup script
# Example config is provided in the setup script output

# Build and deploy using the deployment manifests
cd ../deployments/alert-engine
./build.sh
kubectl apply -k .
```

## 🎯 Infrastructure Details

### Components Deployed

| Component | Namespace | Purpose | HA Configuration |
|-----------|-----------|---------|------------------|
| **Kafka Cluster** | `amq-streams-kafka` | Message streaming | 3 brokers + 3 ZooKeeper |
| **Redis Cluster** | `redis-cluster` | State storage | 6 nodes (3 masters + 3 replicas) |
| **OpenShift Logging** | `openshift-logging` | Log collection | Vector-based log forwarding |
| **Alert Engine** | `alert-engine` | Application namespace | RBAC + Service Account |

### Network Configuration

The setup includes network policies for secure communication:

- **Kafka**: Allows access from `alert-engine` and `openshift-logging` namespaces
- **Redis**: Allows access from `alert-engine` namespace only
- **Logging**: System-wide log collection with forwarding to Kafka

### Connection Details

After successful setup, you'll get connection details like:

```yaml
kafka:
  brokers: ["alert-kafka-cluster-kafka-bootstrap.amq-streams-kafka.svc.cluster.local:9092"]
  topic: "application-logs"

redis:
  mode: "cluster"
  addresses: [
    "redis-cluster-0.redis-cluster.redis-cluster.svc.cluster.local:6379",
    # ... more nodes
  ]
```

## 🛠️ Troubleshooting

### Common Issues

1. **Storage Class Not Found**:
   ```bash
   # Check available storage classes
   oc get storageclass
   
   # Use custom storage class
   STORAGE_CLASS=your-storage-class ./setup_openshift_infrastructure.sh
   ```

2. **Operator Installation Fails**:
   ```bash
   # Check operator subscription status
   oc get subscription -n amq-streams-kafka
   oc get csv -n amq-streams-kafka
   
   # Check for missing OperatorGroup (fixed in script)
   oc get operatorgroup -n amq-streams-kafka
   ```

3. **Redis Cluster Initialization Issues**:
   ```bash
   # Check pod logs
   oc logs redis-cluster-0 -n redis-cluster
   
   # Manual cluster initialization if needed
   oc exec -it redis-cluster-0 -n redis-cluster -- redis-cli --cluster create \
     $(oc get pods -l app=redis-cluster -n redis-cluster -o jsonpath='{range.items[*]}{.status.podIP}:6379 ') \
     --cluster-replicas 1 --cluster-yes
   ```

4. **Log Forwarding Not Working**:
   ```bash
   # Check ClusterLogForwarder status
   oc get clusterlogforwarder kafka-forwarder -n openshift-logging -o yaml
   
   # Check Vector pods
   oc get pods -A -l app.kubernetes.io/component=collector
   
   # Validate logs in Kafka
   oc exec -n amq-streams-kafka alert-kafka-cluster-kafka-0 -- \
     bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic application-logs --from-beginning --max-messages 5
   ```

### Validation Commands

```bash
# Quick health check
./validate_openshift_infrastructure.sh

# Individual component checks
oc get kafka alert-kafka-cluster -n amq-streams-kafka
oc get pods -l app=redis-cluster -n redis-cluster
oc get clusterlogforwarder kafka-forwarder -n openshift-logging
```

## 📋 Script Comparison & Usage Guidelines

### 🎯 Script Purpose Matrix

| Script | Purpose | Use Case | Validation Depth |
|--------|---------|----------|------------------|
| `setup_openshift_infrastructure.sh` | **Infrastructure Setup** | Deploy complete OpenShift infrastructure | N/A (Setup) |
| `validate_openshift_infrastructure.sh` | **Post-Setup Validation** | Verify infrastructure health after setup | **Deep** (Health + Connectivity) |
| `verify_resources_before_cleanup.sh` | **Pre-Cleanup Inventory** | List resources before cleanup | **Shallow** (Existence Only) |
| `cleanup_openshift_infrastructure.sh` | **Infrastructure Cleanup** | Remove all OpenShift infrastructure | N/A (Cleanup) |

### 🧩 Key Differences Explained

#### Why Two "Validation" Scripts?

**Different Questions They Answer:**

| Script | Question | Example Output |
|--------|----------|----------------|
| `validate_openshift_infrastructure.sh` | "Is everything working correctly?" | ✅ Kafka cluster healthy<br/>✅ Redis connectivity test passed<br/>✅ End-to-end log flow working |
| `verify_resources_before_cleanup.sh` | "What exists that can be deleted?" | ✅ Found: kafka/alert-kafka-cluster<br/>✅ Found: statefulset/redis-cluster<br/>Total resources to cleanup: 47 |

**Different Validation Approaches:**

```bash
# Post-setup validation (deep)
validate_openshift_infrastructure.sh:
├── Check resource exists ✓
├── Check status condition ✓  
├── Test connectivity ✓
├── Run producer-consumer test ✓
└── Validate end-to-end flow ✓

# Pre-cleanup verification (shallow)  
verify_resources_before_cleanup.sh:
├── Check resource exists ✓
└── Count for cleanup summary ✓
```

### 🏗️ Architecture Rationale

**Separation of Concerns:**
- **Setup** scripts focus on deployment and configuration
- **Validation** scripts focus on health verification  
- **Cleanup** scripts focus on resource removal
- **Testing** scripts focus on code quality verification

**Different Lifecycles:**
- Setup → Validation → Development → Cleanup
- Each phase has different requirements and outputs
- Scripts are optimized for their specific use case

**Shared Utilities:**
All scripts use `openshift_utils.sh` to eliminate code duplication while maintaining their distinct purposes.

## 📚 Additional Resources

- [OpenShift Infrastructure Setup Guide](../alert_engine_infra_setup.md) - Detailed manual setup instructions
- [Alert Engine Deployment](../deployments/alert-engine/README.md) - Application deployment guide
- [Local E2E Testing](../local_e2e/README.md) - Local development and testing
- [Test Strategy](./test_strategy.md) - Comprehensive testing approach 