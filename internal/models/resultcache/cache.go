package resultcache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type entry[T any] struct {
	value     T
	expiresAt time.Time
}

// Cache is a small process-local, tenant-keyed result cache. It is bounded so
// document ingestion cannot grow model response memory without limit.
type Cache[T any] struct {
	mu      sync.Mutex
	entries map[string]entry[T]
	ttl     time.Duration
	max     int
}

func New[T any]() *Cache[T] {
	ttl := 10 * time.Minute
	if raw := strings.TrimSpace(os.Getenv("LORELATTICE_MODEL_RESULT_CACHE_TTL")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed >= 0 {
			ttl = parsed
		}
	}
	max := 2000
	if raw := strings.TrimSpace(os.Getenv("LORELATTICE_MODEL_RESULT_CACHE_MAX_ENTRIES")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			max = parsed
		}
	}
	return &Cache[T]{entries: make(map[string]entry[T]), ttl: ttl, max: max}
}

func (c *Cache[T]) Get(key string) (T, bool) {
	var zero T
	if c == nil || c.ttl <= 0 {
		return zero, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.entries[key]
	if !ok {
		return zero, false
	}
	if time.Now().After(item.expiresAt) {
		delete(c.entries, key)
		return zero, false
	}
	return item.value, true
}

func (c *Cache[T]) Set(key string, value T) {
	if c == nil || c.ttl <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.max {
		now := time.Now()
		for existing, item := range c.entries {
			if now.After(item.expiresAt) {
				delete(c.entries, existing)
			}
		}
		if len(c.entries) >= c.max {
			// Random-map eviction is O(1), deterministic behavior is unnecessary
			// for a best-effort performance cache.
			for existing := range c.entries {
				delete(c.entries, existing)
				break
			}
		}
	}
	c.entries[key] = entry[T]{value: value, expiresAt: time.Now().Add(c.ttl)}
}

func Key(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		h.Write([]byte{0})
		h.Write([]byte(part))
	}
	return hex.EncodeToString(h.Sum(nil))
}
