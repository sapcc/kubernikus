package guttle

import (
	"sync"
	"time"

	"github.com/cenkalti/backoff"
)

// Backoff defines behavior of staggering reconnection retries.
type Backoff interface {
	// Next returns the duration to sleep before retrying reconnections.
	// If the returned value is negative, the retry is aborted.
	NextBackOff() time.Duration

	// Reset is used to signal a reconnection was successful and next
	// call to Next should return desired time duration for 1st reconnection
	// attempt.
	Reset()
}

type expBackoff struct {
	mu sync.Mutex
	bk *backoff.ExponentialBackOff
}

func (eb *expBackoff) NextBackOff() time.Duration {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	return eb.bk.NextBackOff()
}

func (eb *expBackoff) Reset() {
	eb.mu.Lock()
	eb.bk.Reset()
	eb.mu.Unlock()
}

func newForeverBackoff() *expBackoff {
	eb := &expBackoff{
		bk: backoff.NewExponentialBackOff(),
	}
	eb.bk.MaxElapsedTime = 0             // never stops
	eb.bk.MaxInterval = 20 * time.Second // wait no longer than 20 seconds before retrying
	return eb
}
