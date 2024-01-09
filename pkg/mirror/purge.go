package mirror

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
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

			tagsToCopy, err := m.getTagsToCopy(image)
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

	// crane.Catalog()
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
