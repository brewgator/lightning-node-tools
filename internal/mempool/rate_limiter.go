package mempool

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	tokens    chan struct{}
	ticker    *time.Ticker
	maxTokens int
	interval  time.Duration
	mu        sync.RWMutex
	stopped   bool
}

// NewRateLimiter creates a new rate limiter
// maxRequests: maximum number of requests allowed
// interval: time window for the requests (e.g., per minute)
func NewRateLimiter(maxRequests int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		tokens:    make(chan struct{}, maxRequests),
		maxTokens: maxRequests,
		interval:  interval,
	}
	
	// Fill initial tokens
	for i := 0; i < maxRequests; i++ {
		rl.tokens <- struct{}{}
	}
	
	// Start token replenishment
	rl.ticker = time.NewTicker(interval / time.Duration(maxRequests))
	go rl.replenishTokens()
	
	return rl
}

// Wait blocks until a token is available or context is cancelled
func (rl *RateLimiter) Wait() error {
	return rl.WaitWithContext(context.Background())
}

// WaitWithContext blocks until a token is available or context is cancelled
func (rl *RateLimiter) WaitWithContext(ctx context.Context) error {
	rl.mu.RLock()
	if rl.stopped {
		rl.mu.RUnlock()
		return fmt.Errorf("rate limiter is stopped")
	}
	rl.mu.RUnlock()
	
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryAcquire attempts to acquire a token without blocking
func (rl *RateLimiter) TryAcquire() bool {
	rl.mu.RLock()
	if rl.stopped {
		rl.mu.RUnlock()
		return false
	}
	rl.mu.RUnlock()
	
	select {
	case <-rl.tokens:
		return true
	default:
		return false
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	if !rl.stopped {
		rl.stopped = true
		rl.ticker.Stop()
		close(rl.tokens)
	}
}

// replenishTokens adds tokens back to the bucket periodically
func (rl *RateLimiter) replenishTokens() {
	for range rl.ticker.C {
		rl.mu.RLock()
		stopped := rl.stopped
		rl.mu.RUnlock()
		
		if stopped {
			return
		}
		
		select {
		case rl.tokens <- struct{}{}:
		default:
			// Bucket is full, ignore
		}
	}
}

// Available returns the number of available tokens
func (rl *RateLimiter) Available() int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	if rl.stopped {
		return 0
	}
	
	return len(rl.tokens)
}