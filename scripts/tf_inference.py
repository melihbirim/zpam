#!/usr/bin/env python3
"""
TensorFlow Inference Script for ZPO Spam Filter

This script loads a TensorFlow model and performs inference for spam classification.
It can be used as a backend for the ZPO TensorFlow plugin.

Usage:
    python3 tf_inference.py --model <model_path> --input <input_file> --output json
"""

import argparse
import json
import os
import sys
import logging
from typing import List, Dict, Any, Tuple
import numpy as np

try:
    import tensorflow as tf
except ImportError:
    print(json.dumps({
        "error": "TensorFlow not installed. Install with: pip install tensorflow"
    }))
    sys.exit(1)

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class SpamClassifier:
    """TensorFlow-based spam classifier"""
    
    def __init__(self, model_path: str):
        self.model_path = model_path
        self.model = None
        self.input_signature = None
        self.output_signature = None
        
    def load_model(self) -> bool:
        """Load the TensorFlow model"""
        try:
            if os.path.isdir(self.model_path):
                # Load SavedModel format
                self.model = tf.saved_model.load(self.model_path)
                logger.info(f"Loaded SavedModel from {self.model_path}")
                
                # Get serving signature
                if hasattr(self.model, 'signatures'):
                    serving_default = self.model.signatures.get('serving_default')
                    if serving_default:
                        self.input_signature = serving_default.structured_input_signature
                        self.output_signature = serving_default.structured_outputs
                        logger.info("Found serving_default signature")
                
            elif self.model_path.endswith('.h5'):
                # Load Keras model
                self.model = tf.keras.models.load_model(self.model_path)
                logger.info(f"Loaded Keras model from {self.model_path}")
                
            else:
                logger.error(f"Unsupported model format: {self.model_path}")
                return False
                
            return True
            
        except Exception as e:
            logger.error(f"Failed to load model: {e}")
            return False
    
    def predict(self, features: List[float]) -> Tuple[float, float]:
        """
        Predict spam probability for given features
        
        Args:
            features: List of 25 numerical features
            
        Returns:
            Tuple of (ham_probability, spam_probability)
        """
        try:
            # Convert to numpy array and reshape
            input_data = np.array(features, dtype=np.float32).reshape(1, -1)
            
            # Validate input shape
            if input_data.shape[1] != 25:
                raise ValueError(f"Expected 25 features, got {input_data.shape[1]}")
            
            # Run inference
            if hasattr(self.model, 'signatures'):
                # SavedModel with signatures
                serving_fn = self.model.signatures['serving_default']
                
                # Find input tensor name
                input_key = list(serving_fn.structured_input_signature[1].keys())[0]
                input_tensor = tf.constant(input_data)
                
                predictions = serving_fn(**{input_key: input_tensor})
                
                # Extract output
                output_key = list(predictions.keys())[0]
                output = predictions[output_key].numpy()
                
            else:
                # Keras model or direct callable
                output = self.model(input_data).numpy()
            
            # Handle different output formats
            if output.shape[-1] == 2:
                # Binary classification with 2 outputs [ham, spam]
                ham_prob = float(output[0][0])
                spam_prob = float(output[0][1])
            elif output.shape[-1] == 1:
                # Binary classification with single output (spam probability)
                spam_prob = float(output[0][0])
                ham_prob = 1.0 - spam_prob
            else:
                raise ValueError(f"Unexpected output shape: {output.shape}")
            
            # Apply softmax if probabilities don't sum to 1
            total = ham_prob + spam_prob
            if abs(total - 1.0) > 0.01:
                ham_prob = np.exp(ham_prob) / (np.exp(ham_prob) + np.exp(spam_prob))
                spam_prob = 1.0 - ham_prob
            
            return float(ham_prob), float(spam_prob)
            
        except Exception as e:
            logger.error(f"Prediction failed: {e}")
            raise
    
    def get_model_info(self) -> Dict[str, Any]:
        """Get information about the loaded model"""
        info = {
            "model_path": self.model_path,
            "model_type": "unknown",
            "input_shape": "unknown",
            "output_shape": "unknown"
        }
        
        try:
            if hasattr(self.model, 'signatures'):
                info["model_type"] = "SavedModel"
                if self.input_signature:
                    input_spec = list(self.input_signature[1].values())[0]
                    info["input_shape"] = str(input_spec.shape)
                if self.output_signature:
                    output_spec = list(self.output_signature.values())[0]
                    info["output_shape"] = str(output_spec.shape)
            else:
                info["model_type"] = "Keras"
                if hasattr(self.model, 'input_shape'):
                    info["input_shape"] = str(self.model.input_shape)
                if hasattr(self.model, 'output_shape'):
                    info["output_shape"] = str(self.model.output_shape)
                    
        except Exception as e:
            logger.warning(f"Could not extract model info: {e}")
            
        return info


def create_sample_model(model_path: str) -> bool:
    """Create a sample TensorFlow model for testing"""
    try:
        # Create a simple neural network for spam classification
        model = tf.keras.Sequential([
            tf.keras.layers.Dense(64, activation='relu', input_shape=(25,)),
            tf.keras.layers.Dropout(0.3),
            tf.keras.layers.Dense(32, activation='relu'),
            tf.keras.layers.Dropout(0.2),
            tf.keras.layers.Dense(16, activation='relu'),
            tf.keras.layers.Dense(2, activation='softmax')  # Ham, Spam
        ])
        
        # Compile the model
        model.compile(
            optimizer='adam',
            loss='categorical_crossentropy',
            metrics=['accuracy']
        )
        
        # Initialize with random weights
        dummy_input = np.random.random((1, 25))
        _ = model(dummy_input)
        
        # Save the model
        os.makedirs(os.path.dirname(model_path), exist_ok=True)
        
        if model_path.endswith('.h5'):
            model.save(model_path)
        else:
            # Save as SavedModel
            tf.saved_model.save(model, model_path)
        
        logger.info(f"Sample model created at {model_path}")
        return True
        
    except Exception as e:
        logger.error(f"Failed to create sample model: {e}")
        return False


def main():
    parser = argparse.ArgumentParser(description='TensorFlow Inference for ZPO')
    parser.add_argument('--model', required=True, help='Path to TensorFlow model')
    parser.add_argument('--input', required=True, help='Input JSON file with features')
    parser.add_argument('--output', default='json', choices=['json', 'text'], 
                       help='Output format')
    parser.add_argument('--create-sample', action='store_true', 
                       help='Create a sample model for testing')
    parser.add_argument('--model-info', action='store_true',
                       help='Show model information')
    
    args = parser.parse_args()
    
    # Create sample model if requested
    if args.create_sample:
        if create_sample_model(args.model):
            print(json.dumps({"status": "Sample model created successfully"}))
        else:
            print(json.dumps({"error": "Failed to create sample model"}))
        return
    
    try:
        # Initialize classifier
        classifier = SpamClassifier(args.model)
        
        # Load model
        if not classifier.load_model():
            print(json.dumps({"error": "Failed to load model"}))
            return
        
        # Show model info if requested
        if args.model_info:
            info = classifier.get_model_info()
            print(json.dumps({"model_info": info}))
            return
        
        # Load input features
        with open(args.input, 'r') as f:
            input_data = json.load(f)
        
        if 'features' not in input_data:
            print(json.dumps({"error": "Input file must contain 'features' array"}))
            return
        
        features = input_data['features']
        
        if len(features) != 25:
            print(json.dumps({
                "error": f"Expected 25 features, got {len(features)}"
            }))
            return
        
        # Run prediction
        ham_prob, spam_prob = classifier.predict(features)
        
        # Output results
        result = {
            "predictions": [ham_prob, spam_prob],
            "ham_probability": ham_prob,
            "spam_probability": spam_prob,
            "is_spam": spam_prob > ham_prob,
            "confidence": max(ham_prob, spam_prob)
        }
        
        if args.output == 'json':
            print(json.dumps(result))
        else:
            print(f"Ham: {ham_prob:.4f}, Spam: {spam_prob:.4f}")
            print(f"Classification: {'SPAM' if spam_prob > ham_prob else 'HAM'}")
            print(f"Confidence: {max(ham_prob, spam_prob):.4f}")
            
    except Exception as e:
        error_result = {"error": str(e)}
        if args.output == 'json':
            print(json.dumps(error_result))
        else:
            print(f"Error: {e}")


if __name__ == "__main__":
    main() 