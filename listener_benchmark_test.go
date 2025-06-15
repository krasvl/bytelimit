package bytelimit

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func BenchmarkListenerGetRequestsKeepAliveFalse(b *testing.B) {
	benchmarkCases := []struct {
		name        string
		withLimiter bool
	}{
		{"with_limiter", true},
		{"without_limiter", false},
	}

	b.ReportAllocs()
	for _, bc := range benchmarkCases {
		b.Run(bc.name, func(b *testing.B) {
			ln, err := net.Listen("tcp", ":0")
			if err != nil {
				b.Fatal("cant create listener", err)
			}
			defer ln.Close()

			var listener net.Listener
			if bc.withLimiter {
				limiter := NewLimiter(100*KiB, 100*KiB, 10*MiB, time.Minute)
				keyBuilder := NewKeyBuilder()
				keyBuilder.AddHeader("X-User")
				listener = NewListener(ln, limiter, keyBuilder)
			} else {
				listener = ln
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			srv := &http.Server{
				Handler: handler,
			}

			go func() {
				_ = srv.Serve(listener)
			}()
			defer srv.Close()

			time.Sleep(100 * time.Millisecond)

			addr := ln.Addr().String()
			client := &http.Client{
				Transport: &http.Transport{
					DisableKeepAlives: true,
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				req, err := http.NewRequest(http.MethodGet, "http://"+addr+"/", nil)
				if err != nil {
					b.Fatal(err)
				}
				req.Header.Set("X-User", fmt.Sprintf("user%d", i%10))

				resp, err := client.Do(req)
				if err != nil {
					b.Fatal(err)
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		})
	}
}

func BenchmarkListenerPostRequestsKeepAliveFalse(b *testing.B) {
	benchmarkCases := []struct {
		name        string
		withLimiter bool
	}{
		{"with_limiter", true},
		{"without_limiter", false},
	}

	b.ReportAllocs()
	for _, bc := range benchmarkCases {
		b.Run(bc.name, func(b *testing.B) {
			ln, err := net.Listen("tcp", ":0")
			if err != nil {
				b.Fatal("cant create listener", err)
			}
			defer ln.Close()

			var listener net.Listener
			if bc.withLimiter {
				limiter := NewLimiter(100*KiB, 100*KiB, 10*MiB, time.Minute)
				keyBuilder := NewKeyBuilder()
				keyBuilder.AddHeader("X-User")
				listener = NewListener(ln, limiter, keyBuilder)
			} else {
				listener = ln
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			srv := &http.Server{
				Handler: handler,
			}

			go func() {
				_ = srv.Serve(listener)
			}()
			defer srv.Close()

			time.Sleep(100 * time.Millisecond)

			addr := ln.Addr().String()
			client := &http.Client{
				Transport: &http.Transport{
					DisableKeepAlives: true,
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/", strings.NewReader(strings.Repeat("a", i/10)))
				if err != nil {
					b.Fatal(err)
				}
				req.Header.Set("X-User", fmt.Sprintf("user%d", i%10))

				resp, err := client.Do(req)
				if err != nil {
					b.Fatal(err)
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		})
	}
}

func BenchmarkListenerShuffleRequestsKeepAliveFalse(b *testing.B) {
	benchmarkCases := []struct {
		name        string
		withLimiter bool
	}{
		{"with_limiter", true},
		{"without_limiter", false},
	}

	b.ReportAllocs()
	for _, bc := range benchmarkCases {
		b.Run(bc.name, func(b *testing.B) {
			ln, err := net.Listen("tcp", ":0")
			if err != nil {
				b.Fatal("cant create listener", err)
			}
			defer ln.Close()

			var listener net.Listener
			if bc.withLimiter {
				limiter := NewLimiter(100*KiB, 100*KiB, 10*MiB, time.Minute)
				keyBuilder := NewKeyBuilder()
				keyBuilder.AddHeader("X-User")
				listener = NewListener(ln, limiter, keyBuilder)
			} else {
				listener = ln
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			srv := &http.Server{
				Handler: handler,
			}

			go func() {
				_ = srv.Serve(listener)
			}()
			defer srv.Close()

			time.Sleep(100 * time.Millisecond)

			addr := ln.Addr().String()
			client := &http.Client{
				Transport: &http.Transport{
					DisableKeepAlives: true,
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var method string
				switch i % 4 {
				case 0:
					method = http.MethodGet
				case 1:
					method = http.MethodPost
				case 2:
					method = http.MethodPut
				case 3:
					method = http.MethodDelete
				}

				var body io.Reader
				if method == http.MethodPost || method == http.MethodPut {
					body = strings.NewReader(strings.Repeat("a", i/10))
				}

				req, err := http.NewRequest(method, "http://"+addr+"/", body)
				if err != nil {
					b.Fatal(err)
				}
				req.Header.Set("X-User", fmt.Sprintf("user%d", i%10))

				resp, err := client.Do(req)
				if err != nil {
					b.Fatal(err)
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		})
	}
}
