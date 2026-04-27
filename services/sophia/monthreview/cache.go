package monthreview

import "sync"

// Cache stores MonthReview objects keyed by (user_id, period).
//
// Phase 1 limitation: this is a process-local in-memory cache with no TTL or
// persistence. Restarting Sophia clears all cached reviews; if you run more
// than one Sophia instance the caches are independent. Re-generation
// overwrites the existing entry.
type Cache interface {
	Get(userID string, period Period) (MonthReview, bool)
	Set(userID string, period Period, review MonthReview)
	Invalidate(userID string, period Period)
}

type memoryCache struct {
	mu sync.RWMutex
	m  map[string]MonthReview
}

// NewMemoryCache returns a goroutine-safe in-memory Cache.
func NewMemoryCache() Cache {
	return &memoryCache{m: map[string]MonthReview{}}
}

func cacheKey(userID string, p Period) string {
	return userID + "|" + p.String()
}

func (c *memoryCache) Get(userID string, period Period) (MonthReview, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.m[cacheKey(userID, period)]
	return v, ok
}

func (c *memoryCache) Set(userID string, period Period, review MonthReview) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[cacheKey(userID, period)] = review
}

func (c *memoryCache) Invalidate(userID string, period Period) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, cacheKey(userID, period))
}
