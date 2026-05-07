package service

import (
	"strings"
	"sync"
)

// Cache stores compressed context per session, avoiding recomputation.
// When a new message arrives, only the new message is compressed and appended
// to the cached context — we never reprocess old messages.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
}

// CacheEntry holds the cached compressed context for a single session.
type CacheEntry struct {
	// Tokens is the accumulated list of compressed keywords from all messages.
	Tokens []string
	// InstructionsHash is used to detect instruction changes (cache invalidation).
	InstructionsHash string
}

// NewCache creates a new prompt cache.
func NewCache() *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
	}
}

// Get retrieves the cached compressed tokens for a session.
// Returns nil if not cached.
func (c *Cache) Get(sessionID string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[sessionID]
	if !ok {
		return nil
	}

	// Return a copy.
	tokens := make([]string, len(entry.Tokens))
	copy(tokens, entry.Tokens)
	return tokens
}

// Append adds new compressed tokens to the cache for a session.
// If the instructions have changed, the cache is invalidated first.
func (c *Cache) Append(sessionID string, newTokens []string, instructions string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	hash := simpleHash(instructions)

	entry, ok := c.entries[sessionID]
	if !ok || entry.InstructionsHash != hash {
		// New session or instructions changed — start fresh.
		c.entries[sessionID] = &CacheEntry{
			Tokens:           deduplicate(newTokens),
			InstructionsHash: hash,
		}
		return
	}

	// Append new tokens and deduplicate.
	entry.Tokens = deduplicate(append(entry.Tokens, newTokens...))
}

// Invalidate removes a session's cache entry.
func (c *Cache) Invalidate(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, sessionID)
}

// BuildContextString reconstructs a compressed context string from cached tokens.
func (c *Cache) BuildContextString(sessionID string) string {
	tokens := c.Get(sessionID)
	if len(tokens) == 0 {
		return ""
	}
	return strings.Join(tokens, " | ")
}

// --- helpers ---

// simpleHash produces a fast string hash for change detection.
// Not cryptographic — only used for equality comparison.
func simpleHash(s string) string {
	// For simplicity, we use the string itself trimmed.
	// In production, use xxhash or fnv for large instructions.
	return strings.TrimSpace(s)
}

// deduplicate removes duplicate strings while preserving order.
func deduplicate(tokens []string) []string {
	seen := make(map[string]bool, len(tokens))
	result := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	return result
}
