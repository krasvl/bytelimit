# TestLimit

A network rate limiter implementation in Go that provides TCP connection rate limiting with configurable rules. This package allows you to limit the rate of incoming http requests based on various criteria like IP addresses or custom Headers.

## Features

- TCP connection rate limiting with configurable rules
- Flexible key building system for rate limiting rules
- Configurable rate limits and burst sizes
- TTL (Time To Live) support for rate limiters

## Quick Start

```go
package bytelimit

import (
	"net"
	"net/http"
	"time"
)

func main() {
	// Create a base TCP listener
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	// Create a rate limiter with:
	// - 10 MiB byte limit per second per key
	// - 20 MiB burst size
	// - 100 MiB cache size
	// - 1 hour TTL
	limiter := NewLimiter(10*MiB, 20*MiB, 1000*MiB, time.Hour)

	// Create a key builder (rate limit by IP by default)
	keyBuilder := NewKeyBuilder()

	// Wrap the listener with rate limiting
	rateLimitedListener := NewListener(listener, limiter, keyBuilder)

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create a server and serve on the rate-limited listener
	srv := &http.Server{
		Handler: handler,
	}

	// Serve on the rate-limited listener
	if err := srv.Serve(rateLimitedListener); err != nil {
		panic(err)
	}
}
```

## Integration

This library is designed to be modular and composable — it can be integrated into other libraries and frameworks that accept or wrap a `net.Listener`. This makes it suitable for embedding into:

- **HTTP servers** based on the standard `net/http` or alternatives like `fasthttp`
- **Middleware stacks** or **service frameworks** that allow listener wrapping

You can use `NewListener(...)` to wrap any existing `net.Listener` with rate limiting logic, enabling seamless integration into your existing infrastructure without modifying your handlers or business logic.

## Usage Examples

### Basic Rate Limiting by IP

```go
// Create a key builder that limits by IP address (default behavior)
keyBuilder := NewKeyBuilder()
```

### Rate Limiting by HTTP Headers

```go
keyBuilder := NewKeyBuilder()

// Rate limit by API key in header
keyBuilder.AddHeader("X-API-Key")

// Rate limit by multiple headers combined
keyBuilder.AddHeader("X-API-Key")
keyBuilder.AddHeader("X-Client-ID")
```

### Rate Limiting by Query Parameters

```go
keyBuilder := NewKeyBuilder()

// Rate limit by user ID in query
keyBuilder.AddQuery("user_id")

// Rate limit by multiple query parameters
keyBuilder.AddQuery("user_id")
keyBuilder.AddQuery("tenant_id")
```

### Method-based Rate Limiting

```go
keyBuilder := NewKeyBuilder()

// Allow only specific HTTP methods
keyBuilder.AllowMethod("GET")
keyBuilder.AllowMethod("POST")

// Skip rate limiting for specific methods
keyBuilder.SkipMethod("OPTIONS")
keyBuilder.SkipMethod("HEAD")
```

### Header-based Allow/Skip Rules

```go
keyBuilder := NewKeyBuilder()

// Allow requests only with specific header values
keyBuilder.AllowHeader("X-API-Key", "premium-key")
keyBuilder.AllowHeader("Content-Type", "application/json")

// Skip rate limiting for specific header values
keyBuilder.SkipHeader("X-Bypass-Rate-Limit", "true")
keyBuilder.SkipHeader("User-Agent", "health-check")
```

### Complex Rate Limiting Rules

```go
keyBuilder := NewKeyBuilder()

// Rate limit by IP and API key combination
keyBuilder.AddIP()
keyBuilder.AddHeader("X-API-Key")

// Allow only POST requests with specific API key
keyBuilder.AllowMethod("POST")
keyBuilder.AllowHeader("X-API-Key", "premium-key")

// Skip rate limiting for health checks
keyBuilder.SkipHeader("X-Health-Check", "true")

// Confing can be chained as well
keyBuilder := NewKeyBuilder().
    AddIP().
    AddHeader("X-API-Key").
    AllowMethod("POST").
    AllowHeader("X-API-Key", "premium-key").
    SkipHeader("X-Health-Check", "true")
```

## Dependencies

- [github.com/dgraph-io/ristretto](https://github.com/dgraph-io/ristretto) - High-performance cache
- [golang.org/x/time](https://pkg.go.dev/golang.org/x/time) - Rate limiting implementation

## License

MIT License