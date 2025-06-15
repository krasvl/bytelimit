package bytelimit

import (
	"time"

	"github.com/dgraph-io/ristretto"
	"golang.org/x/time/rate"
)

// Limiter is a rate limiter that uses a ristretto cache to store rate limiters for each key.
// It is thread-safe and can be used concurrently.
type Limiter struct {
	limit Unit
	burst Unit
	ttl   time.Duration
	cache *ristretto.Cache
}

// NewLimiter creates a new Limiter with the given limit, burst, cache size, and ttl.
// The cache size is the *ristretto.Cache MaxCost (not max keys)
// The ttl is the time to live for the rate limiters in the cache.
// The limit is the maximum number of requests per second that can be made.
// The burst is the maximum number of requests that can be made in a burst.
func NewLimiter(limit Unit, burst Unit, cacheSize Unit, ttl time.Duration) *Limiter {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     int64(cacheSize),
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}
	return &Limiter{limit: limit, burst: burst, ttl: ttl, cache: cache}
}

// AllowN checks if the given key is allowed to make the given number of requests.
// If the key is not in the cache, a new rate limiter is created and added to the cache.
// If the key is in the cache, the rate limiter is retrieved from the cache.
func (l *Limiter) AllowN(key string, payload Unit) bool {
	if l.limit == 0 {
		return true
	}

	var limiter *rate.Limiter
	if val, ok := l.cache.Get(key); ok {
		limiter = val.(*rate.Limiter)
	} else {
		limiter = rate.NewLimiter(rate.Limit(l.limit), int(l.burst))
		l.cache.SetWithTTL(key, limiter, int64(payload), l.ttl)
	}

	return limiter.AllowN(time.Now(), int(payload))
}
