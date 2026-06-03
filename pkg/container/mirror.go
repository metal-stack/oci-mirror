package container

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	apiv1 "github.com/metal-stack/oci-mirror/api/v1"
)

type mirror struct {
	log         *slog.Logger
	config      apiv1.Config
	retryPolicy *RetryPolicy
}

func New(log *slog.Logger, config apiv1.Config, retryPolicy *RetryPolicy) *mirror {
	return &mirror{
		log:         log,
		config:      config,
		retryPolicy: retryPolicy,
	}
}

func (m *mirror) Mirror(ctx context.Context) error {
	var (
		errs []error
	)
	m.log.Debug("start mirroring images", "retryPolicy", m.retryPolicy)
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

		m.log.Info("consider mirror from", "source", image.Source, "destination", image.Destination)

		if image.Match.AllTags {
			m.log.Info("mirror all tags from", "source", image.Source, "destination", image.Destination)
			err := m.withRetry("copy_repository", image.Source, func() error {
				return crane.CopyRepository(image.Source, image.Destination, opts...)
			})
			if err != nil {
				m.log.Error("unable to copy all images", "image", image.Source, "error", err)
				errs = append(errs, err)
			}
			continue
		}

		tagsToCopy, err := m.getTagsToCopy(image, opts)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		for src, dst := range tagsToCopy {
			if !strings.HasSuffix(dst, ":latest") {
				opts = append(opts, crane.WithNoClobber(false))
			}
			m.log.Info("mirror from", "source", src, "destination", dst)
			var rawmanifest []byte
			err := m.withRetry("read_manifest", src, func() error {
				var err2 error
				rawmanifest, err2 = crane.Manifest(src, opts...)
				return err2
			})
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

			_, err = crane.Digest(dst, opts...)
			if err == nil && !strings.HasSuffix(dst, ":latest") {
				m.log.Info("image already exists, skip copy", "image", dst)
				continue
			}

			m.log.Info("copy image", "source", src, "destination", dst)
			err = m.withRetry("copy_image", src, func() error {
				return crane.Copy(src, dst, opts...)
			})
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
