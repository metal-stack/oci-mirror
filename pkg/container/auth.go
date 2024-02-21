package container

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	apiv1 "github.com/metal-stack/oci-mirror/api/v1"
)

func (m *mirror) ensureAuthOption(image *apiv1.ImageMirror) ([]crane.Option, error) {
	var opts []crane.Option
	if image == nil {
		return opts, fmt.Errorf("image is nil")
	}
	if strings.HasPrefix(image.Destination, "http://") {
		opts = append(opts, crane.Insecure)
		image.Destination = strings.ReplaceAll(image.Destination, "http://", "")
	}
	dstRef, err := name.ParseReference(image.Destination)
	if err != nil {
		return opts, err
	}
	registryName := dstRef.Context().Registry.Name()
	registry, ok := m.config.Registries[registryName]
	if !ok {
		return opts, nil
	}
	auth := crane.WithAuth(&authn.Basic{
		Username: registry.Auth.Username,
		Password: registry.Auth.Password,
	})
	opts = append(opts, auth)
	return opts, nil
}
