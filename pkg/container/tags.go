package container

import (
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-containerregistry/pkg/crane"
	apiv1 "github.com/metal-stack/oci-mirror/api/v1"
)

// // image is in the form:  <registry>/<name>:<tag>
// type image struct {
// 	registry string
// 	name     string
// 	tags     []tags
// }

// type tags struct {
// 	tag    string
// 	digest string
// }

type tagsToCopy map[string]string

func (t tagsToCopy) destinationTags() []string {
	var dsts []string
	for _, dst := range t {
		dsts = append(dsts, dst)
	}
	return dsts
}

func (m *mirror) tagMatches(source, tag, semverstring string) (bool, error) {
	c, err := semver.NewConstraint(semverstring)
	if err != nil {
		m.log.Error("unable to parse image match pattern", "error", err)
		return false, err
	}
	v, err := semver.NewVersion(tag)
	if err != nil {
		m.log.Debug("pattern given, ignoring non-semver", "image", source, "tag", tag)
		// This is not treated as an error
		return false, nil // nolint:nilerr
	}
	if c.Check(v) {
		return true, nil
	}
	return false, nil
}

func (m *mirror) getTagsToCopy(image apiv1.ImageMirror, opts []crane.Option) (tagsToCopy, error) {
	var (
		errs       []error
		tagsToCopy = tagsToCopy{}
		semverTags []*semver.Version
	)

	tags, err := crane.ListTags(image.Source, opts...)
	if err != nil {
		m.log.Error("unable to list tags of", "image", image.Source, "error", err)
		return nil, fmt.Errorf("unable to list tags of image:%q error %w", image.Source, err)
	}

	for _, tag := range tags {
		src := image.Source + ":" + tag
		dst := image.Destination + ":" + tag

		if slices.Contains(image.Match.Tags, tag) {
			tagsToCopy[src] = dst
		}

		if image.Match.Semver != nil {
			ok, err := m.tagMatches(image.Source, tag, *image.Match.Semver)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if ok {
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

	if len(errs) > 0 {
		return tagsToCopy, errors.Join(errs...)
	}
	return tagsToCopy, nil
}

func (m *mirror) purge(image string, tags []string, opts []crane.Option) error {
	var errs []error
	for _, tag := range tags {
		tag := tag
		digest, err := crane.Digest(tag, opts...)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		dst := image + "@" + digest
		m.log.Info("purge image", "tag", tag, "dst", dst)
		err = crane.Delete(dst, opts...)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		m.log.Info("purged image", "tag", tag, "dst", dst)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
