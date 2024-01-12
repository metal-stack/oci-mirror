package container_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	apiv1 "github.com/metal-stack/oci-mirror/api/v1"
	"github.com/metal-stack/oci-mirror/pkg/container"
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
	dstBusybox := fmt.Sprintf("%s/library/busybox", dstRegistry)
	dstFoo := fmt.Sprintf("%s/library/foo", dstRegistry)

	for _, tag := range []string{"foo", "bar", "3.10", "3.11", "3.12", "3.13", "3.14", "3.15", "3.16", "3.17", "3.18", "3.19"} {
		err = createImage(dstAlpine, tag)
		require.NoError(t, err)
	}
	for _, tag := range []string{"foo", "bar", "1.1", "1.2", "1.3", "1.4", "1.5", "1.6"} {
		err = createImage(dstBusybox, tag)
		require.NoError(t, err)
	}
	for _, tag := range []string{"foo", "bar"} {
		err = createImage(dstFoo, tag)
		require.NoError(t, err)
	}

	config := apiv1.Config{
		Images: []apiv1.ImageMirror{
			{
				Source:      dstAlpine,
				Destination: "http://" + dstAlpine,
				Match: apiv1.Match{
					Semver: pointer.Pointer(">= 3.17"),
				},
				Purge: &apiv1.Purge{
					Tags:   []string{"foo"},
					Semver: pointer.Pointer("<= 3.15"),
				},
			},
			{
				Source:      dstBusybox,
				Destination: "http://" + dstBusybox,
				Match: apiv1.Match{
					Semver: pointer.Pointer(">= 1.3"),
				},
				Purge: &apiv1.Purge{
					NoMatch: true,
				},
			},
		},
	}

	m := container.New(slog.Default(), config)
	err = m.Purge(context.Background())
	require.NoError(t, err)

	tags, err := crane.ListTags(dstAlpine)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"bar", "3.16", "3.17", "3.18", "3.19", "latest"}, tags)

	t.Logf("alpine tags:%s", tags)

	tags, err = crane.ListTags(dstBusybox)
	t.Logf("busybox tags:%s", tags)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"1.3", "1.4", "1.5", "1.6", "latest"}, tags)
}

func TestPurgeUnknown(t *testing.T) {

	env := map[string]string{
		"REGISTRY_STORAGE_DELETE_ENABLED": "true",
	}

	dstip, dstport, err := startRegistry(env, nil, nil)
	require.NoError(t, err)
	dstRegistry := fmt.Sprintf("%s:%d", dstip, dstport)

	dstAlpine := fmt.Sprintf("%s/library/alpine", dstRegistry)
	dstBusybox := fmt.Sprintf("%s/library/busybox", dstRegistry)
	dstFoo := fmt.Sprintf("%s/library/foo", dstRegistry)

	for _, tag := range []string{"foo", "bar", "3.10", "3.11", "3.12", "3.13", "3.14", "3.15", "3.16", "3.17", "3.18", "3.19"} {
		err = createImage(dstAlpine, tag)
		require.NoError(t, err)
	}
	for _, tag := range []string{"foo", "bar", "1.1", "1.2", "1.3", "1.4", "1.5", "1.6"} {
		err = createImage(dstBusybox, tag)
		require.NoError(t, err)
	}
	for _, tag := range []string{"foo", "bar"} {
		err = createImage(dstFoo, tag)
		require.NoError(t, err)
	}

	config := apiv1.Config{
		Images: []apiv1.ImageMirror{
			{
				Source:      dstAlpine,
				Destination: "http://" + dstAlpine,
				Match: apiv1.Match{
					Semver: pointer.Pointer(">= 3.17"),
				},
				Purge: &apiv1.Purge{
					Tags:   []string{"foo"},
					Semver: pointer.Pointer("<= 3.15"),
				},
			},
			{
				Source:      dstBusybox,
				Destination: "http://" + dstBusybox,
				Match: apiv1.Match{
					Semver: pointer.Pointer(">= 1.3"),
				},
				Purge: &apiv1.Purge{
					NoMatch: true,
				},
			},
		},
	}

	m := container.New(slog.Default(), config)
	err = m.PurgeUnknown(context.Background())
	require.NoError(t, err)

	tags, err := crane.ListTags(dstAlpine)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"3.17", "3.18", "3.19", "latest"}, tags)

	t.Logf("alpine tags:%s", tags)

	tags, err = crane.ListTags(dstBusybox)
	t.Logf("busybox tags:%s", tags)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"1.3", "1.4", "1.5", "1.6", "latest"}, tags)

	tags, err = crane.ListTags(dstFoo)
	t.Logf("foo tags:%s", tags)
	require.NoError(t, err)
	require.Empty(t, tags)
}
