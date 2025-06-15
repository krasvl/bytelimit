package bytelimit

import (
	"net"
	"net/http"
	"strings"
)

// KeyBuilder is a builder for the key of the rate limiter.
// Map request to a key.
type KeyBuilder struct {
	keyMappers   []func(r *http.Request) string
	allowMappers []func(r *http.Request) bool
	skipMappers  []func(r *http.Request) bool
}

// NewKeyBuilder creates a new KeyBuilder.
func NewKeyBuilder() *KeyBuilder {
	return &KeyBuilder{
		keyMappers:   []func(r *http.Request) string{},
		allowMappers: []func(r *http.Request) bool{},
		skipMappers:  []func(r *http.Request) bool{},
	}
}

// Key builds a key from the request.
// The key is the concatenation of the results of the mappers.
// If no mappers are added, the IP mapper is added by default.
func (kb *KeyBuilder) Key(r *http.Request) string {
	if len(kb.keyMappers) == 0 {
		kb.AddIP()
		return kb.Key(r)
	}

	keys := make([]string, len(kb.keyMappers))
	for i, mapper := range kb.keyMappers {
		keys[i] = mapper(r)
	}
	return strings.Join(keys, "|")
}

// AddIP adds a mapper for the IP of the request.
// The mapper is a function that takes a request and returns a string.
// The string is used as a key for the rate limiter.
func (kb *KeyBuilder) AddIP() *KeyBuilder {
	kb.addKeyMapper(func(r *http.Request) string {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}
		return ip
	})
	return kb
}

// AddHeader adds a mapper for the header of the request.
// The mapper is a function that takes a request and returns a string.
// The string is used as a key for the rate limiter.
func (kb *KeyBuilder) AddHeader(header string) *KeyBuilder {
	kb.addKeyMapper(func(r *http.Request) string {
		return r.Header.Get(header)
	})
	return kb
}

// AddQuery adds a mapper for the query of the request.
// The mapper is a function that takes a request and returns a string.
// The string is used as a key for the rate limiter.
func (kb *KeyBuilder) AddQuery(query string) *KeyBuilder {
	kb.addKeyMapper(func(r *http.Request) string {
		return r.URL.Query().Get(query)
	})
	return kb
}

// addKeyMapper adds a mapper to the KeyBuilder.
// The mapper is a function that takes a request and returns a string.
// The string is used as a key for the rate limiter.
func (kb *KeyBuilder) addKeyMapper(mapper func(r *http.Request) string) {
	kb.keyMappers = append(kb.keyMappers, mapper)
}

// Allow checks if the request is allowed.
// The request is allowed if all the allow mappers return true.
func (kb *KeyBuilder) Allow(r *http.Request) bool {
	// If any allow mapper returns false, the request is not allowed.
	for _, mapper := range kb.allowMappers {
		if !mapper(r) {
			return false
		}
	}

	// If any skip mapper returns true, the request is skipped.
	for _, mapper := range kb.skipMappers {
		if mapper(r) {
			return false
		}
	}

	return true
}

// AllowMethod adds a mapper for the method of the request.
// The mapper is a function that takes a request and returns a boolean.
// The boolean is used to determine if the request is allowed.
func (kb *KeyBuilder) AllowMethod(method string) *KeyBuilder {
	kb.addAllowMapper(func(r *http.Request) bool {
		return r.Method == method
	})
	return kb
}

// AllowHeader adds a mapper for the header of the request.
// The mapper is a function that takes a request and returns a boolean.
// The boolean is used to determine if the request is allowed.
func (kb *KeyBuilder) AllowHeader(header string, value string) *KeyBuilder {
	kb.addAllowMapper(func(r *http.Request) bool {
		return r.Header.Get(header) == value
	})
	return kb
}

// addAllowMapper adds a mapper to the KeyBuilder.
// The mapper is a function that takes a request and returns a boolean.
// The boolean is used to determine if the request is allowed.
func (kb *KeyBuilder) addAllowMapper(mapper func(r *http.Request) bool) {
	kb.allowMappers = append(kb.allowMappers, mapper)
}

// SkipMethod adds a mapper for the method of the request.
// The mapper is a function that takes a request and returns a boolean.
// The boolean is used to determine if the request is skipped.
func (kb *KeyBuilder) SkipMethod(method string) *KeyBuilder {
	kb.addSkipMapper(func(r *http.Request) bool {
		return r.Method == method
	})
	return kb
}

// SkipHeader adds a mapper for the header of the request.
// The mapper is a function that takes a request and returns a boolean.
// The boolean is used to determine if the request is skipped.
func (kb *KeyBuilder) SkipHeader(header string, value string) *KeyBuilder {
	kb.addSkipMapper(func(r *http.Request) bool {
		return r.Header.Get(header) == value
	})
	return kb
}

// addSkip adds a skip function to the KeyBuilder.
// The skip function is a function that takes a request and returns a boolean.
// If the skip function returns true, the request is skipped and the key is not built.
func (kb *KeyBuilder) addSkipMapper(skip func(r *http.Request) bool) {
	kb.skipMappers = append(kb.skipMappers, skip)
}
