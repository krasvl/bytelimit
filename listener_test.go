package bytelimit

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestListenerDialBase(t *testing.T) {
	tests := []struct {
		name     string
		request  string
		expected int
	}{
		{
			name:     "no limit exceeded GET user1",
			request:  "GET / HTTP/1.1\r\nHost: localhost\r\nX-User: user1\r\n\r\n",
			expected: http.StatusOK, // 50 bytes < 100 bytes limit
		},
		{
			name:     "no limit exceeded POST user2",
			request:  "POST / HTTP/1.1\r\nHost: localhost\r\nX-User: user2\r\nContent-Length: 10\r\n\r\n" + strings.Repeat("a", 10),
			expected: http.StatusOK, // 81 bytes < 100 bytes limit
		},
		{
			name:     "limit exceeded GET user2",
			request:  "GET / HTTP/1.1\r\nHost: localhost\r\nX-User: user2\r\n\r\n",
			expected: http.StatusTooManyRequests, // 81 + 50 bytes > 100 bytes limit
		},
		{
			name:     "limit exceeded POST user3",
			request:  "POST / HTTP/1.1\r\nHost: localhost\r\nX-User: user3\r\nContent-Length: 1000\r\n\r\n" + strings.Repeat("a", 1000),
			expected: http.StatusTooManyRequests, // 1071 bytes > 100 bytes limit
		},
	}

	// listen on unused port
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal("cant create listener", err)
	}
	defer ln.Close()

	limiter := NewLimiter(100, 100, 10*MiB, time.Minute)

	keyBuilder := NewKeyBuilder()
	keyBuilder.AddHeader("X-User")

	// create listener
	listener := NewListener(ln, limiter, keyBuilder)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Handler: handler,
	}

	// serve on listener
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Errorf("unexpected server error: %v", err)
		}
	}()

	// wait for server to start
	time.Sleep(100 * time.Millisecond)

	// get listener address
	addr := ln.Addr().String()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// new tcp connecton per request
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				t.Fatal("cant dial server:", err)
			}
			defer conn.Close()

			// send raw request
			_, err = conn.Write([]byte(tt.request))
			if err != nil {
				t.Fatal("cant write request:", err)
			}

			// read response
			br := bufio.NewReader(conn)
			resp, err := http.ReadResponse(br, nil)
			if err != nil {
				t.Fatal("cant read response:", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expected {
				t.Errorf("unexpected response status: got %d, want %d", resp.StatusCode, tt.expected)
			}
		})
	}

	srv.Close()
}

func TestListenerHttpClientKeepAliveFalseBase(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		url      string
		headers  map[string]string
		body     string
		expected int
	}{
		{
			name:     "no limit exceeded GET user1",
			method:   http.MethodGet,
			url:      "/",
			headers:  map[string]string{"X-User": "user1"},
			body:     "",
			expected: http.StatusOK, // 125 bytes < 200 bytes limit
		},
		{
			name:     "no limit exceeded POST user2",
			method:   http.MethodPost,
			url:      "/",
			headers:  map[string]string{"X-User": "user2", "Content-Length": "10"},
			body:     strings.Repeat("a", 10),
			expected: http.StatusOK, // 156 bytes < 200 bytes limit
		},
		{
			name:     "limit exceeded GET user2",
			method:   http.MethodGet,
			url:      "/",
			headers:  map[string]string{"X-User": "user2"},
			body:     "",
			expected: http.StatusTooManyRequests, // 125 + 156 bytes > 200 bytes limit
		},
		{
			name:     "limit exceeded POST user3",
			method:   http.MethodPost,
			url:      "/",
			headers:  map[string]string{"X-User": "user3", "Content-Length": "1000"},
			body:     strings.Repeat("a", 1000),
			expected: http.StatusTooManyRequests, // 1148 bytes > 200 bytes limit
		},
	}

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal("cant create listener", err)
	}
	defer ln.Close()

	limiter := NewLimiter(200, 200, 10*MiB, time.Minute)

	keyBuilder := NewKeyBuilder()
	keyBuilder.AddHeader("X-User")

	listener := NewListener(ln, limiter, keyBuilder)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Handler: handler,
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Errorf("unexpected server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	addr := ln.Addr().String()
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true, // no keep-alive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// no keep-alive
			req, err := http.NewRequest(tt.method, "http://"+addr+tt.url, strings.NewReader(tt.body))
			if err != nil {
				t.Fatal("cannot create request:", err)
			}

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatal("request error:", err)
			}
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != tt.expected {
				t.Errorf("unexpected response status: got %d, want %d, body: %q", resp.StatusCode, tt.expected, string(bodyBytes))
			}
		})
	}

	srv.Close()
}

func TestListenerHttpClientKeepAliveFalseRateReset(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		url      string
		headers  map[string]string
		body     string
		wait     time.Duration
		expected int
	}{
		{
			name:     "no limit exceeded POST user",
			method:   http.MethodPost,
			url:      "/",
			headers:  map[string]string{"X-User": "user", "Content-Length": "10"},
			body:     strings.Repeat("a", 10),
			wait:     0,
			expected: http.StatusOK, // 156 bytes < 200 bytes limit
		},
		{
			name:     "limit exceeded POST user",
			method:   http.MethodPost,
			url:      "/",
			headers:  map[string]string{"X-User": "user", "Content-Length": "10"},
			body:     strings.Repeat("a", 10),
			wait:     0,
			expected: http.StatusTooManyRequests, // 156 + 156 bytes > 200 bytes limit
		},
		{
			name:     "limit reset POST user",
			method:   http.MethodPost,
			url:      "/",
			headers:  map[string]string{"X-User": "user", "Content-Length": "10"},
			body:     strings.Repeat("a", 10),
			wait:     2 * time.Second,
			expected: http.StatusOK, // 156 bytes < 200 bytes limit
		},
	}

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal("cant create listener", err)
	}
	defer ln.Close()

	limiter := NewLimiter(200, 200, 10*MiB, time.Minute)

	keyBuilder := NewKeyBuilder()
	keyBuilder.AddHeader("X-User")

	listener := NewListener(ln, limiter, keyBuilder)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Handler: handler,
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Errorf("unexpected server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	addr := ln.Addr().String()
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true, // no keep-alive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// wait for rate reset
			time.Sleep(tt.wait)

			// no keep-alive
			req, err := http.NewRequest(tt.method, "http://"+addr+tt.url, strings.NewReader(tt.body))
			if err != nil {
				t.Fatal("cannot create request:", err)
			}

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatal("request error:", err)
			}
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != tt.expected {
				t.Errorf("unexpected response status: got %d, want %d, body: %q", resp.StatusCode, tt.expected, string(bodyBytes))
			}
		})
	}

	srv.Close()
}

func TestListenerHttpClientKeepAliveFalseShuffledRequests(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		url      string
		headers  map[string]string
		body     string
		expected int
	}{
		{
			name:     "no limit exceeded GET user1",
			method:   http.MethodGet,
			url:      "/",
			headers:  map[string]string{"X-User": "user1"},
			body:     "",
			expected: http.StatusOK, // 125 bytes < 200 bytes limit
		},
		{
			name:     "no limit exceeded POST user2",
			method:   http.MethodPost,
			url:      "/",
			headers:  map[string]string{"X-User": "user2", "Content-Length": "10"},
			body:     strings.Repeat("a", 10),
			expected: http.StatusOK, // 156 bytes < 200 bytes limit
		},
		{
			name:     "no limit exceeded PUT user3",
			method:   http.MethodPut,
			url:      "/",
			headers:  map[string]string{"X-User": "user3", "Content-Length": "10"},
			body:     strings.Repeat("a", 10),
			expected: http.StatusOK, // 156 bytes < 200 bytes limit
		},
		{
			name:     "no limit exceeded DELETE user4",
			method:   http.MethodDelete,
			url:      "/",
			headers:  map[string]string{"X-User": "user4"},
			body:     "",
			expected: http.StatusOK, // 125 bytes < 200 bytes limit
		},
		{
			name:     "limit exceeded GET user1",
			method:   http.MethodGet,
			url:      "/",
			headers:  map[string]string{"X-User": "user1"},
			body:     "",
			expected: http.StatusTooManyRequests, // 125 + 125 bytes > 200 bytes limit
		},
		{
			name:     "limit exceeded POST user2",
			method:   http.MethodPost,
			url:      "/",
			headers:  map[string]string{"X-User": "user2", "Content-Length": "10"},
			body:     strings.Repeat("a", 10),
			expected: http.StatusTooManyRequests, // 156 + 156 bytes > 200 bytes limit
		},
		{
			name:     "limit exceeded PUT user3",
			method:   http.MethodPut,
			url:      "/",
			headers:  map[string]string{"X-User": "user3", "Content-Length": "10"},
			body:     strings.Repeat("a", 10),
			expected: http.StatusTooManyRequests, // 156 + 156 bytes > 200 bytes limit
		},
		{
			name:     "limit exceeded DELETE user4",
			method:   http.MethodDelete,
			url:      "/",
			headers:  map[string]string{"X-User": "user4"},
			body:     "",
			expected: http.StatusTooManyRequests, // 125 + 125 bytes > 200 bytes limit
		},
	}

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal("cant create listener", err)
	}
	defer ln.Close()

	limiter := NewLimiter(200, 200, 10*MiB, time.Minute)

	keyBuilder := NewKeyBuilder()
	keyBuilder.AddHeader("X-User")

	listener := NewListener(ln, limiter, keyBuilder)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Handler: handler,
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Errorf("unexpected server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	addr := ln.Addr().String()
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true, // no keep-alive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// no keep-alive
			req, err := http.NewRequest(tt.method, "http://"+addr+tt.url, strings.NewReader(tt.body))
			if err != nil {
				t.Fatal("cannot create request:", err)
			}

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatal("request error:", err)
			}
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != tt.expected {
				t.Errorf("unexpected response status: got %d, want %d, body: %q", resp.StatusCode, tt.expected, string(bodyBytes))
			}
		})
	}

	srv.Close()
}

func TestListenerHttpClientKeepAliveFalseAllow(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		url      string
		headers  map[string]string
		body     string
		expected int
	}{
		{
			name:     "allow POST user1",
			method:   http.MethodPost,
			url:      "/",
			headers:  map[string]string{"X-User": "user1", "X-Allow": "true"},
			body:     strings.Repeat("a", 1000),
			expected: http.StatusTooManyRequests, // > 200 bytes limit
		},
		{
			name:     "allow POST user2",
			method:   http.MethodPost,
			url:      "/",
			headers:  map[string]string{"X-User": "user2"},
			body:     strings.Repeat("a", 1000),
			expected: http.StatusOK, // skip no X-Allow header
		},
		{
			name:     "skip GET user3",
			method:   http.MethodGet,
			url:      "/",
			headers:  map[string]string{"X-User": "user3", "X-Allow": "true"},
			body:     "",
			expected: http.StatusOK, // skip post
		},
		{
			name:     "skip POST user4",
			method:   http.MethodPost,
			url:      "/",
			headers:  map[string]string{"X-User": "user4", "X-Allow": "true", "X-Skip": "true"},
			body:     strings.Repeat("a", 1000),
			expected: http.StatusOK, // skip X-Skip header
		},
	}

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal("cant create listener", err)
	}
	defer ln.Close()

	limiter := NewLimiter(200, 200, 10*MiB, time.Minute)

	keyBuilder := NewKeyBuilder()
	keyBuilder.AddHeader("X-User")
	keyBuilder.AllowHeader("X-Allow", "true")
	keyBuilder.AllowMethod(http.MethodPost)
	keyBuilder.SkipHeader("X-Skip", "true")

	listener := NewListener(ln, limiter, keyBuilder)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Handler: handler,
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Errorf("unexpected server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	addr := ln.Addr().String()
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true, // no keep-alive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// no keep-alive
			req, err := http.NewRequest(tt.method, "http://"+addr+tt.url, strings.NewReader(tt.body))
			if err != nil {
				t.Fatal("cannot create request:", err)
			}

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatal("request error:", err)
			}
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != tt.expected {
				t.Errorf("unexpected response status: got %d, want %d, body: %q", resp.StatusCode, tt.expected, string(bodyBytes))
			}
		})
	}

	srv.Close()
}
