// Package service contains the core business logic.
// This file implements the Compression Engine — the heart of the system.
package service

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/n3055/backend-project/internal/config"
)

// Compressor extracts and scores keywords from text.
type Compressor struct {
	cfg       config.CompressorConfig
	stopWords map[string]bool
}

// NewCompressor creates a new compression engine with the given config.
func NewCompressor(cfg config.CompressorConfig) *Compressor {
	return &Compressor{
		cfg:       cfg,
		stopWords: buildStopWordSet(),
	}
}

// scoredTerm holds a term and its computed relevance score.
type scoredTerm struct {
	term  string
	score float64
}

// Compress takes raw text and returns the top-K most important keywords.
//
// Algorithm:
//  1. Split text into sentences for positional scoring
//  2. Tokenize each sentence into words
//  3. Normalize (lowercase, strip punctuation)
//  4. Remove stop words and short words
//  5. Score each term: tf × position_weight × length_bonus
//  6. Deduplicate and sort by score
//  7. Return top-K terms
func (c *Compressor) Compress(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	sentences := splitSentences(text)
	totalSentences := len(sentences)

	// termFreq tracks how often each normalized term appears.
	termFreq := make(map[string]int)
	// termPosition tracks whether a term appears in first/last sentence.
	termPosition := make(map[string]bool)

	for i, sentence := range sentences {
		words := tokenize(sentence)
		isEdgeSentence := i == 0 || i == totalSentences-1

		for _, word := range words {
			normalized := normalize(word)
			if normalized == "" {
				continue
			}
			if len(normalized) < c.cfg.MinWordLength {
				continue
			}
			if c.stopWords[normalized] {
				continue
			}

			termFreq[normalized]++
			if isEdgeSentence {
				termPosition[normalized] = true
			}
		}
	}

	// Score each unique term.
	scored := make([]scoredTerm, 0, len(termFreq))
	for term, freq := range termFreq {
		score := float64(freq)

		// Position weight: terms in first/last sentence get a boost.
		if termPosition[term] {
			score *= c.cfg.PositionWeightBoost
		}

		// Length bonus: longer words tend to be more meaningful.
		if len(term) > c.cfg.LengthBonusThreshold {
			score *= c.cfg.LengthBonusMultiplier
		}

		scored = append(scored, scoredTerm{term: term, score: score})
	}

	// Sort by score descending, then alphabetically for determinism.
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].term < scored[j].term
	})

	// Take top-K keywords.
	limit := c.cfg.MaxKeywords
	if limit > len(scored) {
		limit = len(scored)
	}

	result := make([]string, limit)
	for i := 0; i < limit; i++ {
		result[i] = scored[i].term
	}

	return result
}

// CountTokens returns an approximate token count for the given text.
// Uses simple whitespace splitting as a fast approximation.
// For production LLM token counting, replace with tiktoken or similar.
func CountTokens(text string) int {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	return len(strings.Fields(text))
}

// SavingsPercent calculates the percentage of tokens saved.
func SavingsPercent(original, compressed int) float64 {
	if original == 0 {
		return 0
	}
	savings := float64(original-compressed) / float64(original) * 100
	return math.Round(savings*100) / 100 // Round to 2 decimal places.
}

// --- Internal helpers ---

// splitSentences splits text on sentence boundaries (., !, ?, newline).
func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for _, r := range text {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' || r == '\n' {
			s := strings.TrimSpace(current.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}

	// Remaining text that didn't end with a sentence delimiter.
	if s := strings.TrimSpace(current.String()); s != "" {
		sentences = append(sentences, s)
	}

	if len(sentences) == 0 {
		return []string{text}
	}

	return sentences
}

// tokenize splits a sentence into words on whitespace and punctuation boundaries.
func tokenize(sentence string) []string {
	return strings.FieldsFunc(sentence, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_'
	})
}

// normalize lowercases and trims a word.
func normalize(word string) string {
	return strings.ToLower(strings.TrimSpace(word))
}

// buildStopWordSet returns a set of common English stop words to filter out.
func buildStopWordSet() map[string]bool {
	words := []string{
		// Articles
		"a", "an", "the",
		// Pronouns
		"i", "me", "my", "we", "our", "you", "your", "he", "him", "his",
		"she", "her", "it", "its", "they", "them", "their", "this", "that",
		"these", "those", "who", "whom", "which", "what", "whose",
		// Prepositions
		"in", "on", "at", "by", "for", "with", "about", "against", "between",
		"through", "during", "before", "after", "above", "below", "to", "from",
		"up", "down", "out", "off", "over", "under", "into", "onto",
		// Conjunctions
		"and", "but", "or", "nor", "so", "yet", "both", "either", "neither",
		// Common verbs (when used as auxiliaries)
		"is", "am", "are", "was", "were", "be", "been", "being",
		"have", "has", "had", "having", "do", "does", "did", "doing",
		"will", "would", "shall", "should", "may", "might", "must", "can", "could",
		// Adverbs
		"not", "no", "very", "just", "also", "too", "more", "most", "only",
		"then", "than", "when", "where", "how", "why", "all", "each", "every",
		"any", "some", "such", "here", "there", "now",
		// Other common words
		"if", "as", "of", "like", "please", "thanks", "thank", "okay", "yes",
		"yeah", "sure", "well", "really", "actually", "basically", "right",
		"know", "think", "want", "need", "tell", "get", "got", "make", "let",
		"say", "said", "see", "seem", "take", "give", "come", "go", "going",
		"thing", "things", "way", "even", "still", "already", "much", "many",
	}

	set := make(map[string]bool, len(words))
	for _, w := range words {
		set[w] = true
	}
	return set
}
