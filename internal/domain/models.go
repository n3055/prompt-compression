// Package domain defines the core business types.
// This package has ZERO imports from other internal packages (Clean Architecture innermost ring).
package domain

import "time"

// Role represents who sent the message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// Message represents a single turn in a conversation.
type Message struct {
	ID               string    `json:"id"`
	Role             Role      `json:"role"`
	RawContent       string    `json:"raw_content"`
	CompressedTokens []string  `json:"compressed_tokens"`
	RawTokenCount    int       `json:"raw_token_count"`
	CompTokenCount   int       `json:"compressed_token_count"`
	CreatedAt        time.Time `json:"created_at"`
}

// Session represents a conversation chain.
type Session struct {
	ID           string    `json:"id"`
	Instructions string    `json:"instructions"`
	Messages     []Message `json:"messages"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TurnCount returns the number of messages in this session.
func (s *Session) TurnCount() int {
	return len(s.Messages)
}

// TotalRawTokens returns the sum of raw tokens across all messages.
func (s *Session) TotalRawTokens() int {
	total := 0
	for _, m := range s.Messages {
		total += m.RawTokenCount
	}
	return total
}

// TotalCompressedTokens returns the sum of compressed tokens across all messages.
func (s *Session) TotalCompressedTokens() int {
	total := 0
	for _, m := range s.Messages {
		total += m.CompTokenCount
	}
	return total
}

// CompressedContext returns all compressed tokens from history, joined.
func (s *Session) CompressedContext() []string {
	var tokens []string
	for _, m := range s.Messages {
		tokens = append(tokens, m.CompressedTokens...)
	}
	return tokens
}

// ProcessResult is the output of processing a chat message.
type ProcessResult struct {
	SessionID            string  `json:"session_id"`
	CompressedPrompt     string  `json:"compressed_prompt"`
	OriginalContextTokens int    `json:"original_context_tokens"`
	CompressedContextTokens int  `json:"compressed_context_tokens"`
	SavingsPercent       float64 `json:"savings_percent"`
	TurnNumber           int     `json:"turn_number"`
	Message              string  `json:"message"`
}

// SessionInfo provides summary stats about a session.
type SessionInfo struct {
	ID                  string    `json:"id"`
	Instructions        string    `json:"instructions"`
	TurnCount           int       `json:"turn_count"`
	TotalRawTokens      int       `json:"total_raw_tokens"`
	TotalCompTokens     int       `json:"total_compressed_tokens"`
	OverallSavings      float64   `json:"overall_savings_percent"`
	CachedContextTokens []string  `json:"cached_context_tokens"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// HistoryEntry is a single message with compression stats for API output.
type HistoryEntry struct {
	Turn             int      `json:"turn"`
	Role             Role     `json:"role"`
	RawContent       string   `json:"raw_content"`
	CompressedTokens []string `json:"compressed_tokens"`
	RawTokenCount    int      `json:"raw_token_count"`
	CompTokenCount   int      `json:"compressed_token_count"`
	SavingsPercent   float64  `json:"savings_percent"`
}
