package mirror

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-containerregistry/pkg/crane"
)

func (m *mirror) Purge(ctx context.Context) error {
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

		if image.Purge == nil {
			continue
		}

		tags, err := crane.ListTags(image.Destination)
		if err != nil {
			m.log.Error("unable to list tags of", "image", image.Source, "error", err)
			errs = append(errs, err)
			continue
		}

		var (
			tagsToPurge []string
		)

		for _, tag := range tags {
			dst := image.Destination + ":" + tag

			if slices.Contains(image.Purge.Tags, tag) {
				tagsToPurge = append(tagsToPurge, dst)
			}

			if image.Purge.Semver != nil {
				c, err := semver.NewConstraint(*image.Purge.Semver)
				if err != nil {
					m.log.Error("unable to parse image purge pattern", "error", err)
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
					tagsToPurge = append(tagsToPurge, dst)
				}
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
			m.log.Info("purge image", "dst", dst)
			err = crane.Delete(dst, opts...)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			m.log.Info("purged image", "dst", dst)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
