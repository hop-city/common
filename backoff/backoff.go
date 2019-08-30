package backoff

import (
	"context"
	"math"
	"math/rand"
	"time"
)

type (
	Backoff struct {
		generator     *rand.Rand
		jitter        float64
		multiplier    float64
		durationMills float64
		maxDelay      float64
	}
)

// New - creates new exponential backoff generator
// Default jitter is 0.1
func New() *Backoff {
	rand.Seed(time.Now().Unix())
	s := rand.NewSource(time.Now().Unix())
	generator := rand.New(s)
	b := &Backoff{
		generator:     generator,
		jitter:        0.1,
		multiplier:    1,
		durationMills: 1000,
		maxDelay:      math.MaxInt64,
	}
	return b
}

// SetJitter - sets jitter value between 0 and 1
// Values from outside that range will be trimmed.
// Resulting duration values will be multiplied by random
// value from range 1 +- jitter
func (b *Backoff) SetJitter(jitter float64) *Backoff {
	if jitter > 1 {
		jitter = 1
	} else if jitter < 0 {
		jitter = 0
	}
	b.jitter = jitter
	return b
}

// SetBaseDuration - sets base value for minimal duration.
// 1 second by default. Will be modified by jitter.
func (b *Backoff) SetBaseDuration(durationMills float64) *Backoff {
	b.durationMills = durationMills
	return b
}

// SetMaxDelay - sets maximum delay.
// if limit is reached, each next iteration will
// return max value with jitter.
func (b *Backoff) SetMaxDelay(mills uint32) *Backoff {
	b.maxDelay = float64(mills)
	return b
}

// Backoff - returns channel closed after duration incremented exponentially
// for specified retry number or when context is closed. It doesn't keep
// it's own counter! If retry count is zero, closed channel is returned.
// If max value is reached - it will also be affected by jitter each time
// it is generated.
func (b *Backoff) Wait(ctx context.Context, retry uint) chan struct{} {
	ch := make(chan struct{})
	if retry == 0 {
		close(ch)
		return ch
	}
	jitter := b.genJitterMultiplier()
	delay := math.Pow(2, float64(retry-1))
	delay = delay * jitter
	delay = math.Min(delay, b.maxDelay*jitter)
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(delay) * time.Millisecond):
			close(ch)
		}
	}()

	return ch
}

// Once - generates duration with default value and jitter
func (b *Backoff) Once(ctx context.Context) chan struct{} {
	return b.Wait(ctx, 1)
}

func (b *Backoff) genJitterMultiplier() float64 {
	return 1 + b.jitter - (b.generator.Float64() * 2 * b.jitter)
}
