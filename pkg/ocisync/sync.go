package ocisync

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sort"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"

	"github.com/Masterminds/semver/v3"
	apiv1 "github.com/metal-stack/oci-mirror/api/v1"
)

type syncher struct {
	log    *slog.Logger
	config apiv1.SyncConfig
}

func New(log *slog.Logger, config apiv1.SyncConfig) *syncher {
	return &syncher{
		log:    log,
		config: config,
	}
}

func (s *syncher) Sync(ctx context.Context) error {
	var opts []crane.Option

	var errs []error
	for _, image := range s.config.Images {
		// Refactor auth
		dstRef, err := name.ParseReference(image.Destination)
		if err != nil {
			return err
		}
		registryName := dstRef.Context().Registry.Name()
		registry, ok := s.config.Registries[registryName]
		if ok {
			auth := crane.WithAuth(&authn.Basic{
				Username: registry.Auth.Username,
				Password: registry.Auth.Password,
			})
			opts = append(opts, auth)
		}
		s.log.Info("registry", "name", dstRef.Context().Registry.Name())
		// crane.WithAuth()
		if image.Match.AllTags {
			opts = append(opts, crane.WithNoClobber(true))
			err := crane.CopyRepository(image.Source, image.Destination, opts...)
			if err != nil {
				s.log.Error("unable to copy all images", "image", image.Source, "error", err)
				errs = append(errs, err)
			}
			continue
		}

		tags, err := crane.ListTags(image.Source)
		if err != nil {
			s.log.Error("unable to list tags of", "image", image.Source, "error", err)
			errs = append(errs, err)
			continue
		}

		var tagsToCopy = make(map[string]string)
		var semverTags []*semver.Version

		for _, tag := range tags {
			src := image.Source + ":" + tag
			dst := image.Destination + ":" + tag

			if slices.Contains(image.Match.Tags, tag) {
				tagsToCopy[src] = dst
			}

			if image.Match.Pattern != nil {
				c, err := semver.NewConstraint(*image.Match.Pattern)
				if err != nil {
					s.log.Error("unable to parse image match pattern", "error", err)
					errs = append(errs, err)
					continue
				}
				v, err := semver.NewVersion(tag)
				if err != nil {
					s.log.Debug("pattern given, ignoring non-semver", "image", image.Source, "tag", tag)
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

		sort.Sort(semver.Collection(semverTags))

		if image.Match.Last != nil {
			for _, v := range semverTags[len(semverTags)-int(*image.Match.Last):] {
				if slices.Contains(tags, v.String()) {
					src := image.Source + ":" + v.String()
					dst := image.Destination + ":" + v.String()
					tagsToCopy[src] = dst
				}
			}
		}

		for src, dst := range tagsToCopy {
			err := crane.Copy(src, dst, crane.WithNoClobber(true), crane.WithContext(ctx))
			if err != nil {
				s.log.Error("unable to copy", "source", src, "dst", dst, "error", err)
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
