package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Middleware applies browser protections and bounded, process-local rate limits.
func Middleware(next http.Handler) http.Handler {
	general := newRateLimiter(240, time.Minute)
	auth := newRateLimiter(20, 15*time.Minute)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet && r.Method != http.MethodHead && !sameOrigin(r) {
			http.Error(w, "Cross-origin requests are not allowed", http.StatusForbidden)
			return
		}
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		limiter := general
		if r.Method == http.MethodPost && (r.URL.Path == "/login" || r.URL.Path == "/register") {
			limiter = auth
		}
		staticAsset := (r.Method == http.MethodGet || r.Method == http.MethodHead) && (strings.HasPrefix(r.URL.Path, "/js/") || strings.HasPrefix(r.URL.Path, "/css/") || r.URL.Path == "/favicon.svg")
		if !staticAsset && !limiter.allow(ip) {
			w.Header().Set("Retry-After", strconv.Itoa(int(limiter.window.Seconds())))
			http.Error(w, "Too many requests. Try again later.", http.StatusTooManyRequests)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
		next.ServeHTTP(w, r)
	})
}

func sameOrigin(r *http.Request) bool {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	} // CLI clients do not send Origin.
	expected := os.Getenv("PUBLIC_ORIGIN")
	if expected == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		expected = scheme + "://" + r.Host
	}
	actual, err := url.Parse(origin)
	return err == nil && actual.Scheme != "" && actual.Host != "" && origin == expected
}

func requireMethod(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, method := range methods {
		if r.Method == method {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	return false
}

func decodeJSON(w http.ResponseWriter, r *http.Request, value interface{}) bool {
	if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "Expected one JSON object", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func parseForm(w http.ResponseWriter, r *http.Request) bool {
	var err error
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		err = r.ParseMultipartForm(16 * 1024)
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
			if len(r.MultipartForm.File) > 0 {
				http.Error(w, "File uploads are not supported", http.StatusBadRequest)
				return false
			}
		}
	} else if strings.Split(r.Header.Get("Content-Type"), ";")[0] == "application/x-www-form-urlencoded" {
		err = r.ParseForm()
	} else {
		http.Error(w, "Use a form-encoded request", http.StatusUnsupportedMediaType)
		return false
	}
	if err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return false
	}
	return true
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("request failed: %v", err)
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

func positiveID(value string) (int, error) {
	id, err := strconv.Atoi(value)
	if err != nil || id < 1 {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}

func pagination(r *http.Request) (int, int, error) {
	limit, offset := 10, 0
	var err error
	if value := r.URL.Query().Get("limit"); value != "" {
		limit, err = strconv.Atoi(value)
		if err != nil {
			return 0, 0, err
		}
	}
	if value := r.URL.Query().Get("offset"); value != "" {
		offset, err = strconv.Atoi(value)
		if err != nil {
			return 0, 0, err
		}
	}
	if limit < 1 || limit > 50 || offset < 0 || offset > 1000000 {
		return 0, 0, strconv.ErrRange
	}
	return limit, offset, nil
}

type rateEntry struct {
	count int
	reset time.Time
}
type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateEntry
	limit   int
	window  time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{entries: make(map[string]rateEntry), limit: limit, window: window}
}
func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	entry := l.entries[key]
	if now.After(entry.reset) {
		if len(l.entries) >= 10000 {
			for k, v := range l.entries {
				if now.After(v.reset) {
					delete(l.entries, k)
				}
			}
			if _, exists := l.entries[key]; !exists && len(l.entries) >= 10000 {
				return false
			}
		}
		entry = rateEntry{reset: now.Add(l.window)}
	}
	if entry.count >= l.limit {
		return false
	}
	entry.count++
	l.entries[key] = entry
	return true
}
