package monthreview

import (
	"sync"
	"testing"
)

func TestMemoryCache_GetSetInvalidate(t *testing.T) {
	c := NewMemoryCache()
	p := Period{Year: 2026, Month: 4}

	if _, ok := c.Get("u1", p); ok {
		t.Fatalf("expected miss")
	}

	rev := MonthReview{UserID: "u1", Period: p.String()}
	c.Set("u1", p, rev)

	got, ok := c.Get("u1", p)
	if !ok {
		t.Fatalf("expected hit")
	}
	if got.UserID != "u1" {
		t.Fatalf("user mismatch: %v", got)
	}

	// Different user shouldn't collide.
	if _, ok := c.Get("u2", p); ok {
		t.Fatalf("expected miss for u2")
	}

	c.Invalidate("u1", p)
	if _, ok := c.Get("u1", p); ok {
		t.Fatalf("expected miss after invalidate")
	}
}

func TestMemoryCache_ConcurrentSafety(t *testing.T) {
	c := NewMemoryCache()
	p := Period{Year: 2026, Month: 4}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			c.Set("u", p, MonthReview{UserID: "u", Period: p.String()})
		}()
		go func() {
			defer wg.Done()
			_, _ = c.Get("u", p)
		}()
	}
	wg.Wait()
}
