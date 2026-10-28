package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type urlEntry struct {
	URL       string    `json:"url"`
	Clicks    int       `json:"clicks"`
	CreatedAt time.Time `json:"created_at"`
}

type store struct {
	mu   sync.Mutex
	path string
	data map[string]*urlEntry
}

func newStore(path string) *store {
	s := &store{path: path, data: map[string]*urlEntry{}}
	s.load()
	return s
}

func (s *store) load() {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(b, &s.data)
}

func (s *store) save() {
	b, _ := json.MarshalIndent(s.data, "", "  ")
	tmp := s.path + ".tmp"
	_ = os.WriteFile(tmp, b, 0644)
	_ = os.Rename(tmp, s.path)
}

func randomCode(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	for i := range b {
		b[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(b)
}

func (s *store) shorten(url string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var code string
	for {
		code = randomCode(6)
		if _, exists := s.data[code]; !exists {
			break
		}
	}
	s.data[code] = &urlEntry{URL: url, CreatedAt: time.Now()}
	s.save()
	return code
}

func (s *store) resolve(code string) (*urlEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[code]
	if ok {
		e.Clicks++
		s.save()
	}
	return e, ok
}

func sendJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	dbPath := os.Getenv("SHORTENER_DB")
	if dbPath == "" {
		dbPath = "shortener.json"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	s := newStore(dbPath)

	mux := http.NewServeMux()

	mux.HandleFunc("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			sendJSON(w, 405, map[string]string{"error": "POST only"})
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			sendJSON(w, 400, map[string]string{"error": "cannot read body"})
			return
		}
		var payload struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(body, &payload); err != nil || payload.URL == "" {
			sendJSON(w, 400, map[string]string{"error": "url is required"})
			return
		}
		if !strings.HasPrefix(payload.URL, "http://") && !strings.HasPrefix(payload.URL, "https://") {
			sendJSON(w, 400, map[string]string{"error": "url must start with http:// or https://"})
			return
		}
		code := s.shorten(payload.URL)
		sendJSON(w, 201, map[string]string{"code": code, "short_url": "/" + code})
	})

	mux.HandleFunc("/api/stats/", func(w http.ResponseWriter, r *http.Request) {
		code := strings.TrimPrefix(r.URL.Path, "/api/stats/")
		e, ok := s.data[code]
		if !ok {
			sendJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		sendJSON(w, 200, map[string]interface{}{
			"url": e.URL, "clicks": e.Clicks, "created_at": e.CreatedAt,
		})
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		sendJSON(w, 200, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := strings.TrimPrefix(r.URL.Path, "/")
		if code == "" {
			sendJSON(w, 200, map[string]string{"service": "url-shortener"})
			return
		}
		e, ok := s.resolve(code)
		if !ok {
			sendJSON(w, 404, map[string]string{"error": "short URL not found"})
			return
		}
		http.Redirect(w, r, e.URL, http.StatusFound)
	})

	fmt.Printf("URL shortener listening on :%s (db: %s)\n", port, dbPath)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
