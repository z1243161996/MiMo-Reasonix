package context

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"mimo-reasonix/internal/provider"
)

// PrefixCache tracks the SHA-256 fingerprint of the stable system prompt prefix
// to detect mutations that invalidate the provider's prompt cache. The cache
// prefix includes the system message and any fixed tool schemas; messages
// appended after that (user/assistant turns) are append-only and don't
// invalidate the cache.
type PrefixCache struct {
	mu             sync.RWMutex
	currentHash    string // hex-encoded SHA-256 of the current prefix
	hitCount       int64
	missCount      int64
	lastToolHash   string // hash of the tool schema at last check
	lastSystemHash string // hash of the system prompt at last check
}

// NewPrefixCache creates a new, empty prefix cache.
func NewPrefixCache() *PrefixCache {
	return &PrefixCache{}
}

// ComputePrefixHash computes the SHA-256 hash of the cacheable prefix. The
// prefix consists of the system message (if present) followed by tool schemas.
// Returns the hex-encoded hash and the number of messages consumed as prefix.
func ComputePrefixHash(messages []provider.Message, toolSchemas []provider.ToolSchema) (hash string, prefixLen int) {
	h := sha256.New()

	// Hash system message (always first if present).
	for i, m := range messages {
		if m.Role != provider.RoleSystem {
			break
		}
		h.Write([]byte(m.Content))
		h.Write([]byte{0}) // separator
		prefixLen = i + 1
	}

	// Hash tool schemas (stable across turns).
	for _, ts := range toolSchemas {
		h.Write([]byte(ts.Name))
		h.Write([]byte{0})
		h.Write(ts.Parameters)
		h.Write([]byte{0})
	}

	return hex.EncodeToString(h.Sum(nil)), prefixLen
}

// CheckPrefix validates whether the current prefix matches the cached hash.
// It returns true if the prefix is unchanged (cache hit), false if it mutated
// (cache miss). After a miss, the internal hash is updated to the new value.
func (pc *PrefixCache) CheckPrefix(messages []provider.Message, toolSchemas []provider.ToolSchema) bool {
	newHash, _ := ComputePrefixHash(messages, toolSchemas)

	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.currentHash == "" {
		// First call — record the hash and report a miss (cold start).
		pc.currentHash = newHash
		pc.missCount++
		return false
	}

	if pc.currentHash == newHash {
		pc.hitCount++
		return true
	}

	// Prefix mutated — cache invalidated.
	pc.currentHash = newHash
	pc.missCount++
	return false
}

// CheckSystemPromptOnly validates whether just the system prompt changed.
// This is useful for detecting mutations in the system prompt without
// considering tool schema changes.
func (pc *PrefixCache) CheckSystemPromptOnly(messages []provider.Message) bool {
	h := sha256.New()
	for _, m := range messages {
		if m.Role != provider.RoleSystem {
			break
		}
		h.Write([]byte(m.Content))
		h.Write([]byte{0})
	}
	newHash := hex.EncodeToString(h.Sum(nil))

	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.lastSystemHash == "" {
		pc.lastSystemHash = newHash
		pc.missCount++
		return false
	}

	if pc.lastSystemHash == newHash {
		pc.hitCount++
		return true
	}

	pc.lastSystemHash = newHash
	pc.missCount++
	return false
}

// Stats returns the current hit/miss counts and the cache hit ratio.
func (pc *PrefixCache) Stats() (hits, misses int64, ratio float64) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	total := pc.hitCount + pc.missCount
	if total > 0 {
		ratio = float64(pc.hitCount) / float64(total)
	}
	return pc.hitCount, pc.missCount, ratio
}

// Reset clears all cached state and counters.
func (pc *PrefixCache) Reset() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.currentHash = ""
	pc.hitCount = 0
	pc.missCount = 0
	pc.lastToolHash = ""
	pc.lastSystemHash = ""
}

// CurrentHash returns the current prefix hash (hex-encoded SHA-256).
// Returns empty string if no prefix has been computed yet.
func (pc *PrefixCache) CurrentHash() string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.currentHash
}

// Invalidate forces a cache miss on the next CheckPrefix call by clearing
// the stored hash.
func (pc *PrefixCache) Invalidate() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.currentHash = ""
}
