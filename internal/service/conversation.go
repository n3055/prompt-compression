package service

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/n3055/backend-project/internal/domain"
	"github.com/n3055/backend-project/internal/store"
)

// ConversationService orchestrates the full chat processing flow.
// It is the primary entry point for business logic.
type ConversationService struct {
	store      store.Store
	compressor *Compressor
	cache      *Cache
	log        *slog.Logger
}

// NewConversationService creates the conversation service with all dependencies injected.
func NewConversationService(
	st store.Store,
	comp *Compressor,
	cache *Cache,
	log *slog.Logger,
) *ConversationService {
	return &ConversationService{
		store:      st,
		compressor: comp,
		cache:      cache,
		log:        log,
	}
}

// ProcessMessage handles an incoming chat message:
//  1. Gets or creates the session
//  2. Compresses the query into keywords
//  3. Updates the prompt cache
//  4. Builds the final compressed prompt
//  5. Saves the message to the session
//  6. Returns the result with token savings stats
func (s *ConversationService) ProcessMessage(sessionID, instructions, query string) (*domain.ProcessResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("query cannot be empty")
	}
	if strings.TrimSpace(instructions) == "" {
		return nil, errors.New("instructions cannot be empty")
	}

	// 1. Get or create session.
	session, err := s.getOrCreateSession(sessionID, instructions)
	if err != nil {
		return nil, fmt.Errorf("session error: %w", err)
	}

	s.log.Info("processing message",
		"session_id", session.ID,
		"turn", session.TurnCount()+1,
		"query_tokens", CountTokens(query),
	)

	// 2. Compress the query.
	compressedTokens := s.compressor.Compress(query)
	queryTokenCount := CountTokens(query)

	// 3. Update prompt cache with new compressed tokens.
	s.cache.Append(session.ID, compressedTokens, instructions)

	// 4. Build the compressed prompt.
	cachedContext := s.cache.BuildContextString(session.ID)
	compressedPrompt := buildPrompt(instructions, cachedContext, query)

	// 5. Create and save the message.
	msg := domain.Message{
		ID:               uuid.New().String(),
		Role:             domain.RoleUser,
		RawContent:       query,
		CompressedTokens: compressedTokens,
		RawTokenCount:    queryTokenCount,
		CompTokenCount:   len(compressedTokens),
		CreatedAt:        time.Now().UTC(),
	}
	session.Messages = append(session.Messages, msg)

	if err := s.store.UpdateSession(session); err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	// 6. Calculate stats.
	originalContextTokens := session.TotalRawTokens()
	compressedContextTokens := len(s.cache.Get(session.ID))
	savings := SavingsPercent(originalContextTokens, compressedContextTokens)

	s.log.Info("message processed",
		"session_id", session.ID,
		"original_tokens", originalContextTokens,
		"compressed_tokens", compressedContextTokens,
		"savings_percent", savings,
	)

	return &domain.ProcessResult{
		SessionID:               session.ID,
		CompressedPrompt:        compressedPrompt,
		OriginalContextTokens:   originalContextTokens,
		CompressedContextTokens: compressedContextTokens,
		SavingsPercent:          savings,
		TurnNumber:              session.TurnCount(),
		Message:                 "Prompt compressed and ready. Forward compressed_prompt to your LLM.",
	}, nil
}

// GetSessionInfo returns summary stats for a session.
func (s *ConversationService) GetSessionInfo(sessionID string) (*domain.SessionInfo, error) {
	session, err := s.store.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	cachedTokens := s.cache.Get(sessionID)
	totalRaw := session.TotalRawTokens()
	totalComp := session.TotalCompressedTokens()

	return &domain.SessionInfo{
		ID:                  session.ID,
		Instructions:        session.Instructions,
		TurnCount:           session.TurnCount(),
		TotalRawTokens:      totalRaw,
		TotalCompTokens:     totalComp,
		OverallSavings:      SavingsPercent(totalRaw, totalComp),
		CachedContextTokens: cachedTokens,
		CreatedAt:           session.CreatedAt,
		UpdatedAt:           session.UpdatedAt,
	}, nil
}

// GetHistory returns the full conversation history with per-turn compression stats.
func (s *ConversationService) GetHistory(sessionID string) ([]domain.HistoryEntry, error) {
	session, err := s.store.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	entries := make([]domain.HistoryEntry, len(session.Messages))
	for i, msg := range session.Messages {
		entries[i] = domain.HistoryEntry{
			Turn:             i + 1,
			Role:             msg.Role,
			RawContent:       msg.RawContent,
			CompressedTokens: msg.CompressedTokens,
			RawTokenCount:    msg.RawTokenCount,
			CompTokenCount:   msg.CompTokenCount,
			SavingsPercent:   SavingsPercent(msg.RawTokenCount, msg.CompTokenCount),
		}
	}

	return entries, nil
}

// DeleteSession removes a session and its cache.
func (s *ConversationService) DeleteSession(sessionID string) error {
	s.cache.Invalidate(sessionID)
	return s.store.DeleteSession(sessionID)
}

// --- Internal helpers ---

// getOrCreateSession retrieves an existing session or creates a new one.
func (s *ConversationService) getOrCreateSession(sessionID, instructions string) (*domain.Session, error) {
	if sessionID == "" {
		// Create a new session.
		newID := uuid.New().String()
		return s.store.CreateSession(newID, instructions)
	}

	// Try to get the existing session.
	session, err := s.store.GetSession(sessionID)
	if err != nil {
		var notFound *store.ErrNotFound
		if errors.As(err, &notFound) {
			// Session ID provided but doesn't exist — create it.
			return s.store.CreateSession(sessionID, instructions)
		}
		return nil, err
	}

	// Update instructions if they changed.
	if session.Instructions != instructions {
		session.Instructions = instructions
		s.cache.Invalidate(sessionID) // Instructions changed — invalidate cache.
		if err := s.store.UpdateSession(session); err != nil {
			return nil, err
		}
	}

	return session, nil
}

// buildPrompt constructs the final compressed prompt to send to an LLM.
func buildPrompt(instructions, compressedContext, currentQuery string) string {
	var b strings.Builder

	b.WriteString("[Instructions]\n")
	b.WriteString(instructions)
	b.WriteString("\n\n")

	if compressedContext != "" {
		b.WriteString("[Compressed Context from Conversation History]\n")
		b.WriteString(compressedContext)
		b.WriteString("\n\n")
	}

	b.WriteString("[Current Query]\n")
	b.WriteString(currentQuery)

	return b.String()
}
