package container

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	retry "github.com/avast/retry-go/v5"
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

	return retry.New(
		retry.Attempts(uint(policy.MaxAttempts)),
		retry.Delay(policy.InitialDelay),
		retry.MaxDelay(policy.MaxDelay),
		retry.RetryIf(isRetryable),
		retry.OnRetry(func(attempt uint, err error) {
			m.log.Warn("transient operation failure, retrying", "operation", operation, "image", image, "attempt", attempt+1, "max_attempts", policy.MaxAttempts, "error", err)
		})).Do(fn)

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
