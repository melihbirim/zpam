package learning

// BayesianClassifier defines the interface for Bayesian classifiers
type BayesianClassifier interface {
	TrainSpam(subject, body, user string) error
	TrainHam(subject, body, user string) error
	ClassifyText(subject, body, user string) (float64, error)
	Reset(user string) error
}

// WordFrequencyAdapter wraps WordFrequency to implement BayesianClassifier interface
type WordFrequencyAdapter struct {
	*WordFrequency
}

// NewWordFrequencyAdapter creates a new adapter for WordFrequency
func NewWordFrequencyAdapter(wf *WordFrequency) BayesianClassifier {
	return &WordFrequencyAdapter{WordFrequency: wf}
}

// TrainSpam trains the filter on spam content (user parameter ignored for file-based storage)
func (wfa *WordFrequencyAdapter) TrainSpam(subject, body, user string) error {
	return wfa.WordFrequency.TrainSpam(subject, body)
}

// TrainHam trains the filter on ham content (user parameter ignored for file-based storage)
func (wfa *WordFrequencyAdapter) TrainHam(subject, body, user string) error {
	return wfa.WordFrequency.TrainHam(subject, body)
}

// ClassifyText classifies text (user parameter ignored for file-based storage)
func (wfa *WordFrequencyAdapter) ClassifyText(subject, body, user string) (float64, error) {
	return wfa.WordFrequency.ClassifyText(subject, body), nil
}

// Reset resets training data (user parameter ignored for file-based storage)
func (wfa *WordFrequencyAdapter) Reset(user string) error {
	wfa.WordFrequency.Reset()
	return nil
}

// Ensure both implementations satisfy the interface
var _ BayesianClassifier = (*WordFrequencyAdapter)(nil) // File-based implementation
var _ BayesianClassifier = (*RedisBayesianFilter)(nil)  // Redis implementation
