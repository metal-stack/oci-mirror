package container

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
)

func (m *mirror) Purge(ctx context.Context) error {
	var (
		errs []error
	)
	for _, image := range m.config.Images {
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

		tags, err := crane.ListTags(image.Destination, opts...)
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

		err = m.purge(image.Destination, tagsToPurge, opts)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (m *mirror) PurgeUnknown(ctx context.Context) error {
	var (
		existing  []string
		allowed   []string
		purgeable []string
	)
	registries, err := m.affectedRegistries(destinationRegistry)
	if err != nil {
		return err
	}

	// Collect crane options (insecure + auth) per destination registry from the
	// configured images, otherwise crane would talk https to an insecure registry.
	optsByRegistry := make(map[string][]crane.Option, len(registries))
	for _, image := range m.config.Images {
		img := image
		opts, err := m.ensureAuthOption(&img)
		if err != nil {
			m.log.Warn("unable detect auth, continue unauthenticated", "error", err)
		}
		ref, err := name.ParseReference(img.Destination)
		if err != nil {
			continue
		}
		optsByRegistry[ref.Context().Registry.Name()] = append(opts, crane.WithContext(ctx))
	}

	for _, registry := range registries {
		opts := optsByRegistry[registry]
		catalog, err := crane.Catalog(registry, opts...)
		if err != nil {
			return err
		}
		for _, c := range catalog {
			image := fmt.Sprintf("%s/%s", registry, c)

			tags, err := crane.ListTags(image, opts...)
			if err != nil {
				return err
			}
			for _, tag := range tags {
				// never purge latest
				if tag == "latest" {
					continue
				}
				existing = append(existing, image+":"+tag)
			}
		}
	}
	for _, image := range m.config.Images {
		var (
			err  error
			opts []crane.Option
		)
		opts, err = m.ensureAuthOption(&image)
		if err != nil {
			m.log.Warn("unable detect auth, continue unauthenticated", "error", err)
		}
		opts = append(opts, crane.WithContext(ctx))
		// FIXME howto handle match.Alltags

		tagsToCopy, err := m.getTagsToCopy(image, opts)
		if err != nil {
			return fmt.Errorf("unable to get tags to copy:%w", err)
		}
		allowed = append(allowed, tagsToCopy.destinationTags()...)
	}

	for _, image := range existing {
		if !slices.Contains(allowed, image) {
			purgeable = append(purgeable, image)
		}
	}

	for _, tag := range purgeable {
		m.log.Info("purge unknown", "images", tag)
		// tag is the whole image refspec, split away the tag to get the image alone
		lastInd := strings.LastIndex(tag, ":")
		image := tag[:lastInd]

		ref, err := name.ParseReference(image)
		if err != nil {
			return err
		}
		err = m.purge(image, []string{tag}, optsByRegistry[ref.Context().Registry.Name()])
		if err != nil {
			return err
		}
	}
	return nil
}
