# ZPO - High-Performance Spam Filter 🫏

[![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Speed](https://img.shields.io/badge/Speed-%3C2ms_per_email-brightgreen.svg)](#performance)
[![Docker](https://img.shields.io/badge/Docker-Ready-blue.svg)](docker/)
[![Redis](https://img.shields.io/badge/Redis-Bayesian-red.svg)](#redis-bayesian-learning)

ZPO is a lightning-fast, production-ready spam filter that processes emails in under 2ms with enterprise-grade accuracy. Named after a donkey - it's **free, fast, and reliable**.

## ✨ Key Features

- **⚡ Ultra-Fast**: Sub-2ms processing with Redis-backed Bayesian learning
- **🎯 High Accuracy**: SpamAssassin-inspired scoring with balanced detection
- **🚀 Scalable**: Multi-instance deployment with shared Redis backend
- **🐳 Production Ready**: Full Docker deployment with monitoring
- **📁 Auto-Sorting**: Intelligent spam filtering and email management
- **🔧 Milter Integration**: Real-time filtering for Postfix/Sendmail
- **🧠 Machine Learning**: Advanced Bayesian classification with OSB tokenization
- **🆓 Open Source**: Completely free with no licensing restrictions

## 🚀 Quick Start

### Docker Deployment (Recommended)

```bash
# Clone repository
git clone <repository-url>
cd zpo

# Start ZPO with Redis backend
docker-compose -f docker/docker-compose.yml up -d

# Train the Bayesian filter (if you have training data)
docker-compose -f docker/docker-compose.yml exec zpo ./zpo train \
  --config /app/config.yaml \
  --spam-dir /path/to/spam \
  --ham-dir /path/to/ham

# Test spam detection
docker-compose -f docker/docker-compose.yml exec zpo ./zpo filter \
  --config /app/config.yaml \
  --input /path/to/test/email.eml
```

### Native Installation

```bash
# Install dependencies
go mod tidy

# Build ZPO
go build -o zpo

# Test a single email
./zpo filter --input email.eml

# Run performance benchmark
./testing/benchmark_simple.sh
```

## 📊 Performance Benchmarks

**Latest Benchmark Results:**
- **1000 emails** processed in **1.68 seconds**
- **816 emails/second** sustained throughput
- **1.68ms average** processing time per email
- **Multi-instance capability** with shared Redis learning

| Test Configuration | Emails/Second | Avg Time/Email | Notes |
|-------------------|---------------|----------------|--------|
| Redis Backend (1000) | **816** | 1.68ms | Single instance |
| Multi-Instance (2x500) | **492** | 2.03s total | Parallel processing |
| File Backend (1000) | 720+ | <2ms | Fallback mode |

*See [Testing Documentation](testing/README.md) for complete benchmark results.*

## 🧠 Redis Bayesian Learning

ZPO features a sophisticated Bayesian filter with Redis backend:

- **OSB Tokenization**: Rspamd-compatible Orthogonal Sparse Bigrams
- **Multi-Instance Learning**: Shared model across unlimited instances
- **High Performance**: 4,867+ learned tokens in just 2.31MB memory
- **Per-User Models**: Individual user classification support
- **Real-time Updates**: Continuous learning from new data

```bash
# Train the model
./zpo train --spam-dir spam_emails/ --ham-dir clean_emails/

# Reset and retrain
./zpo train --spam-dir spam_emails/ --ham-dir clean_emails/ --reset
```

## 📧 Email Classification

ZPO uses a balanced 1-5 scoring system:

| Score | Classification | Action | Raw Score Range |
|-------|---------------|--------|-----------------|
| 1-2 | **Clean** | Keep in inbox | 0.0 - 15.0 |
| 3 | **Questionable** | Keep (review) | 15.0 - 25.0 |
| 4-5 | **Spam** | Move to spam | 25.0+ |

### Example Classifications

```bash
# Clean business email → Score: 1/5 (0.32 raw)
# Newsletter → Score: 3/5 (11.52 raw)  
# Spam → Score: 5/5 (44.52 raw)
```

## 🔧 Configuration

ZPO supports multiple configuration profiles:

- **`config.yaml`**: Production configuration with Redis backend  
- **`config-redis.yaml`**: Redis-optimized settings
- **`config-fast.yaml`**: High-speed configuration (DNS disabled)
- **`config-cached.yaml`**: Balanced performance with caching
- **`config-dnstest.yaml`**: DNS testing and development

### Key Settings

```yaml
# Learning backend selection
learning:
  backend: "redis"  # or "file"
  
# Redis configuration
redis:
  redis_url: "redis://redis:6379"
  
# Scoring thresholds
filter:
  spam_threshold: 4  # Move to spam folder
  reject_threshold: 5  # Reject email
```

## 🐳 Docker Deployment

Complete containerized deployment with Docker Compose:

```bash
# Production deployment
docker-compose -f docker/docker-compose.yml up -d

# Testing environment
docker-compose -f docker/docker-compose.test.yml --profile test up

# Scale to multiple instances
docker-compose -f docker/docker-compose.yml up -d --scale zpo=3
```

**Features:**
- Multi-stage builds for minimal image size
- Redis persistence and clustering
- Health checks and monitoring
- Production security best practices

*See [Docker Deployment Guide](docker/README.md) for complete setup instructions.*

## 🧪 Testing & Benchmarking

Run comprehensive tests and benchmarks:

```bash
# Quick performance test
./testing/benchmark_simple.sh

# Comprehensive benchmarks (when available)
./testing/benchmark_zpo.sh

# Unit and integration tests
go test ./pkg/...
```

**Test Coverage:**
- Unit tests for all components
- Integration tests with Redis
- Performance benchmarks with testing scripts
- Multi-instance testing via Docker
- Docker environment validation

*See [Testing Documentation](testing/README.md) for detailed test procedures.*

## 📮 Milter Integration

Real-time email filtering for mail servers:

```bash
# Start milter server
./zpo milter --config config.yaml
```

**Postfix Integration:**
```conf
# /etc/postfix/main.cf
smtpd_milters = inet:localhost:7357
non_smtpd_milters = inet:localhost:7357
milter_default_action = accept
```

**Features:**
- Sub-5ms processing per email
- Concurrent connection handling
- Automatic spam header injection
- Unix socket support
- Production monitoring

## 📚 Documentation

| Document | Description |
|----------|-------------|
| [Docker Deployment](docker/README.md) | Complete containerization guide |
| [Testing Guide](testing/README.md) | Testing procedures and benchmarks |
| [Roadmap](docs/ROADMAP.md) | Future development plans |
| [Custom Plugins](docs/custom_plugins.md) | Plugin development guide |
| [TensorFlow Setup](docs/tensorflow_setup.md) | ML model integration |

## 🛠️ Development

### Project Structure

ZPO has been reorganized for better maintainability and development workflow:

```
zpo/
├── cmd/                 # CLI commands and subcommands
├── pkg/                 # Core packages (filter, learning, plugins)
├── docker/              # Complete Docker deployment setup
│   ├── docker-compose.yml       # Production deployment
│   ├── docker-compose.test.yml  # Testing environment
│   └── README.md                # Docker documentation
├── testing/             # Testing scripts and benchmarks
│   ├── benchmark_simple.sh      # Quick performance tests
│   └── README.md                # Testing documentation
├── docs/                # Project documentation
│   ├── ROADMAP.md               # Development roadmap
│   ├── custom_plugins.md        # Plugin development guide
│   └── tensorflow_setup.md      # ML integration guide
├── examples/            # Sample emails for testing
├── milter/              # Milter integration examples
└── config*.yaml         # Configuration profiles
```

**Improvements:**
- Organized Docker deployment files in `docker/` directory
- Consolidated testing scripts in `testing/` directory  
- Clear separation of deployment, testing, and documentation
- Self-contained examples and configurations

### Building

```bash
# Development build
go build -o zpo

# Production build with optimizations
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o zpo

# Docker build
docker build -f docker/Dockerfile -t zpo:latest .
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests (`go test ./pkg/...` and `./testing/benchmark_simple.sh`)
4. Commit changes (`git commit -m 'Add amazing feature'`)
5. Push to branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**ZPO - Because spam filtering should be as reliable as a donkey! 🫏**

*Fast, Free, and Production-Ready Email Security* 