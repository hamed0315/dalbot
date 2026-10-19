package services

import (
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"sync"
)

// NeuralNet یک پرسپترون چندلایه (MLP) با یک لایه پنهان
type NeuralNet struct {
	mu        sync.RWMutex
	InputSize int        // ۱۰۰۰ ویژگی TF-IDF
	HiddenSize int       // ۶۴ نورون
	W1        [][]float64 // وزن‌های ورودی → پنهان (InputSize x HiddenSize)
	B1        []float64   // بایاس لایه پنهان
	W2        []float64   // وزن‌های پنهان → خروجی (HiddenSize → 1)
	B2        float64     // بایاس خروجی
	Trained   bool
}

var globalNN *NeuralNet

// NewNeuralNet یک شبکهٔ جدید با وزن‌های تصادفی می‌سازد
func NewNeuralNet(inputSize, hiddenSize int) *NeuralNet {
	nn := &NeuralNet{
		InputSize:  inputSize,
		HiddenSize: hiddenSize,
		W1:        make([][]float64, inputSize),
		B1:        make([]float64, hiddenSize),
		W2:        make([]float64, hiddenSize),
		B2:        rand.Float64()*0.1 - 0.05,
	}

	for i := 0; i < inputSize; i++ {
		nn.W1[i] = make([]float64, hiddenSize)
		for j := 0; j < hiddenSize; j++ {
			nn.W1[i][j] = rand.Float64()*0.2 - 0.1 // Xavier init ساده
		}
	}
	for j := 0; j < hiddenSize; j++ {
		nn.B1[j] = rand.Float64()*0.1 - 0.05
		nn.W2[j] = rand.Float64()*0.2 - 0.1
	}

	return nn
}

// sigmoid تابع فعال‌سازی
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// sigmoidDerivative مشتق sigmoid
func sigmoidDerivative(x float64) float64 {
	s := sigmoid(x)
	return s * (1 - s)
}

// Forward انتشار رو به جلو، خروجی احتمال اسپم را برمی‌گرداند
func (nn *NeuralNet) Forward(features []float64) float64 {
	nn.mu.RLock()
	defer nn.mu.RUnlock()

	if len(features) != nn.InputSize || !nn.Trained {
		return 0.0
	}

	// لایه پنهان: W1^T * input + b1
	hidden := make([]float64, nn.HiddenSize)
	for j := 0; j < nn.HiddenSize; j++ {
		sum := nn.B1[j]
		for i := 0; i < nn.InputSize; i++ {
			sum += features[i] * nn.W1[i][j]
		}
		hidden[j] = sigmoid(sum)
	}

	// خروجی: W2^T * hidden + b2
	var output float64 = nn.B2
	for j := 0; j < nn.HiddenSize; j++ {
		output += hidden[j] * nn.W2[j]
	}
	return sigmoid(output)
}

// Predict با آستانه مشخص (پیش‌فرض 0.85) تصمیم می‌گیرد
func (nn *NeuralNet) Predict(features []float64, threshold float64) bool {
	if threshold == 0 {
		threshold = 0.85
	}
	return nn.Forward(features) > threshold
}

// Train یک مرحله آموزش با یک نمونه (Stochastic Gradient Descent)
func (nn *NeuralNet) Train(features []float64, target float64, learningRate float64) {
	nn.mu.Lock()
	defer nn.mu.Unlock()

	if len(features) != nn.InputSize {
		return
	}

	// --- Forward ---
	hidden := make([]float64, nn.HiddenSize)
	for j := 0; j < nn.HiddenSize; j++ {
		sum := nn.B1[j]
		for i := 0; i < nn.InputSize; i++ {
			sum += features[i] * nn.W1[i][j]
		}
		hidden[j] = sigmoid(sum)
	}

	var output float64 = nn.B2
	for j := 0; j < nn.HiddenSize; j++ {
		output += hidden[j] * nn.W2[j]
	}
	predicted := sigmoid(output)

	// --- Backpropagation ---
	errorOutput := (predicted - target) * sigmoidDerivative(output)

	// گرادیان W2 و B2
	for j := 0; j < nn.HiddenSize; j++ {
		nn.W2[j] -= learningRate * errorOutput * hidden[j]
	}
	nn.B2 -= learningRate * errorOutput

	// گرادیان لایه پنهان
	for j := 0; j < nn.HiddenSize; j++ {
		errorHidden := errorOutput * nn.W2[j] * sigmoidDerivative(hidden[j])
		for i := 0; i < nn.InputSize; i++ {
			nn.W1[i][j] -= learningRate * errorHidden * features[i]
		}
		nn.B1[j] -= learningRate * errorHidden
	}
}

// SaveWeights وزن‌ها را در فایل JSON ذخیره می‌کند
func (nn *NeuralNet) SaveWeights(filename string) error {
	nn.mu.RLock()
	defer nn.mu.RUnlock()

	type weightsData struct {
		InputSize  int
		HiddenSize int
		W1         [][]float64
		B1         []float64
		W2         []float64
		B2         float64
	}

	data := weightsData{
		InputSize:  nn.InputSize,
		HiddenSize: nn.HiddenSize,
		W1:         nn.W1,
		B1:         nn.B1,
		W2:         nn.W2,
		B2:         nn.B2,
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(data)
}

// LoadWeights وزن‌ها را از فایل JSON بارگذاری می‌کند
func (nn *NeuralNet) LoadWeights(filename string) error {
	nn.mu.Lock()
	defer nn.mu.Unlock()

	type weightsData struct {
		InputSize  int
		HiddenSize int
		W1         [][]float64
		B1         []float64
		W2         []float64
		B2         float64
	}

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var data weightsData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return err
	}

	if data.InputSize != nn.InputSize || data.HiddenSize != nn.HiddenSize {
		return nil // ابعاد ناسازگار، چشم‌پوشی می‌کنیم
	}

	nn.W1 = data.W1
	nn.B1 = data.B1
	nn.W2 = data.W2
	nn.B2 = data.B2
	nn.Trained = true
	return nil
}
