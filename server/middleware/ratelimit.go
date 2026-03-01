package middleware

import (
	"sync"
	"time"

	"haven/server/config"
)

// bucket implements a token bucket for rate limiting.
type bucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

func newBucket(maxTokens, refillRate float64) *bucket {
	return &bucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (b *bucket) allow() bool {
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
	b.lastRefill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// RateLimiter provides per-identifier token bucket rate limiting.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	hot     *config.HotConfig
	cleanAt time.Time
}

// NewRateLimiter creates a new RateLimiter backed by hot-reloadable config.
func NewRateLimiter(hot *config.HotConfig) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		hot:     hot,
		cleanAt: time.Now().Add(5 * time.Minute),
	}
}

// AllowMessage checks the per-client message rate limit.
func (rl *RateLimiter) AllowMessage(identifier string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.maybeClean()

	key := "msg:" + identifier
	b, ok := rl.buckets[key]
	if !ok {
		limits := rl.hot.RateLimits()
		b = newBucket(float64(limits.MessageBurst), float64(limits.MessagesPerSecond))
		rl.buckets[key] = b
	}
	return b.allow()
}

// AllowAuth checks the per-IP auth attempt rate limit.
func (rl *RateLimiter) AllowAuth(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.maybeClean()

	key := "auth:" + ip
	b, ok := rl.buckets[key]
	if !ok {
		limits := rl.hot.RateLimits()
		b = newBucket(float64(limits.AuthAttemptsPerMinute), float64(limits.AuthAttemptsPerMinute)/60.0)
		rl.buckets[key] = b
	}
	return b.allow()
}

// AllowRegistration checks the per-IP registration rate limit.
func (rl *RateLimiter) AllowRegistration(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.maybeClean()

	key := "reg:" + ip
	b, ok := rl.buckets[key]
	if !ok {
		limits := rl.hot.RateLimits()
		b = newBucket(float64(limits.RegistrationsPerIPPerHour), float64(limits.RegistrationsPerIPPerHour)/3600.0)
		rl.buckets[key] = b
	}
	return b.allow()
}

// maybeClean periodically removes stale buckets.
func (rl *RateLimiter) maybeClean() {
	now := time.Now()
	if now.Before(rl.cleanAt) {
		return
	}
	rl.cleanAt = now.Add(5 * time.Minute)

	staleThreshold := now.Add(-10 * time.Minute)
	for k, b := range rl.buckets {
		if b.lastRefill.Before(staleThreshold) {
			delete(rl.buckets, k)
		}
	}
}
