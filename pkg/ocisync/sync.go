package ocisync

import (
	"context"
	"errors"
	"log/slog"
	"slices"

	"github.com/google/go-containerregistry/pkg/crane"

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

	var errs []error
	for _, image := range s.config.Images {
		if image.Match.AllTags {
			err := crane.CopyRepository(image.Source, image.Target, crane.WithNoClobber(true))
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

		for _, tag := range tags {
			if slices.Contains(image.Match.Tags, tag) {
				src := image.Source + ":" + tag
				dst := image.Target + ":" + tag
				err := crane.Copy(src, dst, crane.WithNoClobber(true))
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
