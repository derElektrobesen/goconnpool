package goconnpool

import "time"

// Clock interface is required to emulate system clock.
type Clock interface {
	Now() time.Time
	Since(time.Time) time.Duration
}

// SystemClock is the default clock implementation for the package.
// This type of clock just proxies calls to the `time` package.
type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

func (SystemClock) Since(tm time.Time) time.Duration {
	return time.Since(tm)
}
