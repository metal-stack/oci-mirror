package container

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
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
			err     error
			srcOpts []crane.Option
			dstOpts []crane.Option
		)

		srcOpts = append(srcOpts, crane.WithContext(ctx))
		dstOpts, err = m.ensureAuthOption(&image)
		if err != nil {
			m.log.Warn("unable detect auth, continue unauthenticated", "error", err)
		}
		dstOpts = append(dstOpts, crane.WithContext(ctx))

		m.log.Info("consider mirror from", "source", image.Source, "destination", image.Destination)

		if image.Match.AllTags {
			m.log.Info("mirror all tags from", "source", image.Source, "destination", image.Destination)
			var tags []string
			err := m.withRetry("list_tags", image.Source, func() error {
				var err2 error
				tags, err2 = crane.ListTags(image.Source, srcOpts...)
				return err2
			})
			if err != nil {
				m.log.Error("unable to list tags of", "image", image.Source, "error", err)
				errs = append(errs, err)
				continue
			}

			for _, tag := range tags {
				src := image.Source + ":" + tag
				dst := image.Destination + ":" + tag
				err = m.mirrorTag(src, dst, srcOpts, dstOpts)
				if err != nil {
					errs = append(errs, err)
				}
			}
			continue
		}

		tagsToCopy, err := m.getTagsToCopy(image, srcOpts)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		for src, dst := range tagsToCopy {
			err = m.mirrorTag(src, dst, srcOpts, dstOpts)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (m *mirror) mirrorTag(src, dst string, srcOpts, dstOpts []crane.Option) error {
	m.log.Info("mirror from", "source", src, "destination", dst)

	var rawmanifest []byte
	err := m.withRetry("read_manifest", src, func() error {
		var err2 error
		rawmanifest, err2 = crane.Manifest(src, srcOpts...)
		return err2
	})
	if err != nil {
		m.log.Error("unable to read image manifest", "error", err)
		return err
	}

	manifest := v1.Manifest{}
	if err := json.Unmarshal(rawmanifest, &manifest); err != nil {
		m.log.Error("unable to decode image manifest", "error", err)
		return err
	}
	if manifest.SchemaVersion < 2 {
		m.log.Warn("image manifest scheme version to low, ignoring", "image", src, "scheme version", manifest.SchemaVersion)
		return nil
	}

	tagOpts := slices.Clone(dstOpts)
	if !strings.HasSuffix(dst, ":latest") {
		tagOpts = append(tagOpts, crane.WithNoClobber(false))
	}

	_, err = crane.Digest(dst, tagOpts...)
	if err == nil && !strings.HasSuffix(dst, ":latest") {
		m.log.Info("image already exists, skip copy", "image", dst)
		return nil
	}

	m.log.Info("copy image", "source", src, "destination", dst)
	var img v1.Image
	err = m.withRetry("pull_image", src, func() error {
		var err2 error
		img, err2 = crane.Pull(src, srcOpts...)
		return err2
	})
	if err != nil {
		m.log.Error("unable to pull", "source", src, "error", err)
		return err
	}

	err = m.withRetry("push_image", dst, func() error {
		return crane.Push(img, dst, tagOpts...)
	})
	if err != nil {
		m.log.Error("unable to push", "source", src, "dst", dst, "error", err)
		return err
	}

	return nil
}
