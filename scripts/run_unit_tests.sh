#!/bin/bash

# Unit Testing Script for Alert Engine
# This script runs unit tests for all packages using build tags

set -e

echo "=========================================="
echo "Alert Engine - Unit Test Suite"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to run tests for a package
run_package_tests() {
    local package_path=$1
    local package_name=$2
    
    print_status $YELLOW "Testing ${package_name}..."
    
    # Check if there are any test files in the package
    if find "$package_path" -name "*_test.go" -type f | grep -q .; then
        echo "  📁 Package: $package_path"
        
        # Run unit tests with build tag
        if go test -tags=unit -v "$package_path"; then
            print_status $GREEN "  ✅ ${package_name} tests PASSED"
        else
            print_status $RED "  ❌ ${package_name} tests FAILED"
            return 1
        fi
    else
        print_status $YELLOW "  ⚠️  No test files found in ${package_path}"
    fi
    echo ""
}

# Function to run tests with coverage
run_coverage_tests() {
    local package_path=$1
    local package_name=$2
    local coverage_file="coverage_${package_name}.out"
    
    print_status $YELLOW "Running coverage for ${package_name}..."
    
    # Check if there are any test files in the package
    if find "$package_path" -name "*_test.go" -type f | grep -q .; then
        if go test -tags=unit -coverprofile="$coverage_file" -covermode=atomic "$package_path"; then
            # Display coverage results
            coverage=$(go tool cover -func="$coverage_file" | grep total | awk '{print $3}')
            print_status $GREEN "  📊 ${package_name} coverage: $coverage"
            
            # Generate HTML coverage report
            go tool cover -html="$coverage_file" -o "coverage_${package_name}.html"
            print_status $GREEN "  📄 HTML coverage report: coverage_${package_name}.html"
        else
            print_status $RED "  ❌ ${package_name} coverage test FAILED"
            return 1
        fi
    else
        print_status $YELLOW "  ⚠️  No test files found in ${package_path}"
    fi
    echo ""
}

# Main execution
main() {
    echo "Starting unit tests..."
    echo ""
    
    # Set working directory to project root
    cd "$(dirname "$0")/.."
    
    # Clean previous coverage files
    rm -f coverage_*.out coverage_*.html
    
    # Test individual packages
    print_status $YELLOW "=== UNIT TESTS ==="
    
    echo "Testing pkg/models..."
    run_package_tests "./pkg/models" "pkg/models"
    
    echo "Testing internal/alerting..."
    run_package_tests "./internal/alerting" "internal/alerting"
    
    echo "Testing internal/api..."
    run_package_tests "./internal/api" "internal/api"
    
    echo "Testing internal/kafka..."
    run_package_tests "./internal/kafka" "internal/kafka"
    
    echo "Testing internal/notifications..."
    run_package_tests "./internal/notifications" "internal/notifications"
    
    echo "Testing internal/storage..."
    run_package_tests "./internal/storage" "internal/storage"
    
    # Run coverage tests if requested
    if [ "$1" = "--coverage" ]; then
        print_status $YELLOW "=== COVERAGE TESTS ==="
        
        run_coverage_tests "./pkg/models" "models"
        run_coverage_tests "./internal/alerting" "alerting"
        run_coverage_tests "./internal/api" "api"
        run_coverage_tests "./internal/kafka" "kafka"
        run_coverage_tests "./internal/notifications" "notifications"
        run_coverage_tests "./internal/storage" "storage"
        
        # Generate combined coverage report
        print_status $YELLOW "Generating combined coverage report..."
        echo "mode: atomic" > coverage_combined.out
        for f in coverage_*.out; do
            if [ "$f" != "coverage_combined.out" ]; then
                tail -n +2 "$f" >> coverage_combined.out
            fi
        done
        
        total_coverage=$(go tool cover -func=coverage_combined.out | grep total | awk '{print $3}')
        print_status $GREEN "🎯 Total project coverage: $total_coverage"
        
        # Generate combined HTML report
        go tool cover -html=coverage_combined.out -o coverage_combined.html
        print_status $GREEN "📄 Combined HTML coverage report: coverage_combined.html"
    fi
    
    # Run all unit tests together for final verification
    print_status $YELLOW "=== FINAL VERIFICATION ==="
    echo "Running all unit tests together..."
    
    if go test -tags=unit -v ./...; then
        print_status $GREEN "🎉 ALL UNIT TESTS PASSED!"
    else
        print_status $RED "💥 SOME UNIT TESTS FAILED!"
        exit 1
    fi
    
    echo ""
    print_status $GREEN "=========================================="
    print_status $GREEN "Unit Test Suite Completed Successfully!"
    print_status $GREEN "=========================================="
}

# Show usage information
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --coverage    Run tests with coverage analysis"
    echo "  --help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                    # Run unit tests only"
    echo "  $0 --coverage         # Run unit tests with coverage"
    echo ""
}

# Handle command line arguments
case "$1" in
    --help|-h)
        show_usage
        exit 0
        ;;
    --coverage)
        main --coverage
        ;;
    "")
        main
        ;;
    *)
        echo "Unknown option: $1"
        show_usage
        exit 1
        ;;
esac 