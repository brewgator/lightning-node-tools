package mempool

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter with burst support.
//
// The rate limiter starts with a full bucket of tokens (maxRequests), allowing
// immediate burst usage. Tokens are replenished one at a time at regular intervals,
// calculated as interval/maxRequests. For example, with maxRequests=10 and interval=1min,
// tokens are replenished every 6 seconds.
//
// This implementation allows burst traffic (all tokens can be consumed immediately),
// followed by sustained rate-limited access as tokens replenish. If you need strict
// sliding window rate limiting without burst support, consider a different approach.
type RateLimiter struct {
	tokens    chan struct{}
	ticker    *time.Ticker
	maxTokens int
	interval  time.Duration
	mu        sync.RWMutex
	stopped   bool
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewRateLimiter creates a new rate limiter with burst support.
//
// maxRequests: maximum number of tokens in the bucket (burst capacity)
// interval: time window for token replenishment (e.g., 1 minute for "10 requests per minute")
//
// The bucket starts full, allowing immediate burst of maxRequests. After that,
// tokens replenish at a rate of interval/maxRequests, supporting sustained throughput.
func NewRateLimiter(maxRequests int, interval time.Duration) *RateLimiter {
	ctx, cancel := context.WithCancel(context.Background())
	rl := &RateLimiter{
		tokens:    make(chan struct{}, maxRequests),
		maxTokens: maxRequests,
		interval:  interval,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Fill initial tokens
	for i := 0; i < maxRequests; i++ {
		rl.tokens <- struct{}{}
	}

	// Start token replenishment
	rl.ticker = time.NewTicker(interval / time.Duration(maxRequests))
	rl.wg.Add(1)
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
		rl.cancel() // Cancel context first to stop goroutine
		rl.ticker.Stop()
		rl.wg.Wait() // Wait for goroutine to finish before closing channel
		close(rl.tokens)
	}
}

// replenishTokens adds tokens back to the bucket periodically
func (rl *RateLimiter) replenishTokens() {
	defer rl.wg.Done()
	for {
		select {
		case <-rl.ctx.Done():
			return
		case <-rl.ticker.C:
			select {
			case <-rl.ctx.Done():
				return
			case rl.tokens <- struct{}{}:
			default:
				// Bucket is full, ignore
			}
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
