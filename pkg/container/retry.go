package container

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"
)

type retryPolicy struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

var defaultRetryPolicy = retryPolicy{
	MaxAttempts:  6,
	InitialDelay: 10 * time.Second,
	MaxDelay:     5 * time.Minute,
}

func (m *mirror) withRetry(operation, image string, fn func() error) error {
	return m.withRetryPolicy(operation, image, defaultRetryPolicy, fn)
}

func (m *mirror) withRetryPolicy(operation, image string, policy retryPolicy, fn func() error) error {
	delay := policy.InitialDelay
	var lastErr error

	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		if !isRetryable(err) || attempt == policy.MaxAttempts {
			return err
		}

		m.log.Warn("transient operation failure, retrying", "operation", operation, "image", image, "attempt", attempt, "max_attempts", policy.MaxAttempts, "retry_in", delay.String(), "error", err)

		time.Sleep(delay)

		// Exponential backoff, limiting to MaxDelay
		delay = min(time.Duration(float64(delay)*2), policy.MaxDelay)
	}

	return lastErr
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	errString := strings.ToLower(err.Error())
	retryableSubstrings := []string{
		"connection reset",
		"connection refused",
		"no such host",
		"temporary failure",
		"unexpected eof",
		"i/o timeout",
		"tls handshake timeout",
		"timeout awaiting response headers",
		"too many requests",
		"status code 429",
		"status code 5",
		"stopped after 10 redirects",
	}

	for _, sub := range retryableSubstrings {
		if strings.Contains(errString, sub) {
			return true
		}
	}

	return false
}
