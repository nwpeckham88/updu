package models

import "time"

// defaultPushGraceRatio is the fraction of the expected interval used as the
// fallback tolerance when no explicit value is configured. Heartbeats are
// expected to be regular: a tight default surfaces cadence drift early.
const defaultPushGraceRatio = 0.10

// defaultPushGraceCap bounds the fallback tolerance regardless of cadence so
// long-running schedules (daily, weekly) don't accumulate hours of slack.
const defaultPushGraceCap = 10 * time.Minute

// MaxPushGraceSeconds bounds configured push tolerance to seven days.
const MaxPushGraceSeconds = 7 * 24 * 60 * 60

func defaultPushGraceDuration(intervalS int) time.Duration {
	if intervalS <= 0 {
		return 0
	}

	expectedInterval := time.Duration(intervalS) * time.Second
	scaled := time.Duration(float64(expectedInterval) * defaultPushGraceRatio)
	if scaled > defaultPushGraceCap {
		return defaultPushGraceCap
	}
	return scaled
}

// DefaultPushGraceSeconds returns the fallback grace period used when a push
// monitor does not define an explicit tolerance.
func DefaultPushGraceSeconds(intervalS int) int {
	return int(defaultPushGraceDuration(intervalS) / time.Second)
}

// EffectiveGraceDuration resolves the configured push tolerance. If no explicit
// value is present, the historical 30% interval fallback is used.
func (c PushMonitorConfig) EffectiveGraceDuration(intervalS int) time.Duration {
	if c.GracePeriodS != nil && *c.GracePeriodS >= 0 {
		graceSeconds := *c.GracePeriodS
		if graceSeconds > MaxPushGraceSeconds {
			graceSeconds = MaxPushGraceSeconds
		}
		return time.Duration(graceSeconds) * time.Second
	}

	return defaultPushGraceDuration(intervalS)
}

// EffectiveGraceSeconds resolves the configured push tolerance in whole seconds.
func (c PushMonitorConfig) EffectiveGraceSeconds(intervalS int) int {
	return int(c.EffectiveGraceDuration(intervalS) / time.Second)
}
