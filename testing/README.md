# Testing Documentation

This directory contains comprehensive testing scripts and benchmarks for ZPAM spam filter.

## 🚀 Quick Testing

```bash
# Quick performance test (recommended first step)
./testing/benchmark_simple.sh

# Run unit tests
go test ./pkg/...

# Full integration tests
./testing/test_redis_bayes.sh all
```

## 📊 Available Test Scripts

### `benchmark_simple.sh`
**Purpose:** Quick performance benchmark with minimal setup
**Usage:**
```bash
./testing/benchmark_simple.sh [num_emails]
```

**Features:**
- Tests processing speed with configurable email count (default: 1000)
- Measures emails/second and average processing time
- Uses sample emails from `examples/` directory
- No external dependencies required
- Generates timestamped results

**Sample Output:**
```
Processing 1000 emails...
Completed in 1.68 seconds
Average: 816 emails/second (1.68ms per email)
```

### `benchmark_zpam.sh`
**Purpose:** Comprehensive benchmark suite with multiple configurations
**Usage:**
```bash
./testing/benchmark_zpam.sh [config_file]
```

**Features:**
- Tests multiple configuration profiles
- Memory usage monitoring
- Concurrent processing tests
- Redis backend performance
- Detailed performance metrics
- HTML report generation

**Test Scenarios:**
- Single-threaded processing
- Multi-instance parallel processing
- Redis vs file backend comparison
- Memory leak detection
- Load testing with varying email sizes

### `test_redis_bayes.sh`
**Purpose:** Redis Bayesian learning integration tests
**Usage:**
```bash
./testing/test_redis_bayes.sh [unit|integration|benchmark|all]
```

**Test Types:**

#### Unit Tests (`unit`)
- Redis connection handling
- Bayesian tokenization
- OSB (Orthogonal Sparse Bigrams) generation
- Model training and classification
- Error handling and recovery

#### Integration Tests (`integration`)
- Full Redis deployment via Docker
- Multi-instance learning synchronization
- Training data persistence
- Real-world email classification
- Performance under load

#### Benchmark Tests (`benchmark`)
- Learning speed with large datasets
- Classification accuracy metrics
- Memory usage optimization
- Redis cluster performance
- Concurrent instance testing

#### All Tests (`all`)
Runs complete test suite with detailed reporting

## 🧪 Test Data

### Sample Emails
Located in `examples/` directory:
- `clean_email.eml` - Clean business email
- `test_headers.eml` - Email with various headers

### Milter Test Emails
Located in `milter/emails/`:
- `01_clean_business.eml` - Professional email
- `02_clean_personal.eml` - Personal correspondence  
- `03_clean_newsletter.eml` - Legitimate newsletter
- `04_clean_marketing.eml` - Marketing email
- `05_clean_update.eml` - System notification
- `06_spam_phishing.eml` - Phishing attempt
- `07_spam_getrich.eml` - Get-rich-quick scheme
- `08_spam_lottery.eml` - Lottery scam
- `09_spam_drugs.eml` - Pharmaceutical spam
- `10_spam_prize.eml` - Prize notification scam

## 📈 Performance Benchmarks

### Latest Results
Based on recent testing runs:

| Configuration | Emails/Second | Processing Time | Memory Usage |
|--------------|---------------|-----------------|--------------|
| Redis Backend | 816 | 1.68ms | 2.31MB |
| File Backend | 720+ | <2ms | <5MB |
| Multi-Instance (2x) | 492 | 2.03s total | 4.6MB |

### Historical Results
Results are stored in timestamped files:
- `benchmark_results_YYYYMMDD_HHMMSS.txt`

## 🔧 Test Configuration

### Redis Requirements
For Redis-based tests:
```bash
# Start Redis (via Docker)
docker run -d --name redis-test -p 6379:6379 redis:alpine

# Or use Docker Compose
docker-compose -f docker/docker-compose.test.yml up -d redis
```

### Environment Variables
```bash
# Optional: Custom Redis URL
export REDIS_URL="redis://localhost:6379"

# Optional: Custom test email count  
export TEST_EMAIL_COUNT=1000

# Optional: Enable verbose output
export ZPAM_TEST_VERBOSE=1
```

## 🐛 Troubleshooting

### Common Issues

#### Redis Connection Failed
```bash
# Check Redis status
docker ps | grep redis

# Restart Redis
docker restart redis-test
```

#### Permission Denied on Scripts
```bash
# Make scripts executable
chmod +x testing/*.sh
```

#### Out of Memory During Tests
```bash
# Reduce test email count
./testing/benchmark_simple.sh 100

# Monitor memory usage
./testing/benchmark_zpam.sh --memory-profile
```

### Test Dependencies

#### Required
- Go 1.21+
- Access to Redis (for integration tests)
- Docker (for containerized tests)

#### Optional
- `jq` for JSON parsing in scripts
- `bc` for floating-point calculations
- `curl` for HTTP endpoint testing

## 📝 Adding New Tests

### Creating a New Test Script

1. **Follow naming convention:** `test_feature_name.sh`
2. **Add executable permissions:** `chmod +x testing/test_feature_name.sh`
3. **Include usage documentation:** Add `--help` flag support
4. **Use consistent output format:** Follow existing script patterns
5. **Add error handling:** Validate prerequisites and inputs

### Example Test Script Template

```bash
#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Configuration
TEST_NAME="Feature Test"
DEFAULT_COUNT=100

# Usage function
usage() {
    echo "Usage: $0 [options]"
    echo "Options:"
    echo "  -c, --count NUM    Number of test iterations (default: $DEFAULT_COUNT)"
    echo "  -h, --help         Show this help message"
    exit 1
}

# Main test function
run_test() {
    local count=$1
    echo "Running $TEST_NAME with $count iterations..."
    
    # Test implementation here
    
    echo "✅ $TEST_NAME completed successfully"
}

# Parse arguments and run
main() {
    local count=$DEFAULT_COUNT
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -c|--count) count="$2"; shift 2 ;;
            -h|--help) usage ;;
            *) echo "Unknown option: $1"; usage ;;
        esac
    done
    
    run_test "$count"
}

main "$@"
```

## 🔍 Test Results Analysis

### Performance Metrics
- **Throughput:** Emails processed per second
- **Latency:** Average processing time per email
- **Memory:** Peak memory usage during processing
- **Accuracy:** Spam detection precision and recall

### Regression Testing
- Compare results against baseline performance
- Track performance trends over time
- Identify performance regressions automatically
- Generate performance reports

## 🚀 Continuous Integration

### GitHub Actions Integration
```yaml
# Example CI test step
- name: Run ZPAM Tests
  run: |
    ./testing/test_redis_bayes.sh unit
    ./testing/benchmark_simple.sh 100
```

### Docker-based Testing
```bash
# Run tests in isolated environment
docker-compose -f docker/docker-compose.test.yml --profile test up --abort-on-container-exit
```

---

**For more information, see the main [README.md](../README.md) and [development roadmap](../docs/ROADMAP.md).** 