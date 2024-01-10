package mirror

import (
	"context"
	"errors"
	"net/url"
	"slices"

	"github.com/google/go-containerregistry/pkg/crane"
)

func (m *mirror) Purge(ctx context.Context) error {
	var (
		errs []error
	)
	for _, image := range m.config.Images {
		image := image
		if image.Purge == nil {
			continue
		}

		var (
			err         error
			opts        []crane.Option
			tagsToPurge []string
		)

		opts, err = m.ensureAuthOption(&image)
		if err != nil {
			m.log.Warn("unable detect auth, continue unauthenticated", "error", err)
		}
		opts = append(opts, crane.WithContext(ctx))

		tags, err := crane.ListTags(image.Destination)
		if err != nil {
			m.log.Error("unable to list tags of", "image", image.Source, "error", err)
			errs = append(errs, err)
			continue
		}

		for _, tag := range tags {
			// never purge latest
			if tag == "latest" {
				continue
			}
			dst := image.Destination + ":" + tag

			if slices.Contains(image.Purge.Tags, tag) {
				tagsToPurge = append(tagsToPurge, dst)
			}

			if image.Purge.Semver != nil {
				ok, err := m.tagMatches(image.Destination, tag, *image.Purge.Semver)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				if ok {
					tagsToPurge = append(tagsToPurge, dst)
				}
			}

			if !image.Purge.NoMatch {
				continue
			}

			tagsToCopy, err := m.getTagsToCopy(image, opts)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if !slices.Contains(tagsToCopy.destinationTags(), dst) {
				tagsToPurge = append(tagsToPurge, dst)
			}

		}

		for _, tag := range tagsToPurge {
			tag := tag
			digest, err := crane.Digest(tag)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			dst := image.Destination + "@" + digest
			m.log.Info("purge image", "tag", tag, "dst", dst)
			err = crane.Delete(dst, opts...)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			m.log.Info("purged image", "tag", tag, "dst", dst)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (m *mirror) PurgeUnknown(ctx context.Context) error {
	var (
		errs       []error
		catalog    []string
		registries = make(map[string]bool)
	)

	for _, image := range m.config.Images {
		parsed, err := url.Parse(image.Destination)
		if err != nil {
			return err
		}

		registries[parsed.Host] = true
	}
	for registry := range registries {
		c, err := crane.Catalog(registry)
		if err != nil {
			return err
		}
		catalog = append(catalog, c...)
	}
	m.log.Info("catalog", "content", catalog)

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
