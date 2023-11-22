package ocisync

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	apiv1 "github.com/metal-stack/oci-mirror/api/v1"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestSync(t *testing.T) {
	srcip, srcport, err := startRegistry()
	require.NoError(t, err)

	dstip, dstport, err := startRegistry()
	require.NoError(t, err)

	img, err := crane.Image(map[string][]byte{})
	require.NoError(t, err)
	srcAlpine := fmt.Sprintf("%s:%d/library/alpine", srcip, srcport)
	err = crane.Push(img, srcAlpine)
	require.NoError(t, err)
	err = crane.Push(img, srcAlpine+":3.18")
	require.NoError(t, err)

	srcBusybox := fmt.Sprintf("%s:%d/library/busybox", srcip, srcport)
	err = crane.Push(img, srcBusybox)
	require.NoError(t, err)
	err = crane.Push(img, srcBusybox+":1.35.0")
	require.NoError(t, err)
	err = crane.Push(img, srcBusybox+":1.36.0")
	require.NoError(t, err)

	config := apiv1.SyncConfig{
		Images: []apiv1.ImageSync{
			{
				Source:      srcAlpine,
				Destination: fmt.Sprintf("%s:%d/library/alpine", dstip, dstport),
				Match: apiv1.Match{
					Tags: []string{
						"3.18",
						"latest",
					},
				},
			},
			{
				Source:      srcBusybox,
				Destination: fmt.Sprintf("%s:%d/library/busybox", dstip, dstport),
				Match: apiv1.Match{
					Pattern: pointer.Pointer(">= 1.35"),
				},
			},
		},
	}

	syncher := New(slog.Default(), config)
	err = syncher.Sync(context.Background())
	require.NoError(t, err)

	tags, err := crane.ListTags(fmt.Sprintf("%s:%d/library/alpine", dstip, dstport))
	require.NoError(t, err)
	require.Len(t, tags, 2)
	require.Equal(t, []string{"3.18", "latest"}, tags)

	tags, err = crane.ListTags(fmt.Sprintf("%s:%d/library/busybox", dstip, dstport))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"1.35.0", "1.36.0"}, tags)
}

func startRegistry() (string, int, error) {
	ctx := context.Background()
	var (
		c   testcontainers.Container
		err error
	)

	req := testcontainers.ContainerRequest{
		Image:        "registry:2",
		ExposedPorts: []string{"5000/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForLog("listening on"),
			wait.ForListeningPort("5000/tcp"),
		),
	}
	c, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", 0, err
	}

	ip, err := c.Host(ctx)
	if err != nil {
		return ip, 0, err
	}
	port, err := c.MappedPort(ctx, "5000")
	if err != nil {
		return ip, port.Int(), err
	}

	return ip, port.Int(), nil
}
