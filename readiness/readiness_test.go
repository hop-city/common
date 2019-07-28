package readiness

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

// https://golang.org/pkg/testing/#T
// https://github.com/stretchr/testify

// -- helpers
func clearStatuses() {
	data.ready = make(map[string]bool)
}

func TestSet(t *testing.T) {
	clearStatuses()
	Set("a", false)
	assert.False(t, IsReady(),
		"Single not ready entity. IsReady should return false",
	)
}

func TestIsReady_empty(t *testing.T) {
	clearStatuses()
	assert.True(t,
		IsReady(),
		"Empty list should return ready true",
	)
}

func TestSet_emptyKeyString(t *testing.T) {
	clearStatuses()
	Set("", true)
	assert.True(t,
		IsReady(),
		"Empty string as a key. IsReady should return true",
	)
}

func TestSet_emptyUpdate(t *testing.T) {
	clearStatuses()
	Set("", false)
	Set("12", false)

	assert.False(t, IsReady())
	Set("12", true)
	assert.False(t, IsReady())
	Set("", true)
	assert.True(t, IsReady())
	Set("12", false)
	assert.False(t, IsReady())
}

func TestSet_multipleInParallel(t *testing.T) {
	clearStatuses()
	count := 100
	done := make(chan bool, count)
	for i := 0; i < count; i++ {
		localI := i
		go func() {
			Set(strconv.Itoa(localI), true)
			done <- true
		}()
	}
	for i := 0; i < count; i++ {
		<-done
	}

	assert.True(t, IsReady(),
		`Multiple true entries set in parallel.`)
	assert.Equal(t,
		len(data.ready), count,
		"Number of elements mismatch %d/%d",
		len(data.ready),
		count,
	)
	Set("10", false)
	assert.False(t, IsReady(),
		"One of ready statuses was set to false - it was not recognised.")
}

func BenchmarkSet(b *testing.B) {
	b.ResetTimer()
	// if one puts i for k, it will create huge maps and be very slow
	// Logging takes a bit more time than mutex and update together
	for i := 0; i < b.N; i++ {
		Set("a", true)
	}
}
