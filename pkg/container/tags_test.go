package container

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeSemverTag(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want string
	}{
		{
			name: "normal version is unchanged",
			tag:  "v1.2.3",
			want: "v1.2.3",
		},
		{
			name: "ci build and sha suffix is stripped",
			tag:  "v1.34.12-1765374555-6a93b0bbba8d6dc75b651cbafeedb062b2997716",
			want: "v1.34.12",
		},
		{
			name: "regular prerelease remains unchanged",
			tag:  "1.36.0-preview5",
			want: "1.36.0-preview5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, normalizeSemverTag(tt.tag))
		})
	}
}

func TestTagMatchesWithNormalizedBuildSuffix(t *testing.T) {
	m := &mirror{log: slog.New(slog.NewTextHandler(io.Discard, nil))}

	ok, err := m.tagMatches(
		"quay.io/cilium/cilium-envoy",
		"v1.34.12-1765374555-6a93b0bbba8d6dc75b651cbafeedb062b2997716",
		">= 0.34",
	)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestTagMatchesKeepsPrereleaseBehavior(t *testing.T) {
	m := &mirror{log: slog.New(slog.NewTextHandler(io.Discard, nil))}

	ok, err := m.tagMatches("example/image", "1.36.0-preview5", ">= 1.35")
	require.NoError(t, err)
	require.False(t, ok)
}
