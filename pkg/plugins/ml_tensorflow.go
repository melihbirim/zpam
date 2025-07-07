package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// TensorFlowClassifier implements MLBackend interface using TensorFlow Serving or Python
type TensorFlowClassifier struct {
	modelInfo   *ModelInfo
	initialized bool
	config      map[string]interface{}
	client      *http.Client

	// Configuration options
	modelPath    string
	servingURL   string
	modelName    string
	modelVersion string
	usePython    bool
	pythonScript string
	timeout      time.Duration
}

// TensorFlowRequest represents a request to TensorFlow Serving
type TensorFlowRequest struct {
	Instances [][]float64 `json:"instances"`
}

// TensorFlowResponse represents a response from TensorFlow Serving
type TensorFlowResponse struct {
	Predictions [][]float64 `json:"predictions"`
	Error       string      `json:"error,omitempty"`
}

// NewTensorFlowClassifier creates a new TensorFlow classifier
func NewTensorFlowClassifier() MLBackend {
	return &TensorFlowClassifier{
		client:       &http.Client{Timeout: 30 * time.Second},
		modelName:    "spam_classifier",
		modelVersion: "1",
		timeout:      30 * time.Second,
	}
}

// Initialize sets up the TensorFlow classifier with configuration
func (tf *TensorFlowClassifier) Initialize(config map[string]interface{}) error {
	tf.config = config

	// Extract configuration
	if modelPath, ok := config["model_path"].(string); ok && modelPath != "" {
		tf.modelPath = modelPath
	} else {
		tf.modelPath = "models/spam_classifier"
	}

	if servingURL, ok := config["serving_url"].(string); ok && servingURL != "" {
		tf.servingURL = servingURL
	} else {
		tf.servingURL = "http://localhost:8501" // Default TensorFlow Serving port
	}

	if modelName, ok := config["model_name"].(string); ok && modelName != "" {
		tf.modelName = modelName
	}

	if version, ok := config["model_version"].(string); ok && version != "" {
		tf.modelVersion = version
	}

	if timeout, ok := config["timeout"].(float64); ok && timeout > 0 {
		tf.timeout = time.Duration(timeout) * time.Millisecond
		tf.client.Timeout = tf.timeout
	}

	// Check if Python fallback should be used
	if usePython, ok := config["use_python"].(bool); ok {
		tf.usePython = usePython
	}

	if pythonScript, ok := config["python_script"].(string); ok && pythonScript != "" {
		tf.pythonScript = pythonScript
	} else {
		tf.pythonScript = "scripts/tf_inference.py"
	}

	// Validate setup
	if err := tf.validateSetup(); err != nil {
		return fmt.Errorf("TensorFlow setup validation failed: %v", err)
	}

	// Initialize model info
	tf.modelInfo = &ModelInfo{
		Name:      "TensorFlow Spam Classifier",
		Version:   tf.getModelVersion(),
		Type:      "deep_learning",
		Framework: "TensorFlow",
		Features:  25,
		Accuracy:  tf.getModelAccuracy(),
		LoadedAt:  time.Now(),
		FilePath:  tf.modelPath,
		FileSize:  tf.getModelSize(),
	}

	tf.initialized = true
	fmt.Printf("TensorFlow classifier initialized with serving URL: %s\n", tf.servingURL)

	return nil
}

// Predict runs inference using TensorFlow Serving or Python script
func (tf *TensorFlowClassifier) Predict(features []float64) (*MLPrediction, error) {
	if !tf.initialized {
		return nil, fmt.Errorf("TensorFlow classifier not initialized")
	}

	start := time.Now()

	// Validate input features
	if len(features) != 25 {
		return nil, fmt.Errorf("expected 25 features, got %d", len(features))
	}

	var prediction *MLPrediction
	var err error

	// Try TensorFlow Serving first, fallback to Python
	if tf.usePython || !tf.isServingAvailable() {
		prediction, err = tf.predictWithPython(features)
	} else {
		prediction, err = tf.predictWithServing(features)
		if err != nil && tf.pythonFallbackAvailable() {
			// Fallback to Python if serving fails
			prediction, err = tf.predictWithPython(features)
		}
	}

	if err != nil {
		return nil, err
	}

	prediction.ModelName = tf.modelInfo.Name
	prediction.Features = len(features)
	prediction.ProcessTime = time.Since(start).Milliseconds()

	return prediction, nil
}

// predictWithServing uses TensorFlow Serving REST API
func (tf *TensorFlowClassifier) predictWithServing(features []float64) (*MLPrediction, error) {
	// Prepare request
	request := TensorFlowRequest{
		Instances: [][]float64{features},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Make HTTP request
	url := fmt.Sprintf("%s/v1/models/%s/versions/%s:predict",
		tf.servingURL, tf.modelName, tf.modelVersion)

	resp, err := tf.client.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("serving error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response TensorFlowResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if response.Error != "" {
		return nil, fmt.Errorf("serving error: %s", response.Error)
	}

	if len(response.Predictions) == 0 || len(response.Predictions[0]) < 2 {
		return nil, fmt.Errorf("invalid prediction format")
	}

	// Extract scores
	hamScore := response.Predictions[0][0]
	spamScore := response.Predictions[0][1]

	return tf.createPrediction(hamScore, spamScore), nil
}

// predictWithPython uses Python script for inference
func (tf *TensorFlowClassifier) predictWithPython(features []float64) (*MLPrediction, error) {
	// Create temporary input file
	tempFile, err := os.CreateTemp("", "tf_input_*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write features to file
	featureData := map[string][]float64{"features": features}
	if err := json.NewEncoder(tempFile).Encode(featureData); err != nil {
		return nil, fmt.Errorf("failed to write features: %v", err)
	}
	tempFile.Close()

	// Run Python script
	cmd := exec.Command("python3", tf.pythonScript,
		"--model", tf.modelPath,
		"--input", tempFile.Name(),
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Python inference failed: %v", err)
	}

	// Parse Python output
	var result struct {
		Predictions []float64 `json:"predictions"`
		Error       string    `json:"error,omitempty"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Python output: %v", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("Python error: %s", result.Error)
	}

	if len(result.Predictions) < 2 {
		return nil, fmt.Errorf("invalid Python prediction format")
	}

	hamScore := result.Predictions[0]
	spamScore := result.Predictions[1]

	return tf.createPrediction(hamScore, spamScore), nil
}

// GetModelInfo returns information about the loaded model
func (tf *TensorFlowClassifier) GetModelInfo() (*ModelInfo, error) {
	if !tf.initialized {
		return nil, fmt.Errorf("TensorFlow classifier not initialized")
	}
	return tf.modelInfo, nil
}

// IsHealthy checks if the classifier is ready
func (tf *TensorFlowClassifier) IsHealthy() error {
	if !tf.initialized {
		return fmt.Errorf("TensorFlow classifier not initialized")
	}

	// Test prediction with dummy data
	dummyFeatures := make([]float64, 25)
	for i := range dummyFeatures {
		dummyFeatures[i] = 0.5
	}

	_, err := tf.Predict(dummyFeatures)
	if err != nil {
		return fmt.Errorf("health check failed: %v", err)
	}

	return nil
}

// Cleanup releases resources
func (tf *TensorFlowClassifier) Cleanup() error {
	tf.initialized = false
	return nil
}

// validateSetup checks if TensorFlow serving or Python is available
func (tf *TensorFlowClassifier) validateSetup() error {
	// Check if model path exists
	if tf.modelPath != "" {
		if _, err := os.Stat(tf.modelPath); os.IsNotExist(err) {
			return fmt.Errorf("model path does not exist: %s", tf.modelPath)
		}
	}

	// Check TensorFlow Serving availability
	if !tf.usePython {
		if !tf.isServingAvailable() && !tf.pythonFallbackAvailable() {
			return fmt.Errorf("neither TensorFlow Serving nor Python fallback available")
		}
	} else {
		if !tf.pythonFallbackAvailable() {
			return fmt.Errorf("Python fallback not available")
		}
	}

	return nil
}

// isServingAvailable checks if TensorFlow Serving is available
func (tf *TensorFlowClassifier) isServingAvailable() bool {
	// Quick health check to TensorFlow Serving
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("%s/v1/models/%s", tf.servingURL, tf.modelName)

	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// pythonFallbackAvailable checks if Python inference is available
func (tf *TensorFlowClassifier) pythonFallbackAvailable() bool {
	// Check if Python script exists
	if _, err := os.Stat(tf.pythonScript); os.IsNotExist(err) {
		return false
	}

	// Check if Python3 is available
	_, err := exec.LookPath("python3")
	return err == nil
}

// createPrediction creates MLPrediction from scores
func (tf *TensorFlowClassifier) createPrediction(hamScore, spamScore float64) *MLPrediction {
	// Normalize scores (softmax if needed)
	total := hamScore + spamScore
	if total > 0 {
		hamScore /= total
		spamScore /= total
	}

	isSpam := spamScore > hamScore
	confidence := spamScore
	if hamScore > spamScore {
		confidence = hamScore
	}

	return &MLPrediction{
		IsSpam:     isSpam,
		Confidence: confidence,
		SpamScore:  spamScore,
		HamScore:   hamScore,
	}
}

// getModelVersion attempts to extract model version
func (tf *TensorFlowClassifier) getModelVersion() string {
	// Try to read version from model metadata
	versionFile := filepath.Join(tf.modelPath, "version.txt")
	if data, err := os.ReadFile(versionFile); err == nil {
		return strings.TrimSpace(string(data))
	}

	return tf.modelVersion
}

// getModelAccuracy attempts to extract model accuracy
func (tf *TensorFlowClassifier) getModelAccuracy() float64 {
	// Try to read accuracy from model metadata
	accuracyFile := filepath.Join(tf.modelPath, "accuracy.txt")
	if data, err := os.ReadFile(accuracyFile); err == nil {
		if accuracy, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
			return accuracy
		}
	}

	// Default accuracy estimate for TensorFlow models
	return 0.94
}

// getModelSize calculates model file size
func (tf *TensorFlowClassifier) getModelSize() int64 {
	var totalSize int64

	if tf.modelPath == "" {
		return 0
	}

	err := filepath.Walk(tf.modelPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0
	}

	return totalSize
}
