package bytelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKeyBuilderDefaultIpMapper(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.RemoteAddr = "1.2.3.4:56789"

	kb := NewKeyBuilder()
	key := kb.Key(req)

	if key != "1.2.3.4" {
		t.Errorf("expected key '1.2.3.4', got %q", key)
	}
}

func TestKeyBuilderAddIpMapper(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.RemoteAddr = "5.6.7.8:12345"

	kb := NewKeyBuilder().AddIP()
	key := kb.Key(req)

	if key != "5.6.7.8" {
		t.Errorf("expected key '5.6.7.8', got %q", key)
	}
}

func TestKeyBuilderAddHeaderMapper(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set("X-User", "user1")

	kb := NewKeyBuilder().AddHeader("X-User")
	key := kb.Key(req)

	if key != "user1" {
		t.Errorf("expected key 'user1', got %q", key)
	}
}

func TestKeyBuilderAddQueryMapper(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com?token=abc123", nil)

	kb := NewKeyBuilder().AddQuery("token")
	key := kb.Key(req)

	if key != "abc123" {
		t.Errorf("expected key 'abc123', got %q", key)
	}
}

func TestKeyBuilderCombinedMappers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com?token=xyz", nil)
	req.Header.Set("X-User", "user2")
	req.RemoteAddr = "10.0.0.1:4444"

	kb := NewKeyBuilder().
		AddIP().
		AddHeader("X-User").
		AddQuery("token")

	key := kb.Key(req)
	expected := "10.0.0.1|user2|xyz"

	if key != expected {
		t.Errorf("expected key %q, got %q", expected, key)
	}
}

func TestKeyBuilderAllowMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://example.com", nil)

	kb := NewKeyBuilder()
	kb.AllowMethod(http.MethodPost)

	if !kb.Allow(req) {
		t.Error("expected request with method POST to be allowed")
	}

	req.Method = http.MethodGet
	if kb.Allow(req) {
		t.Error("expected request with method GET to be denied")
	}
}

func TestKeyBuilderAllowHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set("X-Auth", "token")

	kb := NewKeyBuilder()
	kb.AllowHeader("X-Auth", "token")

	if !kb.Allow(req) {
		t.Error("expected request with X-Auth header to be allowed")
	}

	req.Header.Del("X-Auth")
	if kb.Allow(req) {
		t.Error("expected request without X-Auth header to be denied")
	}
}

func TestKeyBuilderSkipMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://example.com", nil)

	kb := NewKeyBuilder()
	kb.SkipMethod(http.MethodPost)

	if kb.Allow(req) {
		t.Error("expected request with method POST to be skipped (denied)")
	}

	req.Method = http.MethodGet
	if !kb.Allow(req) {
		t.Error("expected request with method GET to be allowed")
	}
}

func TestKeyBuilderSkipHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set("X-Skip", "true")

	kb := NewKeyBuilder()
	kb.SkipHeader("X-Skip", "true")

	if kb.Allow(req) {
		t.Error("expected request with X-Skip header to be skipped (denied)")
	}

	req.Header.Del("X-Skip")
	if !kb.Allow(req) {
		t.Error("expected request without X-Skip header to be allowed")
	}
}

func TestKeyBuilderAllowAndSkipCombined(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
	req.Header.Set("X-Auth", "token")
	req.Header.Set("X-Skip", "true")

	kb := NewKeyBuilder()
	kb.AllowMethod(http.MethodPost)
	kb.AllowHeader("X-Auth", "token")
	kb.SkipHeader("X-Skip", "true")

	if kb.Allow(req) {
		t.Error("expected request to be skipped due to X-Skip header")
	}

	req.Header.Del("X-Skip")
	if !kb.Allow(req) {
		t.Error("expected request to be allowed after removing X-Skip header")
	}

	req.Header.Del("X-Auth")
	if kb.Allow(req) {
		t.Error("expected request to be denied due to missing X-Auth header")
	}
}
