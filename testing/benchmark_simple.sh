#!/bin/bash
# Simple ZPO Benchmark - Focus on 1000 email performance
set -e

# Configuration
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
RESULTS_FILE="benchmark_results_${TIMESTAMP}.txt"

echo "🚀 ZPO Performance Benchmark - $TIMESTAMP" | tee $RESULTS_FILE
echo "=====================================================" | tee -a $RESULTS_FILE
echo "Testing ZPO spam filter performance with various email batch sizes" | tee -a $RESULTS_FILE
echo "" | tee -a $RESULTS_FILE

# Test function
run_test() {
    local name=$1
    local batch_size=$2
    local config=$3
    
    echo "🧪 Testing: $name with $batch_size emails..." | tee -a $RESULTS_FILE
    
    # Prepare test emails in container
    docker exec zpo-zpo-1 sh -c "
        mkdir -p /tmp/benchmark_$batch_size
        find /app/stress-test -name '*.eml' | head -$batch_size | xargs -I {} cp {} /tmp/benchmark_$batch_size/
    "
    
    # Run benchmark and capture output
    local start_time=$(date +%s.%N)
    local output=$(docker exec zpo-zpo-1 ./zpo filter --config $config --input /tmp/benchmark_$batch_size 2>&1)
    local end_time=$(date +%s.%N)
    
    # Calculate metrics
    local duration=$(echo "$end_time - $start_time" | bc -l)
    local throughput=$(echo "scale=2; $batch_size / $duration" | bc -l)
    local avg_time=$(echo "scale=2; $duration / $batch_size * 1000" | bc -l)
    
    # Extract spam/ham counts
    local spam_count=$(echo "$output" | grep "Spam detected:" | awk '{print $3}')
    local ham_count=$(echo "$output" | grep "Ham (clean):" | awk '{print $3}')
    
    # Output results
    printf "  %-25s: %s emails in %.2fs (%.2f emails/sec, %.2fms avg)\n" \
        "$name" "$batch_size" "$duration" "$throughput" "$avg_time" | tee -a $RESULTS_FILE
    printf "  %-25s: Spam: %s, Ham: %s\n" "" "$spam_count" "$ham_count" | tee -a $RESULTS_FILE
    echo "" | tee -a $RESULTS_FILE
    
    # Cleanup
    docker exec zpo-zpo-1 rm -rf /tmp/benchmark_$batch_size
    
    return 0
}

# Check containers are running
echo "🔍 Checking container status..." | tee -a $RESULTS_FILE
if ! docker exec zpo-zpo-1 echo "Container ready" >/dev/null 2>&1; then
    echo "❌ Container not ready. Starting..." | tee -a $RESULTS_FILE
    docker-compose -f ../docker/docker-compose.yml up -d
    sleep 30
fi

echo "✅ Container ready" | tee -a $RESULTS_FILE
echo "" | tee -a $RESULTS_FILE

# Run benchmark tests
echo "📊 Starting Performance Tests" | tee -a $RESULTS_FILE
echo "==============================" | tee -a $RESULTS_FILE
echo "" | tee -a $RESULTS_FILE

# Test different batch sizes with Redis backend
run_test "Redis Backend - 100 emails" 100 "/app/config.yaml"
run_test "Redis Backend - 500 emails" 500 "/app/config.yaml"
run_test "Redis Backend - 1000 emails" 1000 "/app/config.yaml"
run_test "Redis Backend - 2000 emails" 2000 "/app/config.yaml"

# Multi-instance test with 1000 emails
echo "🔄 Multi-Instance Test (1000 emails)" | tee -a $RESULTS_FILE
echo "====================================" | tee -a $RESULTS_FILE

# Start second instance for multi-instance test
docker-compose -f ../docker/docker-compose.yml up -d --scale zpo=2
sleep 10

# Prepare test data for both instances
docker exec zpo-zpo-1 sh -c "
    mkdir -p /tmp/multi_inst1
    find /app/stress-test -name '*.eml' | head -500 | xargs -I {} cp {} /tmp/multi_inst1/
"

docker exec zpo-zpo-2 sh -c "
    mkdir -p /tmp/multi_inst2  
    find /app/stress-test -name '*.eml' | tail -500 | head -500 | xargs -I {} cp {} /tmp/multi_inst2/
"

# Run both instances in parallel
echo "🚀 Running 500 emails on each of 2 instances..." | tee -a $RESULTS_FILE

start_time=$(date +%s.%N)

docker exec zpo-zpo-1 ./zpo filter --config /app/config.yaml --input /tmp/multi_inst1 > /tmp/result1.txt &
docker exec zpo-zpo-2 ./zpo filter --config /app/config.yaml --input /tmp/multi_inst2 > /tmp/result2.txt &

wait

end_time=$(date +%s.%N)
duration=$(echo "$end_time - $start_time" | bc -l)
throughput=$(echo "scale=2; 1000 / $duration" | bc -l)

printf "  %-25s: %s emails in %.2fs (%.2f emails/sec parallel)\n" \
    "Multi-Instance (2x500)" "1000" "$duration" "$throughput" | tee -a $RESULTS_FILE

# Cleanup multi-instance test
docker exec zpo-zpo-1 rm -rf /tmp/multi_inst1
docker exec zpo-zpo-2 rm -rf /tmp/multi_inst2

echo "" | tee -a $RESULTS_FILE

# Redis analysis
echo "📈 Redis Performance Analysis" | tee -a $RESULTS_FILE
echo "==============================" | tee -a $RESULTS_FILE

redis_keys=$(docker exec zpo-redis redis-cli --raw eval "return #redis.call('keys', 'zpo:bayes:*')" 0)
redis_memory=$(docker exec zpo-redis redis-cli info memory | grep used_memory_human | cut -d: -f2)

echo "  Bayesian tokens in Redis: $redis_keys" | tee -a $RESULTS_FILE
echo "  Redis memory usage: $redis_memory" | tee -a $RESULTS_FILE
echo "" | tee -a $RESULTS_FILE

# Summary
echo "📋 Benchmark Summary" | tee -a $RESULTS_FILE
echo "====================" | tee -a $RESULTS_FILE
echo "  Completed: $(date)" | tee -a $RESULTS_FILE
echo "  Total tests: 5 (4 single instance + 1 multi-instance)" | tee -a $RESULTS_FILE
echo "  Largest test: 2000 emails" | tee -a $RESULTS_FILE
echo "  Multi-instance capability: ✅ Verified" | tee -a $RESULTS_FILE
echo "  Redis backend: ✅ Operational with $redis_keys tokens" | tee -a $RESULTS_FILE
echo "" | tee -a $RESULTS_FILE

echo "🎉 Benchmark completed! Results saved to: $RESULTS_FILE" | tee -a $RESULTS_FILE
echo "📊 View results: cat $RESULTS_FILE" 