// Package retry contains code to retry operations. This package
// is still considered very experimental.
//
// We believe this code will probably be useful in oodnsclient.
package retry

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

// TODO(bassosimone): we need to calibrate these parameters.
const (
	initialMean = 0.5
	finalMean   = 8.0
	meanFactor  = 2.0
	stdevFactor = 0.05
)

// Retry retries op until it succeeds, context expires, or we've
// attempted to retry the operation for too much time.
func Retry(ctx context.Context, op func() error) error {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for mean := initialMean; mean <= finalMean; mean *= meanFactor {
		if err := op(); err == nil {
			return nil
		}
		stdev := stdevFactor * mean
		seconds := rng.NormFloat64()*stdev + mean
		sleepTime := time.Duration(seconds * float64(time.Second))
		timer := time.NewTimer(sleepTime)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// FALLTHROUGH
		}
	}
	return errors.New("retry.Retry: all attempts failed")
}
