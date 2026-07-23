package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

type CaptchaStore struct {
	mu      sync.RWMutex
	items   map[string]*CaptchaItem
	cleanup *time.Ticker
}

type CaptchaItem struct {
	ID        string
	Answer    int
	CreatedAt time.Time
}

var defaultStore = &CaptchaStore{
	items: make(map[string]*CaptchaItem),
}

var storeOnce sync.Once

func getCaptchaStore() *CaptchaStore {
	storeOnce.Do(func() {
		defaultStore.cleanup = time.NewTicker(5 * time.Minute)
		go func() {
			for range defaultStore.cleanup.C {
				defaultStore.mu.Lock()
				for id, item := range defaultStore.items {
					if time.Since(item.CreatedAt) > 10*time.Minute {
						delete(defaultStore.items, id)
					}
				}
				defaultStore.mu.Unlock()
			}
		}()
	})
	return defaultStore
}

func randInt(max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64())
}

func randID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *CaptchaStore) Generate() (string, string, string) {
	// Generate a simple math captcha: a + b = ?
	a := randInt(20) + 1
	b := randInt(20) + 1
	answer := a + b

	id := randID()

	svg := generateCaptchaSVG(a, b)

	s.mu.Lock()
	s.items[id] = &CaptchaItem{
		ID:        id,
		Answer:    answer,
		CreatedAt: time.Now(),
	}
	s.mu.Unlock()

	return id, svg, fmt.Sprintf("%d + %d = ?", a, b)
}

func (s *CaptchaStore) Verify(id string, answer int) bool {
	s.mu.Lock()
	item, ok := s.items[id]
	if ok {
		delete(s.items, id) // one-time use
	}
	s.mu.Unlock()
	if !ok {
		return false
	}
	return item.Answer == answer
}

// generateCaptchaSVG creates an SVG with the math question and visual noise
func generateCaptchaSVG(a, b int) string {
	colors := []string{"#00e5ff", "#40a9ff", "#6c5ce7", "#a855f7", "#00e5a0", "#1e90ff"}
	text := fmt.Sprintf("%d + %d", a, b)

	// Random color from palette
	c := colors[randInt(len(colors))]

	// Build SVG
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 40" width="120" height="40">`))
	sb.WriteString(`<rect width="120" height="40" rx="3" fill="rgba(255,255,255,0.08)" stroke="rgba(255,255,255,0.2)" stroke-width="1"/>`)

	// Noise lines
	for i := 0; i < 3; i++ {
		x1 := randInt(100)
		y1 := randInt(30)
		x2 := randInt(100) + 10
		y2 := randInt(30) + 5
		alpha := 0.1 + float64(randInt(20))/100.0
		nl := colors[randInt(len(colors))]
		sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1" opacity="%.2f"/>`, x1, y1, x2, y2, nl, alpha))
	}

	// Noise dots
	for i := 0; i < 10; i++ {
		cx := randInt(110) + 5
		cy := randInt(30) + 5
		r := randInt(3) + 1
		alpha := 0.05 + float64(randInt(15))/100.0
		nd := colors[randInt(len(colors))]
		sb.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="%d" fill="%s" opacity="%.2f"/>`, cx, cy, r, nd, alpha))
	}

	// Main text with slight rotation
	rot := float64(randInt(6) - 3) // -3 to +3 degrees
	fontSize := 16 + randInt(3)
	sb.WriteString(fmt.Sprintf(`<text x="60" y="28" text-anchor="middle" font-size="%d" font-weight="bold" fill="%s" transform="rotate(%.1f 60 28)">%s</text>`, fontSize, c, rot, text))

	// Question mark
	sb.WriteString(fmt.Sprintf(`<text x="110" y="22" font-size="13" font-weight="bold" fill="%s" opacity="0.7">?</text>`, c))

	sb.WriteString(`</svg>`)
	return sb.String()
}

// ── HTTP Handlers ──

func (h *HTTPHandler) GetCaptcha(w http.ResponseWriter, r *http.Request) {
	store := getCaptchaStore()
	id, svg, _ := store.Generate()
	jsonResp(w, map[string]string{
		"captcha_id": id,
		"svg":        svg,
	}, 200)
}
