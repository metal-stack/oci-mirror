package mirror_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/foomo/htpasswd"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	apiv1 "github.com/metal-stack/oci-mirror/api/v1"
	"github.com/metal-stack/oci-mirror/pkg/mirror"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMirror(t *testing.T) {
	srcip, srcport, err := startRegistry(nil, nil, nil)
	require.NoError(t, err)
	srcRegistry := fmt.Sprintf("%s:%d", srcip, srcport)

	dstip, dstport, err := startRegistry(nil, nil, nil)
	require.NoError(t, err)
	dstRegistry := fmt.Sprintf("%s:%d", dstip, dstport)

	f, err := os.CreateTemp("", "htpasswd")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	err = htpasswd.SetPassword(f.Name(), "user", "secret", htpasswd.HashBCrypt)
	require.NoError(t, err)

	env := map[string]string{
		"REGISTRY_AUTH":                "htpasswd",
		"REGISTRY_AUTH_HTPASSWD_REALM": "registry-login",
		"REGISTRY_AUTH_HTPASSWD_PATH":  "/htpasswd",
	}
	authtip, authport, err := startRegistry(env, pointer.Pointer(f.Name()), pointer.Pointer("/htpasswd"))
	require.NoError(t, err)
	authRegistry := fmt.Sprintf("%s:%d", authtip, authport)

	srcAlpine := fmt.Sprintf("%s/library/alpine", srcRegistry)
	dstAlpine := fmt.Sprintf("%s/library/alpine", dstRegistry)
	err = createImage(srcAlpine, "3.18")
	require.NoError(t, err)

	srcBusybox := fmt.Sprintf("%s/library/busybox", srcRegistry)
	dstBusybox := fmt.Sprintf("%s/library/busybox", dstRegistry)

	err = createImage(srcBusybox, "1.35.0", "1.36.0")
	require.NoError(t, err)

	srcFoo := fmt.Sprintf("%s/library/foo", srcRegistry)
	dstFoo := fmt.Sprintf("%s/library/foo", dstRegistry)
	err = createImage(srcFoo, "1.0.0", "1.0.1", "1.0.2")
	require.NoError(t, err)

	dstAuthFoo := fmt.Sprintf("%s/library/foo", authRegistry)

	config := apiv1.Config{
		Registries: map[string]apiv1.Registry{
			authRegistry: {
				Auth: apiv1.RegistryAuth{
					Username: "user",
					Password: "secret",
				},
			},
		},
		Images: []apiv1.ImageMirror{
			{
				Source:      srcAlpine,
				Destination: "http://" + dstAlpine,
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
					Semver: pointer.Pointer(">= 1.35"),
				},
			},
			{
				Source:      srcFoo,
				Destination: dstFoo,
				Match: apiv1.Match{
					Last: pointer.Pointer(int64(2)),
				},
			},
			{
				Source:      srcFoo,
				Destination: dstAuthFoo,
				Match: apiv1.Match{
					Last: pointer.Pointer(int64(2)),
				},
			},
		},
	}

	m := mirror.New(slog.Default(), config)
	err = m.Mirror(context.Background())
	require.NoError(t, err)

	tags, err := crane.ListTags(dstAlpine)
	require.NoError(t, err)
	require.Len(t, tags, 2)
	require.Equal(t, []string{"3.18", "latest"}, tags)

	tags, err = crane.ListTags(dstBusybox)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"1.35.0", "1.36.0"}, tags)

	tags, err = crane.ListTags(dstFoo)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"1.0.1", "1.0.2"}, tags)

	tags, err = crane.ListTags(dstAuthFoo, crane.WithAuth(&authn.Basic{Username: "user", Password: "secret"}))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"1.0.1", "1.0.2"}, tags)
}

func startRegistry(env map[string]string, src, dst *string) (string, int, error) {
	ctx := context.Background()
	var (
		c   testcontainers.Container
		err error
	)

	req := testcontainers.ContainerRequest{
		Image:        "registry:2",
		ExposedPorts: []string{"5000/tcp"},
		Env:          env,
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
	if src != nil && dst != nil {
		err = c.CopyFileToContainer(ctx, *src, *dst, 0o777)
		if err != nil {
			return "", 0, err
		}
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
