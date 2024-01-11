package container

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

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
		existing []string
		allowed  []string
		purgable []string
	)
	registries, err := m.affectedRegistries(destinationRegistry)
	if err != nil {
		return err
	}
	for _, registry := range registries {
		catalog, err := crane.Catalog(registry)
		if err != nil {
			return err
		}
		for _, c := range catalog {
			image := fmt.Sprintf("%s/%s", registry, c)

			tags, err := crane.ListTags(image)
			if err != nil {
				return err
			}
			for _, tag := range tags {
				tag := tag
				// never purge latest
				if tag == "latest" {
					continue
				}
				existing = append(existing, image+":"+tag)
			}
		}
	}
	m.log.Info("existing", "images", existing)

	for _, image := range m.config.Images {
		image := image
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
	m.log.Info("allowed", "images", allowed)

	for _, image := range existing {
		if !slices.Contains(allowed, image) {
			purgable = append(purgable, image)
		}
	}
	m.log.Info("purgable", "images", purgable)

	for _, tag := range purgable {
		// tag is the whole image refspec, split away the tag to get the image alone
		lastInd := strings.LastIndex(tag, ":")
		image := tag[:lastInd]
		err := m.purge(image, []string{tag}, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
