package outbox

import (
	"math/rand"
	"time"
)

type RetryPolicy struct {
	Base        time.Duration
	Max         time.Duration
	JitterRatio float64
}

func DefaultRetryPolicy(base time.Duration, max time.Duration) RetryPolicy {
	return RetryPolicy{
		Base:        base,
		Max:         max,
		JitterRatio: 0.15,
	}
}

func RetryDelay(policy RetryPolicy, attempts int) time.Duration {
	if attempts <= 0 {
		attempts = 1
	}

	delay := policy.Base
	for i := 1; i < attempts; i++ {
		if delay >= policy.Max {
			return policy.Max
		}
		delay *= 2
	}
	if delay > policy.Max {
		return policy.Max
	}
	return delay
}

func NextRetryAt(policy RetryPolicy, attempts int, now time.Time, rng *rand.Rand) time.Time {
	delay := RetryDelay(policy, attempts)
	if policy.JitterRatio > 0 && rng != nil {
		delay += time.Duration(float64(delay) * policy.JitterRatio * rng.Float64())
	}
	return now.Add(delay)
}
