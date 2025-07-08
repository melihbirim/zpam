# Docker Deployment Guide

Complete containerized deployment guide for ZPO spam filter with Redis backend, monitoring, and production-ready configurations.

## 🚀 Quick Start

```bash
# Clone and start ZPO with Redis
git clone <repository-url>
cd zpo

# Production deployment
docker-compose -f docker/docker-compose.yml up -d

# Test the deployment
docker-compose -f docker/docker-compose.yml exec zpo ./zpo filter --input examples/clean_email.eml
```

## 📁 Docker Files Overview

### `docker-compose.yml`
**Purpose:** Production deployment with Redis backend
**Services:**
- `zpo`: Main spam filter service
- `redis`: Redis server for Bayesian learning
- `redis-commander`: Optional Redis web UI

### `docker-compose.test.yml`  
**Purpose:** Testing environment with isolated containers
**Services:**
- `zpo-test`: ZPO with test configuration
- `redis-test`: Isolated Redis for testing
- `test-runner`: Automated test execution

### `Dockerfile`
**Purpose:** Production ZPO container image
**Features:**
- Multi-stage build for minimal size
- Security-hardened Alpine Linux base
- Non-root user execution
- Health checks included

### `Dockerfile.test`
**Purpose:** Testing container with development tools
**Features:**
- Extended tooling for testing
- Volume mounts for live development
- Debug symbols included
- Test data pre-loaded

## 🏗️ Production Deployment

### Single Instance Deployment

```bash
# Start production services
docker-compose -f docker/docker-compose.yml up -d

# Check service status
docker-compose -f docker/docker-compose.yml ps

# View logs
docker-compose -f docker/docker-compose.yml logs -f zpo
```

**Services Started:**
- ZPO spam filter on port 8080
- Redis server on port 6379
- Redis Commander UI on port 8081 (optional)

### Multi-Instance Scaling

```bash
# Scale ZPO to 3 instances
docker-compose -f docker/docker-compose.yml up -d --scale zpo=3

# Verify scaling
docker-compose -f docker/docker-compose.yml ps zpo
```

**Features:**
- Shared Redis backend across all instances
- Automatic load balancing
- Zero-downtime scaling
- Shared learning model

### Load Balancer Setup

```yaml
# Add to docker-compose.yml
nginx:
  image: nginx:alpine
  ports:
    - "80:80"
  volumes:
    - ./nginx.conf:/etc/nginx/nginx.conf
  depends_on:
    - zpo
```

**Nginx Configuration Example:**
```nginx
upstream zpo_backend {
    server zpo:8080;
    # Add more instances as needed
}

server {
    listen 80;
    location / {
        proxy_pass http://zpo_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 🧪 Testing Environment

### Development Testing

```bash
# Start test environment
docker-compose -f docker/docker-compose.test.yml --profile test up

# Run specific tests
docker-compose -f docker/docker-compose.test.yml --profile test run test-runner ./testing/benchmark_simple.sh

# Interactive testing
docker-compose -f docker/docker-compose.test.yml exec zpo-test bash
```

### Automated Test Pipeline

```bash
# Complete test suite
docker-compose -f docker/docker-compose.test.yml --profile test up --abort-on-container-exit

# Unit tests only
docker-compose -f docker/docker-compose.test.yml run --rm test-runner go test ./pkg/...

# Performance benchmarks
docker-compose -f docker/docker-compose.test.yml run --rm test-runner ./testing/benchmark_simple.sh 1000
```

## 🔧 Configuration

### Environment Variables

#### ZPO Configuration
```bash
# Core settings
ZPO_CONFIG_FILE=/app/config.yaml
ZPO_LOG_LEVEL=info
ZPO_REDIS_URL=redis://redis:6379

# Performance tuning
ZPO_MAX_WORKERS=4
ZPO_TIMEOUT_SECONDS=30
ZPO_BUFFER_SIZE=1000

# Feature flags
ZPO_ENABLE_METRICS=true
ZPO_ENABLE_TRACING=false
```

#### Redis Configuration
```bash
# Redis settings
REDIS_MAXMEMORY=256mb
REDIS_MAXMEMORY_POLICY=allkeys-lru
REDIS_SAVE_INTERVAL=300
```

### Volume Mounts

#### Production Mounts
```yaml
volumes:
  - ./config.yaml:/app/config.yaml:ro
  - ./custom_rules.yml:/app/custom_rules.yml:ro
  - zpo_models:/app/models
  - zpo_logs:/app/logs
```

#### Development Mounts
```yaml
volumes:
  - .:/app:rw
  - /app/vendor
  - go_cache:/go/pkg/mod
```

## 📊 Monitoring & Observability

### Health Checks

ZPO containers include built-in health checks:

```yaml
healthcheck:
  test: ["CMD", "./zpo", "health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

### Metrics Collection

```bash
# Enable Prometheus metrics
docker-compose -f docker/docker-compose.yml exec zpo ./zpo metrics --enable

# Access metrics endpoint
curl http://localhost:8080/metrics
```

### Log Management

```bash
# Follow logs from all services
docker-compose -f docker/docker-compose.yml logs -f

# Filter logs by service
docker-compose -f docker/docker-compose.yml logs -f zpo

# Export logs
docker-compose -f docker/docker-compose.yml logs --no-color > zpo-logs.txt
```

## 🔐 Security

### Container Security

**Non-root Execution:**
```dockerfile
# User setup in Dockerfile
RUN addgroup -g 1000 zpo && \
    adduser -D -s /bin/sh -u 1000 -G zpo zpo
USER zpo
```

**Security Scanning:**
```bash
# Scan for vulnerabilities
docker scan zpo:latest

# Security audit
docker-compose -f docker/docker-compose.yml config --services | xargs -I {} docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy {}
```

### Network Security

```yaml
# Isolated network
networks:
  zpo_network:
    driver: bridge
    internal: true
```

### Secrets Management

```yaml
# Using Docker secrets
secrets:
  redis_password:
    file: ./secrets/redis_password.txt
    
services:
  redis:
    secrets:
      - redis_password
```

## 🚀 Production Best Practices

### Resource Limits

```yaml
services:
  zpo:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

### Restart Policies

```yaml
services:
  zpo:
    restart: unless-stopped
  redis:
    restart: always
```

### Data Persistence

```yaml
volumes:
  redis_data:
    driver: local
  zpo_models:
    driver: local
    
services:
  redis:
    volumes:
      - redis_data:/data
```

## 🔄 Backup & Recovery

### Redis Backup

```bash
# Manual backup
docker-compose -f docker/docker-compose.yml exec redis redis-cli BGSAVE

# Automated backup script
#!/bin/bash
BACKUP_DIR="/backups/$(date +%Y%m%d_%H%M%S)"
docker-compose -f docker/docker-compose.yml exec redis redis-cli --rdb /tmp/dump.rdb
docker cp $(docker-compose -f docker/docker-compose.yml ps -q redis):/tmp/dump.rdb $BACKUP_DIR/
```

### Model Backup

```bash
# Backup trained models
docker-compose -f docker/docker-compose.yml exec zpo tar -czf /tmp/models_backup.tar.gz /app/models/
docker cp $(docker-compose -f docker/docker-compose.yml ps -q zpo):/tmp/models_backup.tar.gz ./backups/
```

## 📈 Performance Tuning

### Redis Optimization

```yaml
# Redis performance tuning
redis:
  command: >
    redis-server
    --maxmemory 256mb
    --maxmemory-policy allkeys-lru
    --save 900 1
    --tcp-keepalive 60
```

### ZPO Optimization

```yaml
# ZPO performance environment
environment:
  - GOMAXPROCS=4
  - GOMEMLIMIT=256MiB
  - ZPO_BUFFER_SIZE=2000
  - ZPO_WORKER_COUNT=8
```

## 🐛 Troubleshooting

### Common Issues

#### Container Won't Start
```bash
# Check logs for errors
docker-compose -f docker/docker-compose.yml logs zpo

# Verify configuration
docker-compose -f docker/docker-compose.yml config

# Check resource availability
docker system df
docker system prune # if needed
```

#### Redis Connection Issues
```bash
# Test Redis connectivity
docker-compose -f docker/docker-compose.yml exec zpo redis-cli -h redis ping

# Check Redis logs
docker-compose -f docker/docker-compose.yml logs redis

# Restart Redis
docker-compose -f docker/docker-compose.yml restart redis
```

#### Performance Issues
```bash
# Monitor resource usage
docker stats

# Check container health
docker-compose -f docker/docker-compose.yml ps

# Run performance benchmark
docker-compose -f docker/docker-compose.yml exec zpo ./testing/benchmark_simple.sh
```

### Debug Mode

```bash
# Enable debug logging
docker-compose -f docker/docker-compose.yml exec zpo ./zpo --log-level debug

# Interactive debugging
docker-compose -f docker/docker-compose.yml exec zpo bash
```

## 🚀 Deployment Scenarios

### Development Environment
```bash
# Quick development setup
docker-compose -f docker/docker-compose.test.yml up -d
# - Live code reloading
# - Development tools included
# - Test data pre-loaded
```

### Staging Environment
```bash
# Production-like testing
docker-compose -f docker/docker-compose.yml up -d
# - Production configuration
# - Real Redis persistence
# - Performance monitoring
```

### Production Environment
```bash
# Full production deployment
docker-compose -f docker/docker-compose.yml up -d --scale zpo=3
# - Multiple ZPO instances
# - Redis clustering
# - Load balancing
# - Monitoring & alerting
```

## 📚 Additional Resources

- [Main README](../README.md) - Project overview and features
- [Testing Guide](../testing/README.md) - Comprehensive testing procedures
- [Development Roadmap](../docs/ROADMAP.md) - Future features and improvements
- [Custom Plugins](../docs/custom_plugins.md) - Plugin development guide

---

**Need help?** Check the troubleshooting section above or open an issue on GitHub. 