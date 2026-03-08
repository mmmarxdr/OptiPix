package middleware

import (
	"log"
	"net/http"
	"sync"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("%s %s %d %s %s", r.Method, r.URL.Path, rw.statusCode, time.Since(start), r.RemoteAddr)
	})
}

func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

type visitor struct {
	requests int
	lastSeen time.Time
}

var (
	visitors = make(map[string]*visitor)
	mu       sync.Mutex
)

func init() {
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()
}

func IPRateLimiter(limitPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if hasPort := len(ip) > 0 && ip[0] == '['; !hasPort {
				for i := len(ip) - 1; i >= 0; i-- {
					if ip[i] == ':' {
						ip = ip[:i]
						break
					}
				}
			}

			if xForwarded := r.Header.Get("X-Forwarded-For"); xForwarded != "" {
				ip = xForwarded
			}

			mu.Lock()
			v, exists := visitors[ip]
			if !exists || time.Since(v.lastSeen) > time.Minute {
				visitors[ip] = &visitor{requests: 1, lastSeen: time.Now()}
				mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			v.requests++
			v.lastSeen = time.Now()
			reqs := v.requests
			mu.Unlock()

			if reqs > limitPerMinute {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "Rate limit exceeded. Try again later."}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
