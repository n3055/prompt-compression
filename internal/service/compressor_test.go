package service

import (
	"reflect"
	"testing"

	"github.com/n3055/backend-project/internal/config"
)

func defaultCompressor() *Compressor {
	return NewCompressor(config.CompressorConfig{
		MaxKeywords:           8,
		MinWordLength:         3,
		PositionWeightBoost:   1.5,
		LengthBonusThreshold:  5,
		LengthBonusMultiplier: 1.2,
	})
}

func TestCompress_EmptyInput(t *testing.T) {
	c := defaultCompressor()
	result := c.Compress("")
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

func TestCompress_SingleWord(t *testing.T) {
	c := defaultCompressor()
	result := c.Compress("goroutines")
	if len(result) != 1 || result[0] != "goroutines" {
		t.Errorf("expected [goroutines], got %v", result)
	}
}

func TestCompress_RemovesStopWords(t *testing.T) {
	c := defaultCompressor()
	result := c.Compress("the quick brown fox jumps over the lazy dog")

	// "the" and "over" are stop words; "fox" and "dog" are 3 chars (min length).
	for _, token := range result {
		if token == "the" || token == "over" {
			t.Errorf("stop word %q should have been removed", token)
		}
	}
}

func TestCompress_RespectsMaxKeywords(t *testing.T) {
	c := NewCompressor(config.CompressorConfig{
		MaxKeywords:           3,
		MinWordLength:         2,
		PositionWeightBoost:   1.5,
		LengthBonusThreshold:  5,
		LengthBonusMultiplier: 1.2,
	})

	result := c.Compress("golang programming language concurrency goroutines channels interfaces structs")
	if len(result) > 3 {
		t.Errorf("expected at most 3 keywords, got %d: %v", len(result), result)
	}
}

func TestCompress_PositionBoost(t *testing.T) {
	c := NewCompressor(config.CompressorConfig{
		MaxKeywords:           3,
		MinWordLength:         3,
		PositionWeightBoost:   10.0, // Very high boost to make it obvious.
		LengthBonusThreshold:  100,  // Disable length bonus.
		LengthBonusMultiplier: 1.0,
	})

	// "kubernetes" is only in the first sentence (gets 10x position boost).
	// "middleware" is only in the middle sentence (gets NO position boost).
	// Result: kubernetes score = 1 * 10.0 = 10.0, middleware score = 1 * 1.0 = 1.0
	result := c.Compress("kubernetes deployment strategy. middleware handles requests. logging enabled.")

	if len(result) < 1 {
		t.Fatal("expected at least 1 result")
	}

	// kubernetes should appear before middleware due to massive position boost.
	kubeIdx, mwIdx := -1, -1
	for i, token := range result {
		if token == "kubernetes" {
			kubeIdx = i
		}
		if token == "middleware" {
			mwIdx = i
		}
	}

	if kubeIdx == -1 {
		t.Errorf("expected 'kubernetes' in results, got %v", result)
	}
	if mwIdx != -1 && kubeIdx > mwIdx {
		t.Errorf("expected 'kubernetes' before 'middleware' due to position boost, got %v", result)
	}
}

func TestCountTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"   ", 0},
		{"hello", 1},
		{"hello world", 2},
		{"one two three four five", 5},
	}

	for _, tt := range tests {
		got := CountTokens(tt.input)
		if got != tt.expected {
			t.Errorf("CountTokens(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestSavingsPercent(t *testing.T) {
	tests := []struct {
		original   int
		compressed int
		expected   float64
	}{
		{100, 20, 80.0},
		{0, 0, 0},
		{10, 10, 0},
		{1000, 100, 90.0},
	}

	for _, tt := range tests {
		got := SavingsPercent(tt.original, tt.compressed)
		if got != tt.expected {
			t.Errorf("SavingsPercent(%d, %d) = %f, want %f", tt.original, tt.compressed, got, tt.expected)
		}
	}
}

func TestDeduplicate(t *testing.T) {
	input := []string{"go", "channels", "go", "goroutines", "channels"}
	expected := []string{"go", "channels", "goroutines"}
	got := deduplicate(input)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("deduplicate(%v) = %v, want %v", input, got, expected)
	}
}
