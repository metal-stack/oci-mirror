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
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMirror(t *testing.T) {
	srcip, srcport, err := startRegistry()
	require.NoError(t, err)

	dstip, dstport, err := startRegistry()
	require.NoError(t, err)

	srcAlpine := fmt.Sprintf("%s:%d/library/alpine", srcip, srcport)
	dstAlpine := fmt.Sprintf("%s:%d/library/alpine", dstip, dstport)
	err = createImage(srcAlpine, "3.18")
	require.NoError(t, err)

	srcBusybox := fmt.Sprintf("%s:%d/library/busybox", srcip, srcport)
	dstBusybox := fmt.Sprintf("%s:%d/library/busybox", dstip, dstport)

	err = createImage(srcBusybox, "1.35.0", "1.36.0")
	require.NoError(t, err)

	srcFoo := fmt.Sprintf("%s:%d/library/foo", srcip, srcport)
	dstFoo := fmt.Sprintf("%s:%d/library/foo", dstip, dstport)
	err = createImage(srcFoo, "1.0.0", "1.0.1", "1.0.2")
	require.NoError(t, err)

	config := apiv1.Config{
		Images: []apiv1.ImageMirror{
			{
				Source:      srcAlpine,
				Destination: dstAlpine,
				Match: apiv1.Match{
					Tags: []string{
						"3.18",
						"latest",
					},
				},
			},
			{
				Source:      srcBusybox,
				Destination: dstBusybox,
				Match: apiv1.Match{
					Pattern: pointer.Pointer(">= 1.35"),
				},
			},
			{
				Source:      srcFoo,
				Destination: dstFoo,
				Match: apiv1.Match{
					Last: pointer.Pointer(int64(2)),
				},
			},
		},
	}

	m := mirror.New(slog.Default(), config)
	err = m.Mirror(context.Background())
	require.NoError(t, err)

	tags, err := crane.ListTags(fmt.Sprintf("%s:%d/library/alpine", dstip, dstport))
	require.NoError(t, err)
	require.Len(t, tags, 2)
	require.Equal(t, []string{"3.18", "latest"}, tags)

	tags, err = crane.ListTags(fmt.Sprintf("%s:%d/library/busybox", dstip, dstport))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"1.35.0", "1.36.0"}, tags)

	tags, err = crane.ListTags(fmt.Sprintf("%s:%d/library/foo", dstip, dstport))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"1.0.1", "1.0.2"}, tags)
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

func createImage(name string, tags ...string) error {
	img, err := crane.Image(map[string][]byte{})
	if err != nil {
		return err
	}
	err = crane.Push(img, name)
	if err != nil {
		return err
	}
	for _, tag := range tags {
		err := crane.Push(img, name+":"+tag)
		if err != nil {
			return err
		}
	}
	return nil
}