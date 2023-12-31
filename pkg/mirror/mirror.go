package mirror

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"sort"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/Masterminds/semver/v3"
	apiv1 "github.com/metal-stack/oci-mirror/api/v1"
)

type mirror struct {
	log    *slog.Logger
	config apiv1.Config
}

func New(log *slog.Logger, config apiv1.Config) *mirror {
	return &mirror{
		log:    log,
		config: config,
	}
}

func (m *mirror) Mirror(ctx context.Context) error {
	var (
		errs []error
	)
	for _, image := range m.config.Images {
		var (
			err  error
			opts []crane.Option
		)
		if strings.HasPrefix(image.Destination, "http://") {
			opts = append(opts, crane.Insecure)
			image.Destination = strings.ReplaceAll(image.Destination, "http://", "")
		}

		auth, err := m.getAuthOption(image)
		if err != nil {
			m.log.Warn("unable detect auth, continue unauthenticated", "error", err)
		}
		if auth != nil {
			opts = append(opts, auth)
		}

		m.log.Info("consider mirror from", "source", image.Source, "destination", image.Destination)

		if _, err := name.ParseReference(image.Source); err != nil {
			m.log.Error("given image source is malformed", "image", image.Source, "error", err)
			errs = append(errs, err)
			continue
		}

		if _, err := name.ParseReference(image.Destination); err != nil {
			m.log.Error("given image destination is malformed", "image", image.Destination, "error", err)
			errs = append(errs, err)
			continue
		}

		if image.Match.AllTags {
			m.log.Info("mirror all tags from", "source", image.Source, "destination", image.Destination)
			err := crane.CopyRepository(image.Source, image.Destination, opts...)
			if err != nil {
				m.log.Error("unable to copy all images", "image", image.Source, "error", err)
				errs = append(errs, err)
			}
			continue
		}

		tags, err := crane.ListTags(image.Source)
		if err != nil {
			m.log.Error("unable to list tags of", "image", image.Source, "error", err)
			errs = append(errs, err)
			continue
		}

		var (
			tagsToCopy = make(map[string]string)
			semverTags []*semver.Version
		)

		for _, tag := range tags {
			src := image.Source + ":" + tag
			dst := image.Destination + ":" + tag

			if slices.Contains(image.Match.Tags, tag) {
				tagsToCopy[src] = dst
			}

			if image.Match.Semver != nil {
				c, err := semver.NewConstraint(*image.Match.Semver)
				if err != nil {
					m.log.Error("unable to parse image match pattern", "error", err)
					errs = append(errs, err)
					continue
				}
				v, err := semver.NewVersion(tag)
				if err != nil {
					m.log.Debug("pattern given, ignoring non-semver", "image", image.Source, "tag", tag)
					// This is not treated as an error
					continue
				}
				if c.Check(v) {
					tagsToCopy[src] = dst
				}
			}

			if image.Match.Last != nil && *image.Match.Last > 0 {
				v, err := semver.NewVersion(tag)
				if err != nil {
					continue
				}
				semverTags = append(semverTags, v)
			}
		}

		// If only the last n images
		sort.Sort(semver.Collection(semverTags))

		if image.Match.Last != nil && semverTags != nil {
			for _, v := range semverTags[len(semverTags)-int(*image.Match.Last):] {
				if slices.Contains(tags, v.String()) {
					src := image.Source + ":" + v.String()
					dst := image.Destination + ":" + v.String()
					tagsToCopy[src] = dst
				}
			}
		}

		for src, dst := range tagsToCopy {
			if !strings.HasSuffix(dst, ":latest") {
				opts = append(opts, crane.WithNoClobber(false))
			}
			m.log.Info("mirror from", "source", src, "destination", dst)
			rawmanifest, err := crane.Manifest(src, opts...)
			if err != nil {
				m.log.Error("unable to read image manifest", "error", err)
				errs = append(errs, err)
				continue
			}
			manifest := v1.Manifest{}
			if err := json.Unmarshal(rawmanifest, &manifest); err != nil {
				m.log.Error("unable to decode image manifest", "error", err)
				errs = append(errs, err)
				continue
			}
			if manifest.SchemaVersion < 2 {
				m.log.Warn("image manifest scheme version to low, ignoring", "image", src, "scheme version", manifest.SchemaVersion)
				continue
			}
			err = crane.Copy(src, dst, opts...)
			if err != nil {
				m.log.Error("unable to copy", "source", src, "dst", dst, "error", err)
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (m *mirror) getAuthOption(image apiv1.ImageMirror) (crane.Option, error) {
	dstRef, err := name.ParseReference(image.Destination)
	if err != nil {
		return nil, err
	}
	registryName := dstRef.Context().Registry.Name()
	registry, ok := m.config.Registries[registryName]
	if !ok {
		return nil, nil
	}
	auth := crane.WithAuth(&authn.Basic{
		Username: registry.Auth.Username,
		Password: registry.Auth.Password,
	})
	return auth, nil
}
