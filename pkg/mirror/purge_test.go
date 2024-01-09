package mirror_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	apiv1 "github.com/metal-stack/oci-mirror/api/v1"
	"github.com/metal-stack/oci-mirror/pkg/mirror"
	"github.com/stretchr/testify/require"
)

func TestPurge(t *testing.T) {

	env := map[string]string{
		"REGISTRY_STORAGE_DELETE_ENABLED": "true",
	}

	dstip, dstport, err := startRegistry(env, nil, nil)
	require.NoError(t, err)
	dstRegistry := fmt.Sprintf("%s:%d", dstip, dstport)

	dstAlpine := fmt.Sprintf("%s/library/alpine", dstRegistry)

	for _, tag := range []string{"foo", "bar", "3.10", "3.11", "3.12", "3.13", "3.14", "3.15", "3.16", "3.17", "3.18", "3.19"} {
		err = createImage(dstAlpine, tag)
		require.NoError(t, err)
	}

	config := apiv1.Config{
		Images: []apiv1.ImageMirror{
			{
				Destination: "http://" + dstAlpine,
				Purge: &apiv1.Purge{
					Tags:   []string{"foo"},
					Semver: pointer.Pointer("<= 3.15"),
				},
			},
		},
	}

	m := mirror.New(slog.Default(), config)
	err = m.Purge(context.Background())
	require.NoError(t, err)

	tags, err := crane.ListTags(dstAlpine)
	require.NoError(t, err)
	require.Len(t, tags, 6)
	require.ElementsMatch(t, []string{"bar", "3.16", "3.17", "3.18", "3.19", "latest"}, tags)
}
