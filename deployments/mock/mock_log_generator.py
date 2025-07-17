#!/usr/bin/env python3
"""
Mock Log Generator for Alert Engine OpenShift Deployment

Generates realistic log entries for all 19 supported alert patterns
and outputs them to stdout in JSON format for collection by Vector/ClusterLogForwarder.

CORRECT ARCHITECTURE: 
MockLogGenerator (stdout) → Vector/ClusterLogForwarder → Kafka → Alert Engine

ALIGNED WITH E2E TESTS: 
- Test mode optimized for e2e test compatibility (threshold=1, specific log levels)
- Continuous mode uses production-style thresholds for realistic simulation
- Supports all 11 patterns tested in comprehensive_e2e_test_config.json
"""

import json
import time
import random
import logging
import os
import signal
import sys
from datetime import datetime
from typing import Dict, List, Any
import threading

class MockLogGenerator:
    def __init__(self):
        # Get configuration from environment variables
        self.mode = os.getenv('LOG_MODE', 'continuous')
        self.log_level = os.getenv('LOG_LEVEL', 'INFO')
        self.generation_interval = float(os.getenv('LOG_GENERATION_INTERVAL', '1.0'))
        self.burst_interval = int(os.getenv('BURST_INTERVAL', '10'))
        
        # Get OpenShift-specific metadata
        self.namespace = os.getenv('POD_NAMESPACE', 'mock-logs')
        self.pod_name = os.getenv('POD_NAME', 'mock-log-generator')
        self.node_name = os.getenv('NODE_NAME', 'unknown-node')
        
        self.running = False
        
        # Service definitions for log generation
        self.services = [
            "payment-service", "user-service", "database-service", 
            "authentication-api", "inventory-service", "notification-service",
            "order-service", "shipping-service", "billing-service", "audit-service",
            "checkout-service", "email-service", "redis-cache", "message-queue"
        ]
        
        # Pattern definitions matching Alert Engine's 19 supported patterns
        self.patterns = {
            "high_error_rate": {
                "conditions": {"log_level": "ERROR", "threshold": 10, "time_window": 300},
                "keywords": ["error", "failed", "exception", "timeout"],
                "services": ["payment-service", "user-service", "database-service"]
            },
            "payment_failures": {
                "conditions": {"service": "payment-service", "keywords": ["payment failed"], "threshold": 5},
                "keywords": ["payment failed", "transaction declined", "payment timeout"],
                "services": ["payment-service"]
            },
            "database_errors": {
                "conditions": {"service": "database-service", "log_level": "ERROR", "threshold": 3},
                "keywords": ["connection refused", "deadlock detected", "query timeout"],
                "services": ["database-service"]
            },
            "authentication_failures": {
                "conditions": {"service": "authentication-api", "keywords": ["authentication failed"], "threshold": 10},
                "keywords": ["authentication failed", "invalid credentials", "token expired"],
                "services": ["authentication-api"]
            },
            "service_timeouts": {
                "conditions": {"keywords": ["timeout"], "threshold": 5, "time_window": 300},
                "keywords": ["timeout", "connection timeout", "request timeout", "gateway timeout"],
                "services": ["payment-service", "user-service", "inventory-service"]
            },
            "critical_namespace_alerts": {
                "conditions": {"namespace": "production", "log_level": "CRITICAL", "threshold": 1},
                "keywords": ["critical", "fatal", "emergency", "disaster"],
                "services": ["order-service", "payment-service", "user-service"]
            },
            "inventory_warnings": {
                "conditions": {"service": "inventory-service", "log_level": "WARN", "threshold": 15},
                "keywords": ["low stock", "inventory warning", "stock alert", "out of stock"],
                "services": ["inventory-service"]
            },
            "notification_failures": {
                "conditions": {"service": "notification-service", "keywords": ["notification failed"], "threshold": 8},
                "keywords": ["notification failed", "email failed", "sms failed", "push notification failed"],
                "services": ["notification-service"]
            },
            "high_warn_rate": {
                "conditions": {"log_level": "WARN", "threshold": 25, "time_window": 600},
                "keywords": ["warning", "deprecated", "slow response", "performance"],
                "services": ["order-service", "shipping-service", "billing-service"]
            },
            "audit_issues": {
                "conditions": {"service": "audit-service", "keywords": ["audit"], "threshold": 5},
                "keywords": ["audit trail", "security audit", "access violation", "unauthorized access"],
                "services": ["audit-service"]
            },
            "checkout_payment_failed": {
                "conditions": {"service": "checkout-service", "keywords": ["payment failed"], "threshold": 1},
                "keywords": ["payment failed", "payment declined", "checkout failed"],
                "services": ["checkout-service"]
            },
            "inventory_stock_unavailable": {
                "conditions": {"service": "inventory-service", "keywords": ["stock unavailable"], "threshold": 1},
                "keywords": ["stock unavailable", "out of stock", "inventory depleted"],
                "services": ["inventory-service"]
            },
            "email_smtp_failed": {
                "conditions": {"service": "email-service", "keywords": ["SMTP connection failed"], "threshold": 1},
                "keywords": ["SMTP connection failed", "email server down", "mail delivery failed"],
                "services": ["email-service"]
            },
            "redis_connection_refused": {
                "conditions": {"service": "redis-cache", "keywords": ["connection refused"], "threshold": 1},
                "keywords": ["connection refused", "redis unavailable", "cache connection failed"],
                "services": ["redis-cache"]
            },
            "message_queue_full": {
                "conditions": {"service": "message-queue", "keywords": ["queue full"], "threshold": 1},
                "keywords": ["queue full", "message queue overflow", "queue capacity exceeded"],
                "services": ["message-queue"]
            },
            "timeout_any_service": {
                "conditions": {"keywords": ["timeout"], "threshold": 1},
                "keywords": ["timeout", "request timeout", "connection timeout"],
                "services": ["payment-service", "user-service", "inventory-service", "order-service"]
            },
            "slow_query": {
                "conditions": {"keywords": ["slow query"], "threshold": 1},
                "keywords": ["slow query", "query timeout", "database performance"],
                "services": ["database-service", "order-service", "user-service"]
            },
            "deadlock_detected": {
                "conditions": {"keywords": ["deadlock detected"], "threshold": 1},
                "keywords": ["deadlock detected", "database deadlock", "transaction deadlock"],
                "services": ["database-service"]
            },
            "cross_service_errors": {
                "conditions": {"keywords": ["service unavailable"], "threshold": 3, "time_window": 180},
                "keywords": ["service unavailable", "dependency failed", "circuit breaker", "upstream error"],
                "services": ["payment-service", "user-service", "inventory-service", "order-service"]
            }
        }
        
        self.setup_logging()
    
    def setup_logging(self):
        """Setup logging to stderr for application logs, stdout for generated logs"""
        log_level = getattr(logging, self.log_level.upper(), logging.INFO)
        
        # Application logs go to stderr so they don't interfere with generated logs on stdout
        logging.basicConfig(
            level=log_level,
            format='%(asctime)s - %(levelname)s - %(message)s',
            stream=sys.stderr
        )
        self.logger = logging.getLogger(__name__)
    
    def generate_base_log(self, service: str, level: str = "INFO") -> Dict[str, Any]:
        """Generate a base log entry with proper OpenShift/Kubernetes structure"""
        # Use actual namespaces or simulate different environments
        namespace = random.choice([self.namespace, "production", "staging", "development"])
        pod_id = f"{service}-{random.randint(10000, 99999)}-{random.choice(['abc', 'def', 'xyz'])}"
        container_id = f"cri-o://{random.randbytes(32).hex()[:12]}"
        
        # Generate base log structure matching pkg/models/log.go LogEntry
        log_entry = {
            "timestamp": datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%S.%fZ"),
            "@timestamp": datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%S.%fZ"),
            "level": level,
            "service": service,
            "namespace": namespace,
            "host": self.node_name,
            "hostname": self.node_name,  # Alternative hostname field
            "log_source": "application",  # Source classification
            "log_type": "structured",     # Type classification
            # OpenShift/Kubernetes structure expected by ClusterLogForwarder and Alert Engine
            "kubernetes": {
                "namespace": namespace,           # Legacy field for compatibility
                "namespace_name": namespace,      # OpenShift format
                "pod": pod_id,                   # Legacy field for compatibility  
                "pod_name": pod_id,              # OpenShift format (preferred)
                "container": service,            # Legacy field for compatibility
                "container_name": service,       # OpenShift format (preferred)
                "labels": {
                    "app": service,
                    "version": f"v{random.randint(1, 3)}.{random.randint(0, 9)}.{random.randint(0, 9)}",
                    "environment": namespace,
                    "component": service.split('-')[0] if '-' in service else service,
                    "generated-by": "mock-log-generator"
                },
                "annotations": {
                    "deployment.kubernetes.io/revision": str(random.randint(1, 10)),
                    "kubectl.kubernetes.io/last-applied-configuration": "{...}",
                    "openshift.io/generated-by": "OpenShiftNewApp"
                },
                "container_id": container_id,
                "pod_ip": f"10.{random.randint(128, 255)}.{random.randint(1, 254)}.{random.randint(1, 254)}",
                "pod_owner": f"ReplicaSet/{service}-{random.randbytes(4).hex()}"
            }
        }
        
        return log_entry
    
    def generate_pattern_log(self, pattern_name: str, pattern_config: Dict) -> Dict[str, Any]:
        """Generate a log entry for a specific alert pattern"""
        service = random.choice(pattern_config["services"])
        
        # Determine log level based on pattern, with e2e test specific overrides
        if "log_level" in pattern_config["conditions"]:
            level = pattern_config["conditions"]["log_level"]
        else:
            level = random.choice(["ERROR", "WARN", "INFO", "FATAL"])
        
        # E2E test specific overrides to match exact test expectations
        if self.mode == "test":
            e2e_overrides = {
                "high_error_rate": {"level": "ERROR", "service": "payment-service"},
                "high_warn_rate": {"level": "WARN", "service": "user-service"},
                "database_errors": {"level": "FATAL", "service": "database-service"},  # E2E expects FATAL
                "authentication_failures": {"level": "ERROR", "service": "authentication-api"}
            }
            
            if pattern_name in e2e_overrides:
                override = e2e_overrides[pattern_name]
                if "level" in override:
                    level = override["level"]
                if "service" in override:
                    service = override["service"]
        
        log = self.generate_base_log(service, level)
        
        # Add pattern-specific message with keywords
        keyword = random.choice(pattern_config["keywords"])
        
        # Generate contextual messages based on pattern
        # Enhanced for e2e test compatibility - ensure keywords match test expectations
        if self.mode == "test":
            # E2E test specific messages to ensure exact keyword matching
            messages = {
                "high_error_rate": f"Payment service error: {keyword} - transaction processing failed for user {random.randint(1000, 9999)}",
                "high_warn_rate": f"User service warning: {keyword} - memory usage approaching limits",
                "database_errors": f"Database fatal error: {keyword} - connection lost to primary database",
                "authentication_failures": f"Authentication API error: {keyword} - invalid token for user {random.randint(1000, 9999)}",
                "checkout_payment_failed": f"Checkout process: {keyword} for order #{random.randint(10000, 99999)}",
                "inventory_stock_unavailable": f"Inventory alert: {keyword} for item SKU-{random.randint(1000, 9999)}",
                "email_smtp_failed": f"Email service: {keyword} while connecting to mail server",
                "redis_connection_refused": f"Redis cache: {keyword} - unable to establish connection",
                "message_queue_full": f"Message queue: {keyword} - broker capacity exceeded",
                "timeout_any_service": f"Service operation: {keyword} after 30 seconds waiting for response",
                "slow_query": f"Database query: {keyword} detected - execution time 2.5 seconds",
                "deadlock_detected": f"Database transaction: {keyword} between user operations"
            }
        else:
            # Production-style messages for continuous mode
            messages = {
                "high_error_rate": f"Service {service} encountered an error: {keyword} during request processing",
                "payment_failures": f"Payment processing failed: {keyword} for transaction ID {random.randint(10000, 99999)}",
                "database_errors": f"Database operation failed: {keyword} on table users_table",
                "authentication_failures": f"User authentication failed: {keyword} for user ID {random.randint(1000, 9999)}",
                "service_timeouts": f"Service request timeout: {keyword} after 30 seconds waiting for response",
                "critical_namespace_alerts": f"Critical system failure: {keyword} - immediate attention required",
                "inventory_warnings": f"Inventory alert: {keyword} for product SKU-{random.randint(1000, 9999)}",
                "notification_failures": f"Notification delivery failed: {keyword} to user {random.randint(1000, 9999)}",
                "high_warn_rate": f"Performance warning: {keyword} detected in service operation",
                "audit_issues": f"Security audit issue: {keyword} detected in user action",
                "cross_service_errors": f"Service communication error: {keyword} when calling downstream service",
                "checkout_payment_failed": f"Checkout process failed: {keyword} for order #{random.randint(10000, 99999)}",
                "inventory_stock_unavailable": f"Inventory management: {keyword} for item SKU-{random.randint(1000, 9999)}",
                "email_smtp_failed": f"Email service error: {keyword} while sending notification",
                "redis_connection_refused": f"Cache service error: {keyword} - unable to connect to Redis",
                "message_queue_full": f"Message broker alert: {keyword} - unable to enqueue message",
                "timeout_any_service": f"Service operation: {keyword} after waiting 30 seconds",
                "slow_query": f"Database performance: {keyword} detected - execution time exceeded threshold",
                "deadlock_detected": f"Database concurrency issue: {keyword} in transaction processing"
            }
        
        log["message"] = messages.get(pattern_name, f"Service log: {keyword}")
        
        # Add raw field for model compliance (contains original log format)
        raw_log = {
            "timestamp": log["timestamp"],
            "level": log["level"],
            "message": log["message"],
            "service": log["service"]
        }
        log["raw"] = json.dumps(raw_log, separators=(',', ':'))
        
        return log
    
    def generate_normal_log(self) -> Dict[str, Any]:
        """Generate a normal log entry (not triggering alerts)"""
        service = random.choice(self.services)
        level = random.choice(["INFO", "DEBUG"])
        log = self.generate_base_log(service, level)
        
        normal_messages = [
            f"Request processed successfully for user {random.randint(1000, 9999)}",
            f"Service {service} started successfully",
            f"Database query completed in {random.randint(10, 500)}ms",
            f"Cache hit for key user:{random.randint(1000, 9999)}",
            f"Health check passed for {service}",
            f"Processing batch job with {random.randint(10, 100)} items",
            f"User session created for user {random.randint(1000, 9999)}",
            f"Configuration loaded successfully",
            f"Metrics published to monitoring system",
            f"Background task completed successfully"
        ]
        
        log["message"] = random.choice(normal_messages)
        
        # Add raw field for model compliance (contains original log format)
        raw_log = {
            "timestamp": log["timestamp"],
            "level": log["level"],
            "message": log["message"],
            "service": log["service"]
        }
        log["raw"] = json.dumps(raw_log, separators=(',', ':'))
        
        return log
    
    def output_log(self, log_entry: Dict[str, Any]):
        """Output log entry to stdout in JSON format for Vector/ClusterLogForwarder collection"""
        try:
            # Output to stdout as JSON - this is what Vector/ClusterLogForwarder will collect
            print(json.dumps(log_entry), flush=True)
        except Exception as e:
            self.logger.error(f"Failed to output log: {e}")
    
    def generate_pattern_burst(self, pattern_name: str, count: int = None):
        """Generate a burst of logs for a specific pattern (to trigger alerts)"""
        pattern_config = self.patterns[pattern_name]
        
        # Auto-calculate count if not provided, using test-friendly thresholds in test mode
        if count is None:
            if self.mode == "test":
                # For e2e tests, use threshold=1 + small buffer to ensure alert triggering
                count = 2  # Simple: always generate 2 logs to exceed threshold=1
            else:
                # For continuous mode, use pattern thresholds + buffer
                threshold = pattern_config["conditions"].get("threshold", 5)
                count = threshold + random.randint(1, 3)
        
        self.logger.info(f"Generating {count} logs for pattern: {pattern_name}")
        
        for _ in range(count):
            log = self.generate_pattern_log(pattern_name, pattern_config)
            self.output_log(log)
            time.sleep(0.1)  # Small delay between logs in burst
    
    def continuous_generation(self):
        """Continuously generate logs with periodic pattern bursts"""
        self.logger.info(f"Starting continuous log generation (interval: {self.generation_interval}s, burst_interval: {self.burst_interval})")
        
        cycle_counter = 0
        while self.running:
            try:
                # Generate normal logs most of the time
                for _ in range(5):
                    if not self.running:
                        break
                    log = self.generate_normal_log()
                    self.output_log(log)
                    time.sleep(self.generation_interval)
                
                # Periodically generate pattern bursts to trigger alerts
                cycle_counter += 1
                if cycle_counter % self.burst_interval == 0:
                    pattern_name = random.choice(list(self.patterns.keys()))
                    
                    # Use auto-calculation which handles continuous vs test mode appropriately
                    self.generate_pattern_burst(pattern_name)
                    
                    # Wait before continuing normal generation
                    time.sleep(5)
                    
            except Exception as e:
                self.logger.error(f"Error in continuous generation: {e}")
                time.sleep(10)  # Wait before retrying
    
    def test_all_patterns(self):
        """Generate test logs for all patterns (one-time) - optimized for e2e testing"""
        self.logger.info(f"Testing all {len(self.patterns)} alert patterns...")
        
        # Focus on the 11 patterns specifically tested in e2e tests
        e2e_test_patterns = [
            "high_error_rate",  # Payment Service ERROR logs
            "high_warn_rate",   # User Service WARN messages  
            "database_errors",  # Database Service FATAL errors -> but we'll use ERROR as that's what pattern does
            "authentication_failures",  # Authentication API ERROR logs
            "checkout_payment_failed",   # Checkout Service payment failed
            "inventory_stock_unavailable",  # Inventory Service stock unavailable
            "email_smtp_failed",         # Email Service SMTP connection failed
            "redis_connection_refused",  # Redis Cache connection refused
            "message_queue_full",        # Message Queue full warnings
            "timeout_any_service",       # Timeout in any service
            "slow_query",               # Slow query in database logs
            "deadlock_detected"         # Deadlock detected in database service
        ]
        
        # Test the key e2e patterns first
        for pattern_name in e2e_test_patterns:
            if pattern_name in self.patterns:
                self.logger.info(f"Testing e2e pattern: {pattern_name}")
                self.generate_pattern_burst(pattern_name)  # Uses auto-calculation for test mode
                time.sleep(1)  # Shorter delay for e2e efficiency
        
        # Test remaining patterns if any
        remaining_patterns = set(self.patterns.keys()) - set(e2e_test_patterns)
        for pattern_name in remaining_patterns:
            self.logger.info(f"Testing additional pattern: {pattern_name}")
            self.generate_pattern_burst(pattern_name)  # Uses auto-calculation for test mode
            time.sleep(1)
        
        self.logger.info("All pattern tests completed")
    
    def start(self):
        """Start the log generator"""
        self.logger.info(f"MockLogGenerator starting in {self.mode} mode")
        self.logger.info(f"Outputting logs to stdout for Vector/ClusterLogForwarder collection")
        self.logger.info(f"Namespace: {self.namespace}")
        self.logger.info(f"Pod Name: {self.pod_name}")
        
        if self.mode == "test":
            self.logger.info("E2E Test Mode: Optimized for e2e test pattern matching")
            self.logger.info("- Using threshold=1 + small buffer for all patterns")
            self.logger.info("- Generating specific log levels and services for e2e compatibility")
            self.logger.info("- Prioritizing 11 key e2e test patterns")
        else:
            self.logger.info("Continuous Mode: Using production-style thresholds and rotation")
        
        self.running = True
        
        try:
            if self.mode == "test":
                self.test_all_patterns()
            elif self.mode == "continuous":
                self.continuous_generation()
            else:
                self.logger.error(f"Unknown mode: {self.mode}")
                return False
        except KeyboardInterrupt:
            self.logger.info("Received interrupt signal")
        except Exception as e:
            self.logger.error(f"Unexpected error: {e}")
            return False
        finally:
            self.stop()
        
        return True
    
    def stop(self):
        """Stop the log generator"""
        self.logger.info("Stopping MockLogGenerator...")
        self.running = False

def signal_handler(signum, frame):
    """Handle interrupt signals"""
    print(f"\nReceived signal {signum}. Shutting down gracefully...", file=sys.stderr)
    sys.exit(0)

if __name__ == "__main__":
    import argparse
    
    # Set up signal handlers for graceful shutdown
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    parser = argparse.ArgumentParser(description="Mock Log Generator for Alert Engine OpenShift Deployment")
    parser.add_argument("--mode", choices=["continuous", "test"], help="Run mode (overrides LOG_MODE env var)")
    
    args = parser.parse_args()
    
    # Create generator instance
    generator = MockLogGenerator()
    
    # Override with command line arguments if provided
    if args.mode:
        generator.mode = args.mode
    
    # Start the generator
    success = generator.start()
    if not success:
        sys.exit(1) 