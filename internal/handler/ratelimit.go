package handler

import (
	"sync"
	"time"
)

var (
	rateLimiters = make(map[string]*tokenBucket)
	rateMu       sync.Mutex
)

type tokenBucket struct {
	tokens   float64
	lastTime time.Time
	maxRate  float64
}

func getLimiter(key string) *tokenBucket {
	rateMu.Lock()
	defer rateMu.Unlock()
	if rl, ok := rateLimiters[key]; ok {
		return rl
	}
	rl := &tokenBucket{tokens: 10, lastTime: time.Now(), maxRate: 10}
	rateLimiters[key] = rl
	return rl
}

func (tb *tokenBucket) allow(maxRPM float64) bool {
	tb.maxRate = maxRPM
	now := time.Now()
	elapsed := now.Sub(tb.lastTime).Seconds()
	tb.tokens += elapsed * maxRPM / 60.0
	if tb.tokens > maxRPM {
		tb.tokens = maxRPM
	}
	tb.lastTime = now
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}
