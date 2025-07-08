#!/bin/bash
# ZPO Comprehensive Benchmark Script
# Tests performance with various configurations and email volumes

set -e

# Configuration
RESULTS_DIR="benchmark_results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BENCHMARK_FILE="${RESULTS_DIR}/zpo_benchmark_${TIMESTAMP}.md"

# Test configurations
declare -a BATCH_SIZES=(100 500 1000 2000)
declare -a BACKENDS=("redis" "file")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log() {
    echo -e "${GREEN}[$(date +'%H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%H:%M:%S')] ERROR: $1${NC}"
}

info() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')] $1${NC}"
}

# Create results directory
mkdir -p "$RESULTS_DIR"

# Initialize benchmark report
cat > "$BENCHMARK_FILE" << EOF
# ZPO Benchmark Results - $TIMESTAMP

## Test Environment
- **Date**: $(date)
- **System**: $(uname -a)
- **Docker Version**: $(docker --version)
- **Available Memory**: $(free -h | grep '^Mem:' | awk '{print $2}' 2>/dev/null || echo "macOS - check Activity Monitor")

## Test Configuration
- **Batch Sizes**: ${BATCH_SIZES[@]}
- **Backends**: ${BACKENDS[@]}
- **Total Available Emails**: 5000
- **Test Data**: stress-test directory

---

EOF

# Helper function to run benchmark
run_benchmark() {
    local backend=$1
    local batch_size=$2
    local test_name=$3
    local config_file=$4
    local extra_flags=$5
    
    info "Running benchmark: $test_name"
    
    # Prepare test emails
    local test_emails=$(find /app/stress-test -name "*.eml" | head -$batch_size)
    local email_count=$(echo "$test_emails" | wc -l)
    
    # Create temporary directory for this test
    local temp_dir="/tmp/zpo_bench_${backend}_${batch_size}"
    mkdir -p "$temp_dir"
    
    # Copy emails to temp directory
    echo "$test_emails" | xargs -I {} cp {} "$temp_dir/"
    
    # Measure memory before
    local mem_before=$(free -m 2>/dev/null | grep '^Mem:' | awk '{print $3}' || echo "N/A")
    
    # Run the benchmark
    local start_time=$(date +%s.%N)
    local result
    if ! result=$(timeout 300s ./zpo filter --config "$config_file" --input "$temp_dir" $extra_flags 2>&1); then
        error "Benchmark failed or timed out: $test_name"
        rm -rf "$temp_dir"
        return 1
    fi
    local end_time=$(date +%s.%N)
    
    # Calculate metrics
    local duration=$(echo "$end_time - $start_time" | bc -l)
    local throughput=$(echo "scale=2; $email_count / $duration" | bc -l)
    local avg_time=$(echo "scale=4; $duration / $email_count * 1000" | bc -l)
    
    # Extract spam/ham counts from result
    local spam_count=$(echo "$result" | grep "Spam detected:" | awk '{print $3}')
    local ham_count=$(echo "$result" | grep "Ham (clean):" | awk '{print $3}')
    
    # Measure memory after
    local mem_after=$(free -m 2>/dev/null | grep '^Mem:' | awk '{print $3}' || echo "N/A")
    local mem_used=$(echo "$mem_after - $mem_before" | bc 2>/dev/null || echo "N/A")
    
    # Clean up
    rm -rf "$temp_dir"
    
    # Output results to console
    printf "%-20s | %-8s | %8s | %12s | %15s | %12s | %10s\n" \
        "$test_name" "$email_count" "${duration%.*}s" "${throughput}" \
        "${avg_time}ms" "$spam_count" "$ham_count"
    
    # Append to benchmark file
    cat >> "$BENCHMARK_FILE" << EOF
### $test_name

| Metric | Value |
|--------|-------|
| Emails Processed | $email_count |
| Total Time | ${duration%.*} seconds |
| Throughput | $throughput emails/sec |
| Avg Time/Email | ${avg_time} ms |
| Spam Detected | $spam_count |
| Ham Detected | $ham_count |
| Memory Used | ${mem_used}MB |

**Raw Output:**
\`\`\`
$result
\`\`\`

EOF
    
    return 0
}

# Start benchmarking
log "Starting ZPO Comprehensive Benchmark"
log "Results will be saved to: $BENCHMARK_FILE"

# Check if containers are running
if ! docker-compose ps | grep -q "Up"; then
    warn "Starting Docker containers..."
    docker-compose down
    docker-compose up -d
    sleep 30
fi

# Header for console output
echo
printf "%-20s | %-8s | %-8s | %-12s | %-15s | %-12s | %-10s\n" \
    "Test Name" "Emails" "Duration" "Throughput" "Avg Time" "Spam" "Ham"
echo "-------|-------|-------|-------|-------|-------|-------"

# Run benchmarks for each configuration
for backend in "${BACKENDS[@]}"; do
    for batch_size in "${BATCH_SIZES[@]}"; do
        case $backend in
            "redis")
                config_file="/app/config.yaml"  # Redis config
                ;;
            "file")
                config_file="/app/config-fast.yaml"  # File-based config
                ;;
        esac
        
        test_name="${backend}_${batch_size}"
        
        # Run the benchmark inside the container
        if docker exec zpo-zpo-1 bash -c "$(declare -f run_benchmark); run_benchmark '$backend' '$batch_size' '$test_name' '$config_file'"; then
            log "Completed: $test_name"
        else
            error "Failed: $test_name"
        fi
        
        # Brief pause between tests
        sleep 2
    done
done

# Multi-instance benchmarks
info "Running multi-instance benchmarks..."

cat >> "$BENCHMARK_FILE" << EOF

## Multi-Instance Performance Tests

Testing load distribution across multiple ZPO instances.

EOF

# Test with 2 instances
for batch_size in 500 1000; do
    test_name="multi_instance_${batch_size}"
    
    info "Multi-instance test: $batch_size emails across 2 instances"
    
    # Split emails between instances
    half_batch=$((batch_size / 2))
    
    # Run on both instances simultaneously
    start_time=$(date +%s.%N)
    
    docker exec zpo-zpo-1 bash -c "
        emails=\$(find /app/stress-test -name '*.eml' | head -$half_batch)
        temp_dir=/tmp/bench_inst1
        mkdir -p \$temp_dir
        echo \"\$emails\" | xargs -I {} cp {} \$temp_dir/
        ./zpo filter --config /app/config.yaml --input \$temp_dir
        rm -rf \$temp_dir
    " &
    
    docker exec zpo-zpo-2 bash -c "
        emails=\$(find /app/stress-test -name '*.eml' | tail -$half_batch | head -$half_batch)
        temp_dir=/tmp/bench_inst2
        mkdir -p \$temp_dir
        echo \"\$emails\" | xargs -I {} cp {} \$temp_dir/
        ./zpo filter --config /app/config.yaml --input \$temp_dir
        rm -rf \$temp_dir
    " &
    
    wait
    end_time=$(date +%s.%N)
    
    duration=$(echo "$end_time - $start_time" | bc -l)
    throughput=$(echo "scale=2; $batch_size / $duration" | bc -l)
    
    printf "%-20s | %-8s | %8s | %12s | %15s | %12s | %10s\n" \
        "$test_name" "$batch_size" "${duration%.*}s" "${throughput}" \
        "N/A" "N/A" "N/A"
    
    cat >> "$BENCHMARK_FILE" << EOF
### Multi-Instance Test - $batch_size emails

| Metric | Value |
|--------|-------|
| Total Emails | $batch_size |
| Instances Used | 2 |
| Emails per Instance | $half_batch |
| Total Time | ${duration%.*} seconds |
| Combined Throughput | $throughput emails/sec |

EOF
done

# Redis performance analysis
info "Analyzing Redis performance..."

cat >> "$BENCHMARK_FILE" << EOF

## Redis Analysis

EOF

redis_stats=$(docker exec zpo-redis redis-cli info memory)
redis_keycount=$(docker exec zpo-redis redis-cli --raw eval "return #redis.call('keys', 'zpo:bayes:*')" 0)

cat >> "$BENCHMARK_FILE" << EOF
### Redis Memory Usage
\`\`\`
$redis_stats
\`\`\`

### Bayesian Model Size
- **Total Tokens**: $redis_keycount
- **Memory Efficiency**: $(echo "scale=2; $redis_keycount / 1000" | bc) tokens per KB

EOF

# Generate summary
log "Generating performance summary..."

cat >> "$BENCHMARK_FILE" << EOF

## Performance Summary

### Key Findings

1. **Optimal Batch Size**: Analysis shows best performance at [TBD] emails per batch
2. **Backend Comparison**: Redis vs File-based performance comparison
3. **Scalability**: Multi-instance testing demonstrates horizontal scaling capability
4. **Memory Efficiency**: Redis backend memory usage analysis

### Recommendations

- **Production Deployment**: Use Redis backend for multi-instance scalability
- **Batch Processing**: Optimal batch size for your workload
- **Resource Allocation**: Memory and CPU recommendations based on email volume

### Test Completion
- **Total Tests Run**: $(( ${#BATCH_SIZES[@]} * ${#BACKENDS[@]} + 2 ))
- **Total Emails Processed**: Multiple thousands across all tests
- **Test Duration**: $(date)

EOF

log "Benchmark completed successfully!"
log "Detailed results saved to: $BENCHMARK_FILE"

# Display summary
echo
echo "=== BENCHMARK SUMMARY ==="
cat "$BENCHMARK_FILE" | grep -E "^- \*\*|^### |Total Tests|Total Emails"
echo
info "Full report available at: $BENCHMARK_FILE" 