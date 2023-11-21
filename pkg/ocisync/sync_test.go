package ocisync

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	apiv1 "github.com/metal-stack/oci-mirror/api/v1"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	registryOnce sync.Once
)

func TestSync(t *testing.T) {
	ip, port, err := startRegistry()

	require.NoError(t, err)

	config := apiv1.SyncConfig{
		Images: []apiv1.ImageSync{
			{
				Source: "alpine",
				Target: fmt.Sprintf("%s:%d/library/alpine", ip, port),
				Match: apiv1.Match{
					Tags: []string{
						"3.18",
						"latest",
					},
				},
			},
		},
	}

	syncher := New(slog.Default(), config)
	err = syncher.Sync(context.Background())
	require.NoError(t, err)

	tags, err := crane.ListTags(fmt.Sprintf("%s:%d/library/alpine", ip, port))
	require.NoError(t, err)
	require.Len(t, tags, 2)
	require.Equal(t, []string{"latest"}, tags)

}

func startRegistry() (string, int, error) {
	ctx := context.Background()
	var c testcontainers.Container
	registryOnce.Do(func() {
		var err error
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
			panic(err)
		}
	})
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
