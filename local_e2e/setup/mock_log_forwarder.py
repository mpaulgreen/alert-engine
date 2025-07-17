#!/usr/bin/env python3
"""
Mock Log Forwarder for Alert Engine E2E Testing

Generates realistic log entries for all 11 supported alert patterns
and forwards them to Kafka for processing by the alert engine.
"""

import json
import time
import random
import logging
from datetime import datetime
from kafka import KafkaProducer
from typing import Dict, List, Any
import threading
import signal
import sys

class MockLogForwarder:
    def __init__(self, kafka_broker: str = "localhost:9094", topic: str = "application-logs"):
        self.kafka_broker = kafka_broker
        self.topic = topic
        self.producer = None
        self.running = False
        self.services = [
            "payment-service", "user-service", "database-service", 
            "authentication-api", "inventory-service", "notification-service",
            "order-service", "shipping-service", "billing-service", "audit-service",
            # Additional services needed by test rules
            "checkout-service", "email-service", "redis-cache", "message-queue"
        ]
        
        # Pattern definitions matching our 11 supported patterns
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
            # Additional patterns to match test rules exactly
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
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(levelname)s - %(message)s'
        )
        self.logger = logging.getLogger(__name__)
    
    def connect_kafka(self):
        """Connect to Kafka producer"""
        try:
            self.producer = KafkaProducer(
                bootstrap_servers=[self.kafka_broker],
                value_serializer=lambda v: json.dumps(v).encode('utf-8'),
                key_serializer=lambda k: k.encode('utf-8') if k else None
            )
            self.logger.info(f"Connected to Kafka at {self.kafka_broker}")
            return True
        except Exception as e:
            self.logger.error(f"Failed to connect to Kafka: {e}")
            return False
    
    def generate_base_log(self, service: str, level: str = "INFO") -> Dict[str, Any]:
        """Generate a base log entry with proper Kubernetes structure"""
        namespace = random.choice(["production", "staging", "development"])
        pod_id = f"{service}-{random.randint(10000, 99999)}-{random.choice(['abc', 'def', 'xyz'])}"
        
        return {
            "timestamp": datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%S.%fZ"),
            "@timestamp": datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%S.%fZ"),
            "level": level,
            "service": service,  # Keep for backward compatibility
            "namespace": namespace,  # Keep top-level for fallback
            "host": f"worker-node-{random.randint(1, 3)}",
            "thread_id": f"thread-{random.randint(1, 20)}",
            "trace_id": f"trace-{random.randint(100000, 999999)}",
            # CRITICAL: Add Kubernetes structure expected by alert engine
            "kubernetes": {
                "namespace": namespace,
                "pod": pod_id,
                "container": service,
                "labels": {
                    "app": service,                    # CRITICAL: Service name for matching
                    "version": f"v{random.randint(1, 3)}.{random.randint(0, 9)}.{random.randint(0, 9)}",
                    "environment": namespace,
                    "component": service.split('-')[0] if '-' in service else service
                }
            }
        }
    
    def generate_pattern_log(self, pattern_name: str, pattern_config: Dict) -> Dict[str, Any]:
        """Generate a log entry for a specific pattern"""
        service = random.choice(pattern_config["services"])
        
        # Determine log level based on pattern
        if "log_level" in pattern_config["conditions"]:
            level = pattern_config["conditions"]["log_level"]
        else:
            level = random.choice(["ERROR", "WARN", "INFO", "FATAL"])
        
        log = self.generate_base_log(service, level)
        
        # Add pattern-specific message with keywords
        keyword = random.choice(pattern_config["keywords"])
        
        # Generate contextual messages based on pattern
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
            # New messages for test rule patterns
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
        
        # Add additional contextual fields
        log["request_id"] = f"req-{random.randint(100000, 999999)}"
        log["user_id"] = f"user-{random.randint(1000, 9999)}"
        log["session_id"] = f"session-{random.randint(100000, 999999)}"
        
        # IMPORTANT: Ensure proper namespace structure for alert matching
        # Some rules check for top-level namespace, others check kubernetes.namespace
        if "kubernetes" in log:
            log["kubernetes"]["namespace_name"] = log["kubernetes"]["namespace"]  # Alternative field
        
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
        log["request_id"] = f"req-{random.randint(100000, 999999)}"
        
        return log
    
    def send_log(self, log_entry: Dict[str, Any]):
        """Send log entry to Kafka"""
        try:
            # CRITICAL: Simulate Vector/ClusterLogForwarder transformation
            # Original log becomes a JSON string in the "message" field
            # This matches the actual production pipeline behavior
            
            # Create the transformed log structure that mimics Vector/ClusterLogForwarder output
            transformed_log = {
                "message": json.dumps(log_entry),  # Original log as JSON string
                "@timestamp": log_entry.get("@timestamp", datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%S.%fZ")),
                "level": log_entry.get("level", "INFO").lower(),  # Vector lowercases levels
                "kubernetes": log_entry.get("kubernetes", {}),
                "host": log_entry.get("host", "unknown"),
                "stream": "stdout",  # Added by Vector
                "tag": "kubernetes.var.log.containers",  # Added by Vector
                "source_type": "kubernetes_logs"  # Added by Vector
            }
            
            key = f"{log_entry['service']}-{log_entry['level']}"
            self.producer.send(self.topic, key=key, value=transformed_log)
            self.logger.debug(f"Sent transformed log: {log_entry['service']} - {log_entry['level']} - {log_entry['message'][:100]}...")
        except Exception as e:
            self.logger.error(f"Failed to send log to Kafka: {e}")
    
    def generate_pattern_burst(self, pattern_name: str, count: int):
        """Generate a burst of logs for a specific pattern (to trigger alerts)"""
        pattern_config = self.patterns[pattern_name]
        self.logger.info(f"Generating {count} logs for pattern: {pattern_name}")
        
        for _ in range(count):
            log = self.generate_pattern_log(pattern_name, pattern_config)
            self.send_log(log)
            time.sleep(0.1)  # Small delay between logs
    
    def continuous_generation(self):
        """Continuously generate logs with periodic pattern bursts"""
        self.logger.info("Starting continuous log generation...")
        
        burst_counter = 0
        while self.running:
            # Generate normal logs most of the time
            for _ in range(5):
                if not self.running:
                    break
                log = self.generate_normal_log()
                self.send_log(log)
                time.sleep(random.uniform(0.5, 2.0))
            
            # Periodically generate pattern bursts to trigger alerts
            burst_counter += 1
            if burst_counter % 10 == 0:  # Every 10 cycles
                pattern_name = random.choice(list(self.patterns.keys()))
                pattern_config = self.patterns[pattern_name]
                
                # Generate enough logs to exceed threshold
                threshold = pattern_config["conditions"].get("threshold", 5)
                burst_count = threshold + random.randint(1, 5)
                
                self.generate_pattern_burst(pattern_name, burst_count)
                
                # Wait a bit before continuing normal generation
                time.sleep(5)
    
    def test_all_patterns(self):
        """Generate test logs for all patterns (one-time)"""
        self.logger.info("Testing all 11 alert patterns...")
        
        for pattern_name, pattern_config in self.patterns.items():
            threshold = pattern_config["conditions"].get("threshold", 5)
            burst_count = threshold + 2  # Ensure threshold is exceeded
            
            self.logger.info(f"Testing {pattern_name}: generating {burst_count} logs")
            self.generate_pattern_burst(pattern_name, burst_count)
            
            # Wait between patterns
            time.sleep(2)
        
        self.logger.info("All pattern tests completed")
    
    def start(self, mode: str = "continuous"):
        """Start the log forwarder"""
        if not self.connect_kafka():
            return False
        
        self.running = True
        
        if mode == "test":
            self.test_all_patterns()
        elif mode == "continuous":
            try:
                self.continuous_generation()
            except KeyboardInterrupt:
                self.logger.info("Received interrupt signal")
        
        self.stop()
        return True
    
    def stop(self):
        """Stop the log forwarder"""
        self.running = False
        if self.producer:
            self.producer.flush()
            self.producer.close()
            self.logger.info("Kafka producer closed")

def signal_handler(signum, frame):
    """Handle interrupt signals"""
    print("\nReceived interrupt signal. Shutting down...")
    sys.exit(0)

if __name__ == "__main__":
    import argparse
    
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    parser = argparse.ArgumentParser(description="Mock Log Forwarder for Alert Engine E2E Testing")
    parser.add_argument("--kafka-broker", default="localhost:9094", help="Kafka broker address")
    parser.add_argument("--topic", default="application-logs", help="Kafka topic name")
    parser.add_argument("--mode", choices=["continuous", "test"], default="continuous",
                       help="Run mode: continuous (ongoing) or test (one-time pattern test)")
    
    args = parser.parse_args()
    
    forwarder = MockLogForwarder(args.kafka_broker, args.topic)
    
    print(f"Starting Mock Log Forwarder...")
    print(f"Kafka Broker: {args.kafka_broker}")
    print(f"Topic: {args.topic}")
    print(f"Mode: {args.mode}")
    print(f"Supported Patterns: {len(forwarder.patterns)}")
    
    if args.mode == "test":
        print("Running pattern tests...")
    else:
        print("Running in continuous mode. Press Ctrl+C to stop.")
    
    success = forwarder.start(args.mode)
    if not success:
        sys.exit(1) 