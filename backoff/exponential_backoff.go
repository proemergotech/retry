package backoff

import (
	"math/rand"
	"time"
)

const (
	DefaultMaxElapsedTime      = 1 * time.Minute
	DefaultMaxInterval         = 5 * time.Second
	DefaultRandomizationFactor = 0.5

	initialInterval = 50 * time.Millisecond
	multiplier      = 1.5
)

type ExponentialBackoff struct {
	currentInterval     time.Duration
	maxElapsedTime      time.Duration
	maxInterval         time.Duration
	randomizationFactor float64
	startTime           time.Time
}

// NewExponentialBackOff returns a new backoff which implements exponential backoff capabilities for retry logic.
//
// maxElapsedTime this parameter is used as a global timeout for the backoff.
// After this much time elapses the next backoff will always give back false.
//
// maxInterval this parameter is used for setting the maximum retry interval which can be higher by the factor of randomization.
// randomizationFactor sets the next backoff interval's maximum difference from the last interval as a multiplier factor.
func NewExponentialBackOff(maxElapsedTime, maxInterval time.Duration, randomizationFactor float64) *ExponentialBackoff {
	return &ExponentialBackoff{
		currentInterval:     initialInterval,
		maxElapsedTime:      maxElapsedTime,
		maxInterval:         maxInterval,
		randomizationFactor: randomizationFactor,
		startTime:           time.Now(),
	}
}

// NextBackOff gives back whether there should be another retry and a duration for the next backoff.
func (b *ExponentialBackoff) NextBackOff() (bool, time.Duration) {
	if time.Now().After(b.startTime.Add(b.maxElapsedTime)) {
		return false, 0
	}
	floatInterval := float64(b.currentInterval)
	if floatInterval >= float64(b.maxInterval)/multiplier {
		b.currentInterval = b.maxInterval
	} else {
		b.currentInterval = time.Duration(floatInterval * multiplier)
	}
	return true, time.Duration(floatInterval * (1 + rand.Float64()*b.randomizationFactor)) //nolint:gosec
}
