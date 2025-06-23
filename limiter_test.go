package bytelimit

import (
	"testing"
	"time"
)

func TestLimitSingleKey(t *testing.T) {
	tests := []struct {
		name    string
		payload Unit
		wait    time.Duration
		ok      bool
	}{
		{
			name:    "within limit",
			payload: KiB,
			wait:    0,
			ok:      true,
		},
		{
			name:    "exceed limit",
			payload: 100 * KiB,
			wait:    0,
			ok:      false,
		},
		{
			name:    "reset limit",
			payload: KiB,
			wait:    20 * time.Millisecond,
			ok:      true,
		},
		{
			name:    "within limit",
			payload: 7 * KiB,
			wait:    0,
			ok:      true,
		},
		{
			name:    "exceed limit",
			payload: 7 * KiB,
			wait:    0,
			ok:      false,
		},
	}

	limit := 10 * KiB
	burst := 10 * KiB
	ttl := 10 * time.Millisecond
	cacheSize := 10 * MiB

	key := "test-key" // same key for all tests
	limiter := NewLimiter(limit, burst, cacheSize, ttl)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// wait before limit check
			time.Sleep(tt.wait)

			ok := limiter.AllowN(key, tt.payload)
			if ok != tt.ok {
				t.Errorf("expected ok %t, got %t", tt.ok, ok)
			}
		})
	}
}

func TestLimitMultipleKeys(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		payload Unit
		ok      bool
	}{
		{
			name:    "within limit",
			key:     "test-key-1",
			payload: KiB,
			ok:      true,
		},
		{
			name:    "within limit with same key",
			key:     "test-key-1",
			payload: KiB,
			ok:      true,
		},
		{
			name:    "exceed limit",
			key:     "test-key-2",
			payload: 100 * KiB,
			ok:      false,
		},
		{
			name:    "within limit",
			key:     "test-key-3",
			payload: KiB,
			ok:      true,
		},
	}

	limit := 10 * KiB
	burst := 10 * KiB
	ttl := time.Second
	cacheSize := 10 * MiB

	limiter := NewLimiter(limit, burst, cacheSize, ttl)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// wait before limit check
			time.Sleep(10 * time.Millisecond)

			ok := limiter.AllowN(tt.key, tt.payload)
			if ok != tt.ok {
				t.Errorf("expected ok %t, got %t", tt.ok, ok)
			}
		})
	}
}

func TestLimitEviction(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		payload Unit
		ok      bool
	}{
		{
			name:    "within limit",
			key:     "test-key-1",
			payload: 8 * KiB,
			ok:      true,
		},
		{
			name:    "within limit",
			key:     "test-key-2",
			payload: 7 * KiB, // minimum payload must be evicted
			ok:      true,
		},
		{
			name:    "within limit",
			key:     "test-key-3",
			payload: 9 * KiB,
			ok:      true,
		},
		{
			name:    "within limit",
			key:     "test-key-2",
			payload: 7 * KiB,
			ok:      true, // test-key-2 must be evicted before
		},
	}

	limit := 10 * KiB
	burst := 10 * KiB
	ttl := time.Second
	cacheSize := 15 * KiB // enough for 2 keys only

	limiter := NewLimiter(limit, burst, cacheSize, ttl)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// wait before limit check
			time.Sleep(10 * time.Millisecond)

			ok := limiter.AllowN(tt.key, tt.payload)
			if ok != tt.ok {
				t.Errorf("expected ok %t, got %t", tt.ok, ok)
			}
		})
	}
}

func TestLimitTTL(t *testing.T) {
	tests := []struct {
		name    string
		payload Unit
		wait    time.Duration
		ok      bool
	}{
		{
			name:    "within limit",
			payload: 10 * KiB,
			wait:    10 * time.Millisecond,
			ok:      true, // ttl is not expired yet payload1 < limit
		},
		{
			name:    "exceed limit",
			payload: 10 * KiB,
			wait:    10 * time.Millisecond,
			ok:      false, // ttl is not expired yet payload1 + payload2 > limit
		},
		{
			name:    "within limit",
			payload: 10 * KiB,
			wait:    100 * time.Millisecond,
			ok:      true, // ttl is expired, payload3 < limit
		},
	}

	limit := 10 * KiB
	burst := 10 * KiB
	ttl := 50 * time.Millisecond
	cacheSize := 10 * MiB

	key := "test-key"
	limiter := NewLimiter(limit, burst, cacheSize, ttl)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// wait before limit check
			time.Sleep(tt.wait)

			ok := limiter.AllowN(key, tt.payload)
			if ok != tt.ok {
				t.Errorf("expected ok %t, got %t", tt.ok, ok)
			}
		})
	}
}

func TestLimitConcurrent(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		payload Unit
		ok      bool
	}{
		{
			name:    "within limit",
			key:     "test-key-1",
			payload: KiB,
			ok:      true,
		},
		{
			name:    "within limit",
			key:     "test-key-1",
			payload: KiB,
			ok:      true,
		},
		{
			name:    "within limit",
			key:     "test-key-2",
			payload: KiB,
			ok:      true,
		},
		{
			name:    "exceed limit",
			key:     "test-key-3",
			payload: 100 * KiB,
			ok:      false,
		},
		{
			name:    "within limit",
			key:     "test-key-4",
			payload: KiB,
			ok:      true,
		},
	}

	limit := 10 * KiB
	burst := 10 * KiB
	ttl := time.Second
	cacheSize := 10 * MiB

	limiter := NewLimiter(limit, burst, cacheSize, ttl)

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ok := limiter.AllowN(tt.key, tt.payload)
			if ok != tt.ok {
				t.Errorf("expected ok %t, got %t", tt.ok, ok)
			}
		})
	}
}
