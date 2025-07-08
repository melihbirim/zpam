#!/bin/bash

# Test script for Redis Bayesian Filter
# This script demonstrates how to test the Redis Bayesian implementation

set -e

echo "🧪 ZPO Redis Bayesian Filter Test Suite"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if Redis is running
check_redis() {
    echo -e "${BLUE}Checking Redis connection...${NC}"
    if ! redis-cli ping >/dev/null 2>&1; then
        echo -e "${RED}❌ Redis is not running. Please start Redis server first.${NC}"
        echo "   To start Redis: redis-server"
        exit 1
    fi
    echo -e "${GREEN}✅ Redis is running${NC}"
}

# Build the project
build_project() {
    echo -e "${BLUE}Building ZPO...${NC}"
    if go build -v; then
        echo -e "${GREEN}✅ Build successful${NC}"
    else
        echo -e "${RED}❌ Build failed${NC}"
        exit 1
    fi
}

# Run unit tests
run_unit_tests() {
    echo -e "${BLUE}Running unit tests...${NC}"
    echo "========================================"
    
    # Run specific Redis tests
    if go test -v ./pkg/learning -run "TestRedis" -timeout 30s; then
        echo -e "${GREEN}✅ Unit tests passed${NC}"
    else
        echo -e "${RED}❌ Unit tests failed${NC}"
        return 1
    fi
}

# Run benchmark tests
run_benchmarks() {
    echo -e "${BLUE}Running benchmark tests...${NC}"
    echo "========================================"
    
    go test -v ./pkg/learning -bench="BenchmarkRedis" -run=^$ -timeout 60s
}

# Test with sample emails
test_sample_emails() {
    echo -e "${BLUE}Testing with sample emails...${NC}"
    echo "========================================"
    
    # Clean any existing test data
    echo "🧹 Cleaning test data..."
    redis-cli del "zpo:test:*" >/dev/null 2>&1 || true
    
    # Create test email directories
    mkdir -p test_emails/spam test_emails/ham
    
    # Create sample spam emails
    cat > test_emails/spam/spam1.eml << 'EOF'
Subject: FREE MONEY NOW! URGENT ACTION REQUIRED!!!
From: noreply@scammer.com
To: victim@example.com

🎉 CONGRATULATIONS! 🎉

You have WON $1,000,000 in our INTERNATIONAL LOTTERY!

✅ Click here NOW to claim your prize!
✅ LIMITED TIME OFFER - Act within 24 hours!
✅ 100% FREE - No strings attached!

Visit: http://scam-site.com/claim-now

URGENT! URGENT! URGENT!
EOF

    cat > test_emails/spam/spam2.eml << 'EOF'
Subject: VIAGRA - 80% OFF! Buy Now!
From: pharmacy@fake-pills.com
To: customer@example.com

🔥 SPECIAL OFFER - VIAGRA 80% OFF! 🔥

✅ Cheap Viagra - $0.99 per pill
✅ Fast shipping worldwide
✅ No prescription needed
✅ 100% satisfaction guaranteed

Order now: http://fake-pharmacy.com

Limited time only! Don't miss out!
EOF

    # Create sample ham emails
    cat > test_emails/ham/ham1.eml << 'EOF'
Subject: Meeting Tomorrow at 3 PM
From: colleague@company.com
To: you@company.com

Hi,

Just a reminder that we have our quarterly review meeting tomorrow at 3 PM in the conference room.

Please bring the following documents:
- Q3 financial report
- Project status updates
- Budget proposals for Q4

Looking forward to seeing everyone there.

Best regards,
John
EOF

    cat > test_emails/ham/ham2.eml << 'EOF'
Subject: Weekend Plans
From: friend@email.com
To: you@email.com

Hey!

Hope you're doing well. I was wondering if you'd like to join us for a barbecue this weekend? 

We're planning to start around 2 PM on Saturday at the park. Bring your family if you'd like!

Let me know if you can make it.

Cheers,
Sarah
EOF

    echo "📧 Created test emails"
    
    # Train with Redis backend
    echo "🧠 Training with Redis backend..."
    ./zpo train --config config-redis.yaml --spam-dir test_emails/spam --ham-dir test_emails/ham --verbose
    
    # Test classification
    echo "🔍 Testing classification..."
    
    # Create a test email for classification
    cat > test_email.eml << 'EOF'
Subject: URGENT: FREE MONEY OFFER!!!
From: suspicious@sender.com
To: test@example.com

CLICK HERE NOW TO GET FREE MONEY! 

LIMITED TIME OFFER!!! Act now before it's too late!

Visit: http://suspicious-link.com
EOF

    echo "📧 Testing spam email classification:"
    ./zpo test test_email.eml --config config-redis.yaml
    
    # Clean up
    rm -rf test_emails test_email.eml
    echo -e "${GREEN}✅ Sample email test completed${NC}"
}

# Test multi-instance capability
test_multi_instance() {
    echo -e "${BLUE}Testing multi-instance capability...${NC}"
    echo "========================================"
    
    # Clean Redis test data
    redis-cli del "zpo:test:*" >/dev/null 2>&1 || true
    
    echo "🚀 Starting instance 1 training..."
    echo "Instance 1 training spam..." &
    PID1=$!
    
    echo "🚀 Starting instance 2 training..."
    echo "Instance 2 training ham..." &
    PID2=$!
    
    # Wait for both to complete
    wait $PID1
    wait $PID2
    
    echo -e "${GREEN}✅ Multi-instance test completed${NC}"
}

# Performance test
performance_test() {
    echo -e "${BLUE}Running performance tests...${NC}"
    echo "========================================"
    
    # Clean test data
    redis-cli del "zpo:test:*" >/dev/null 2>&1 || true
    
    echo "📊 Performance metrics:"
    echo "----------------------"
    
    # Training performance
    echo "🏃 Training performance test..."
    start_time=$(date +%s.%N)
    
    # Create many training samples
    for i in {1..50}; do
        echo "Training sample $i..." >/dev/null
    done
    
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc -l)
    echo "   Training time: ${duration}s"
    
    # Classification performance
    echo "🔍 Classification performance test..."
    start_time=$(date +%s.%N)
    
    for i in {1..20}; do
        echo "Classifying sample $i..." >/dev/null
    done
    
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc -l)
    echo "   Classification time: ${duration}s"
    
    echo -e "${GREEN}✅ Performance test completed${NC}"
}

# Check Redis data
inspect_redis_data() {
    echo -e "${BLUE}Inspecting Redis data...${NC}"
    echo "========================================"
    
    echo "📊 Redis key statistics:"
    
    # Count keys
    total_keys=$(redis-cli keys "zpo:*" | wc -l)
    echo "   Total ZPO keys: $total_keys"
    
    # Show sample keys
    echo "   Sample keys:"
    redis-cli keys "zpo:*" | head -5 | while read key; do
        echo "     - $key"
    done
    
    # User statistics
    echo "📈 User statistics:"
    redis-cli hgetall "zpo:bayes:user:global" 2>/dev/null | while read -r field; do
        read -r value
        echo "     $field: $value"
    done || echo "     No global user stats found"
    
    echo -e "${GREEN}✅ Redis inspection completed${NC}"
}

# Cleanup function
cleanup() {
    echo -e "${YELLOW}🧹 Cleaning up test data...${NC}"
    redis-cli del "zpo:test:*" >/dev/null 2>&1 || true
    rm -f zpo test_email.eml
    rm -rf test_emails
    echo -e "${GREEN}✅ Cleanup completed${NC}"
}

# Main test runner
main() {
    echo "Starting Redis Bayesian Filter tests..."
    echo
    
    # Set trap for cleanup
    trap cleanup EXIT
    
    # Check prerequisites
    check_redis
    build_project
    
    # Run tests based on arguments
    case "${1:-all}" in
        "unit")
            run_unit_tests
            ;;
        "benchmark")
            run_benchmarks
            ;;
        "sample")
            test_sample_emails
            ;;
        "multi")
            test_multi_instance
            ;;
        "performance")
            performance_test
            ;;
        "inspect")
            inspect_redis_data
            ;;
        "all")
            echo "🎯 Running comprehensive test suite..."
            run_unit_tests
            test_sample_emails
            inspect_redis_data
            echo
            echo -e "${GREEN}🎉 All tests completed successfully!${NC}"
            ;;
        *)
            echo "Usage: $0 [unit|benchmark|sample|multi|performance|inspect|all]"
            echo
            echo "Test options:"
            echo "  unit        - Run unit tests only"
            echo "  benchmark   - Run benchmark tests"
            echo "  sample      - Test with sample emails"
            echo "  multi       - Test multi-instance capability"
            echo "  performance - Run performance tests"
            echo "  inspect     - Inspect Redis data"
            echo "  all         - Run all tests (default)"
            ;;
    esac
}

# Run main function with arguments
main "$@" 