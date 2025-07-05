# ZPO Word Frequency Learning

## Overview

ZPO now includes a **Bayesian-like word frequency learning system** that learns from spam and ham emails to improve detection accuracy. This feature embodies the Redis v1 philosophy: simple but powerful core functionality.

## Features

### 🧠 Learning Algorithm
- **Naive Bayes approach**: Calculates spam/ham probabilities based on word frequencies
- **Laplace smoothing**: Handles unseen words gracefully
- **Log probability calculation**: Avoids numerical underflow for long texts
- **Prior probability consideration**: Uses email count ratios as base probabilities

### 📚 Training System
- **Flexible training**: Train on spam and/or ham email directories
- **Incremental learning**: Add to existing models without retraining from scratch
- **Model persistence**: Save/load models as JSON files
- **High-speed training**: Process 6,000+ emails/second during training

### ⚙️ Configuration
```yaml
learning:
  enabled: true                    # Enable/disable learning
  model_path: "zpo-model.json"    # Model file location
  min_word_length: 3              # Filter short words
  max_word_length: 20             # Filter very long words
  case_sensitive: false           # Case handling
  spam_threshold: 0.7             # Classification threshold
  min_word_count: 2               # Minimum occurrences
  smoothing_factor: 1.0           # Laplace smoothing
  use_subject_words: true         # Learn from subjects
  use_body_words: true            # Learn from bodies
  use_header_words: false         # Learn from headers
  max_vocabulary_size: 10000      # Vocabulary limit
  auto_train: false               # Auto-train on classified emails
```

## Usage

### Training a Model

```bash
# Train on spam and ham directories
./zpo train --spam-dir path/to/spam --ham-dir path/to/ham

# Train with specific model path
./zpo train --spam-dir spam/ --ham-dir ham/ --model my-model.json

# Reset existing model and start fresh
./zpo train --spam-dir spam/ --ham-dir ham/ --reset

# Verbose training output
./zpo train --spam-dir spam/ --ham-dir ham/ --verbose
```

### Using the Trained Model

```bash
# Enable learning in config.yaml
learning:
  enabled: true

# Test emails with learning enabled
./zpo test email.eml --config config.yaml

# Benchmark with learning
./zpo benchmark --input emails/ --config config.yaml
```

## Performance Impact

### Before Learning
- **Average time**: 0.06ms per email
- **Throughput**: 66,059 emails/second
- **95th percentile**: 0.15ms

### With Learning Enabled
- **Average time**: 0.15ms per email  
- **Throughput**: 26,277 emails/second
- **95th percentile**: 0.43ms

**Impact**: ~2.5x slower but still **excellent performance** (well under 1ms target)

## Training Results Example

```
🧠 Word Frequency Learning Model
════════════════════════════════════════
Training Data:
  Spam emails: 300
  Ham emails: 700
  Spam words: 6,956
  Ham words: 20,832
  Vocabulary size: 221

📈 Top Spam Words:
   1. phishing        (1.000 spamminess)
   2. opportunity     (1.000 spamminess)
   3. congratulations (1.000 spamminess)
   4. rich            (1.000 spamminess)
   5. gift            (1.000 spamminess)

📉 Top Ham Words:
   1. track           (0.000 spamminess)
   2. report          (0.000 spamminess)
   3. budget          (0.000 spamminess)
   4. invitation      (0.000 spamminess)
   5. meeting         (0.000 spamminess)
```

## Integration

### Configuration Integration
- Added `LearningConfig` to main configuration system
- Seamless YAML configuration management
- Default settings optimized for performance

### Spam Filter Integration
- Automatic model loading when enabled
- Word frequency scoring integrated into spam calculation
- Configurable weight for learning contribution
- Thread-safe concurrent access

### CLI Integration
- New `train` command for model training
- Progress tracking and statistics display
- Model validation and error handling
- Flexible training options

## Technical Implementation

### Core Components

1. **`pkg/learning/bayes.go`**
   - WordFrequency struct with thread-safe operations
   - Bayesian classification algorithm
   - Model persistence (JSON format)
   - Statistics and analysis tools

2. **`cmd/train.go`**
   - Training command implementation
   - Directory traversal and email parsing
   - Progress reporting and error handling
   - Model management

3. **Integration Points**
   - Configuration system extension
   - Spam filter scoring integration
   - CLI command registration

### Algorithm Details

1. **Word Extraction**
   - Regex-based word extraction (`[a-zA-Z]{3,20}`)
   - Case normalization (configurable)
   - Deduplication per email
   - Subject/body/header selection

2. **Probability Calculation**
   ```
   P(spam|words) = P(words|spam) * P(spam) / P(words)
   
   Uses log probabilities to avoid underflow:
   log P(spam|words) = Σ log P(word|spam) + log P(spam)
   ```

3. **Smoothing**
   ```
   P(word|class) = (count(word,class) + α) / (total_words(class) + α * vocabulary_size)
   
   Where α = smoothing_factor (default: 1.0)
   ```

## Benefits

### Accuracy Improvements
- **Adaptive**: Learns from actual email patterns
- **Context-aware**: Considers word combinations and frequencies
- **Personalized**: Can be trained on specific email patterns
- **Evolving**: Model improves with more training data

### Operational Benefits
- **Fast training**: 6,000+ emails/second training speed
- **Minimal impact**: Still processes emails in < 0.5ms
- **Configurable**: Fine-tune learning parameters
- **Persistent**: Models survive restarts
- **Observable**: Rich statistics and word analysis

### Redis v1 Philosophy
- **Simple but powerful**: Easy to understand and configure
- **Performance-focused**: Minimal impact on core speed
- **Modular**: Can be disabled if not needed
- **Configurable**: Extensive options without code changes

## Future Enhancements

- **Auto-training**: Automatically retrain on classified emails
- **Model versioning**: Track and manage model versions
- **Federated learning**: Combine models from multiple sources
- **Advanced features**: N-grams, stemming, stop words
- **Real-time updates**: Update model during operation

---

*ZPO Word Frequency Learning - Simple, Fast, Effective* 