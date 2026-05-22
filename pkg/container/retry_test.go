package container

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "redirect loop is retryable",
			err:       errors.New("Get \"https://example.com/login\": stopped after 10 redirects"),
			retryable: true,
		},
		{
			name:      "status 503 is retryable",
			err:       errors.New("unexpected status code 503 Service Unavailable"),
			retryable: true,
		},
		{
			name:      "context canceled is not retryable",
			err:       context.Canceled,
			retryable: false,
		},
		{
			name:      "authentication error is not retryable",
			err:       errors.New("UNAUTHORIZED: authentication required"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.retryable, isRetryable(tt.err))
		})
	}
}

func TestWithRetryPolicyRetriesUntilSuccess(t *testing.T) {
	m := &mirror{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	attempts := 0
	err := m.withRetryPolicy("list_tags", "example/image", retryPolicy{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond}, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("connection reset by peer")
		}
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, 3, attempts)
}

func TestWithRetryPolicyStopsOnPermanentError(t *testing.T) {
	m := &mirror{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	attempts := 0
	err := m.withRetryPolicy("list_tags", "example/image", retryPolicy{MaxAttempts: 4, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond}, func() error {
		attempts++
		return errors.New("UNAUTHORIZED: authentication required")
	})

	require.Error(t, err)
	require.Equal(t, 1, attempts)
}
