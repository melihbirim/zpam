# TensorFlow Integration for ZPAM

ZPAM includes comprehensive TensorFlow support for advanced spam classification using deep learning models.

## Setup

### Prerequisites
1. Python 3.7+ with TensorFlow installed
2. (Optional) TensorFlow Serving for production deployments

### Installation

```bash
# Install Python dependencies
pip install -r requirements.txt

# Or install manually
pip install tensorflow>=2.12.0 numpy>=1.21.0
```

### Configuration

Update your `config.yaml` to enable TensorFlow:

```yaml
plugins:
  machine_learning:
    enabled: true
    weight: 2.5
    settings:
      type: tensorflow
      model_path: models/spam_classifier
      confidence_threshold: 0.7
      # TensorFlow Serving (production)
      serving_url: http://localhost:8501
      model_name: spam_classifier
      model_version: "1"
      # Python fallback (development)
      use_python: true
      python_script: scripts/tf_inference.py
```

## Usage Modes

### 1. TensorFlow Serving (Production)

TensorFlow Serving provides high-performance model serving via REST API.

```bash
# Start TensorFlow Serving
docker run -p 8501:8501 \
  --mount type=bind,source=/path/to/models,target=/models \
  tensorflow/serving \
  --model_config_file=/models/models.config

# Test the plugin
./zpam plugins test examples/test_headers.eml
```

### 2. Python Script (Development)

Use the included Python script for inference:

```bash
# Create a sample model for testing
python3 scripts/tf_inference.py --model models/spam_classifier --create-sample

# Test model info
python3 scripts/tf_inference.py --model models/spam_classifier --model-info

# Test inference directly
echo '{"features": [0.1, 0.2, ...]}' > test_features.json
python3 scripts/tf_inference.py --model models/spam_classifier --input test_features.json
```

## Model Formats

### SavedModel (Recommended)
```
models/spam_classifier/
├── saved_model.pb
├── variables/
│   ├── variables.data-00000-of-00001
│   └── variables.index
└── assets/
```

### Keras H5 Format
```
models/spam_classifier.h5
```

## Feature Engineering

The TensorFlow backend expects 25 numerical features:

1. **Text Statistics** (5 features)
   - Email length
   - Word count
   - Character count
   - Line count
   - Caps ratio

2. **Spam/Ham Indicators** (8 features)
   - Spam word frequency
   - Ham word frequency
   - Spam/Ham word ratio
   - Subject spam score
   - Financial terms count
   - Urgency indicators
   - Pharmaceutical terms
   - Technology scam terms

3. **Structural Features** (6 features)
   - HTML tag count
   - URL count
   - Email address count
   - Phone number count
   - Attachment count
   - Image count

4. **Reputation Features** (3 features)
   - Domain reputation
   - Sender reputation
   - Header analysis score

5. **Advanced Features** (3 features)
   - Lexical diversity
   - Readability score
   - Sentiment analysis

## Model Training

### Sample Training Script

```python
import tensorflow as tf
import numpy as np
from sklearn.model_selection import train_test_split

# Load your spam/ham dataset
# X should be shape (n_samples, 25)
# y should be shape (n_samples, 2) for [ham, spam] probabilities

def create_model():
    model = tf.keras.Sequential([
        tf.keras.layers.Dense(64, activation='relu', input_shape=(25,)),
        tf.keras.layers.Dropout(0.3),
        tf.keras.layers.Dense(32, activation='relu'),
        tf.keras.layers.Dropout(0.2),
        tf.keras.layers.Dense(16, activation='relu'),
        tf.keras.layers.Dense(2, activation='softmax')
    ])
    
    model.compile(
        optimizer='adam',
        loss='categorical_crossentropy',
        metrics=['accuracy']
    )
    
    return model

# Train the model
model = create_model()
model.fit(X_train, y_train, 
          validation_data=(X_val, y_val),
          epochs=100, batch_size=32)

# Save for ZPAM
tf.saved_model.save(model, 'models/spam_classifier')

# Save metadata
with open('models/spam_classifier/accuracy.txt', 'w') as f:
    f.write(str(model.evaluate(X_test, y_test)[1]))

with open('models/spam_classifier/version.txt', 'w') as f:
    f.write('1.0.0')
```

## Testing

```bash
# Test with sample email
./zpam plugins test examples/test_headers.eml

# Test specific plugin
./zpam plugins test-one machine_learning examples/test_headers.eml

# Plugin statistics
./zpam plugins stats
```

## Troubleshooting

### Common Issues

1. **"TensorFlow model not found"**
   ```bash
   # Create sample model
   python3 scripts/tf_inference.py --model models/spam_classifier --create-sample
   ```

2. **"Python inference failed"**
   ```bash
   # Check Python dependencies
   pip install tensorflow numpy
   ```

3. **"Serving error"**
   ```bash
   # Check TensorFlow Serving
   curl http://localhost:8501/v1/models/spam_classifier
   ```

### Performance Tuning

1. **Use TensorFlow Serving** for production (faster than Python)
2. **Model optimization**: Use TensorFlow Lite for mobile/edge deployments
3. **Batch processing**: Process multiple emails simultaneously
4. **GPU acceleration**: Use CUDA for training large models

### Fallback Behavior

If TensorFlow fails:
1. First tries TensorFlow Serving
2. Falls back to Python script
3. If both fail, disables the plugin
4. Logs detailed error messages

## Model Management

### Version Control
```yaml
# config.yaml
machine_learning:
  settings:
    model_version: "2"  # Use specific version
```

### A/B Testing
```yaml
# Compare models
machine_learning_v1:
  settings:
    model_path: models/spam_classifier_v1
machine_learning_v2:
  settings:
    model_path: models/spam_classifier_v2
```

### Monitoring
```bash
# Check model performance
./zpam plugins stats | grep machine_learning

# Detailed logs
tail -f logs/zpam.log | grep tensorflow
```

## Integration Examples

### With Existing Filters
```yaml
plugins:
  spamassassin:
    enabled: true
    weight: 2.0
  machine_learning:
    enabled: true
    weight: 2.5  # Higher weight for ML
  custom_rules:
    enabled: true
    weight: 1.5
```

### Production Deployment
```bash
# Start TensorFlow Serving
docker run -d --name tf-serving \
  -p 8501:8501 \
  -v $(pwd)/models:/models \
  tensorflow/serving:latest \
  --model_config_file=/models/models.config

# Configure ZPAM
./zpam milter --config config-production.yaml
``` 