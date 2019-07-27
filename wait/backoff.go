package wait

import (
	"context"
	"math"
	"math/rand"
	"time"
)

type (
	Wait struct {
		generator     *rand.Rand
		jitter        float64
		multiplier    float64
		durationMills float64
		maxDelay      float64
	}
)

func New() *Wait {
	rand.Seed(time.Now().Unix())
	s := rand.NewSource(time.Now().Unix())
	generator := rand.New(s)
	b := &Wait{
		generator:     generator,
		jitter:        0.1,
		multiplier:    1,
		durationMills: 1000,
		maxDelay:      math.MaxInt64,
	}
	return b
}

func (b *Wait) SetJitter(jitter float64) *Wait {
	if jitter > 1 {
		jitter = 1
	} else if jitter < 0 {
		jitter = 0
	}
	b.jitter = jitter
	return b
}
func (b *Wait) SetMultiplier(multiplier float64) *Wait {
	b.multiplier = multiplier
	return b
}
func (b *Wait) SetDuration(durationMills float64) *Wait {
	b.durationMills = durationMills
	return b
}
func (b *Wait) SetMaxDelay(mills uint32) *Wait {
	b.maxDelay = float64(mills)
	return b
}

func (b *Wait) Backoff(ctx context.Context, retry uint) chan struct{} {
	ch := make(chan struct{})
	if retry == 0 {
		close(ch)
		return ch
	}
	delay := math.Pow(2, float64(retry-1)) * b.multiplier
	delay = delay * b.genJitterMultiplier()
	delay = math.Min(delay, b.maxDelay)
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

func (b *Wait) Do(ctx context.Context) chan struct{} {
	ch := make(chan struct{})
	delay := b.durationMills * b.genJitterMultiplier()
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

func (b *Wait) genJitterMultiplier() float64 {
	return 1 + b.jitter - (b.generator.Float64() * 2 * b.jitter)
}
