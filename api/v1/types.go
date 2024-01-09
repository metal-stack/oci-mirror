package v1

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-containerregistry/pkg/name"
)

// Config defines which images should be mirrored
type Config struct {
	// Images is a list of repositories to mirror
	Images []ImageMirror `json:"images,omitempty"`
	// Registries defines registries with authentication
	Registries map[string]Registry `json:"registries,omitempty"`
}

// Registry defines a destination registry which requires authentication
type Registry struct {
	Auth RegistryAuth `json:"auth,omitempty"`
}

// RegistryAuth is the authentication for a registry
type RegistryAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// ImageMirror defines the mirror configuration for a single Repo
type ImageMirror struct {
	// Source defines from which repo the images should pulled from
	Source string `json:"source,omitempty"`
	// Destination defines the new image repo the Source should be rewritten
	// If prefixed with http:// insecure registry is considered
	Destination string `json:"destination,omitempty"`
	// Match defines which images to mirror
	Match Match `json:"match,omitempty"`
	// Purge defines which images should be purged
	Purge *Purge `json:"purge,omitempty"`
}

type Match struct {
	// AllTags copies all images if true
	AllTags bool `json:"all_tags,omitempty"`
	// Tags is a exact list of tags to mirror from
	Tags []string `json:"tags,omitempty"`
	// Semver defines a semantic version of tags to mirror
	Semver *string `json:"semver,omitempty"`
	// Last defines how many of the latest tags should be mirrored
	Last *int64 `json:"last,omitempty"`
}

type Purge struct {
	// Tags is a exact list of tags to purge
	Tags []string `json:"tags,omitempty"`
	// Semver defines a semantic version of tags to purge
	Semver *string `json:"semver,omitempty"`
}

func (c Config) Validate() error {
	var errs []error
	sources := make(map[string]bool)
	destinations := make(map[string]bool)
	for _, image := range c.Images {
		if image.Source == "" {
			errs = append(errs, fmt.Errorf("image.source is empty:%#v", image))
		}
		if image.Destination == "" {
			errs = append(errs, fmt.Errorf("image.destination is empty:%#v", image))
		}

		if ok := sources[image.Source]; !ok {
			sources[image.Source] = true
		} else {
			errs = append(errs, fmt.Errorf("image source is duplicate:%q", image.Source))
		}

		if ok := destinations[image.Destination]; !ok {
			destinations[image.Destination] = true
		} else {
			errs = append(errs, fmt.Errorf("image destination is duplicate:%q", image.Destination))
		}

		if ok := destinations[image.Source]; ok {
			errs = append(errs, fmt.Errorf("image source is already specified as destination:%q", image.Source))
		}

		if ok := sources[image.Destination]; ok {
			errs = append(errs, fmt.Errorf("image destination is already specified as source:%q", image.Destination))
		}

		if image.Source == image.Destination {
			errs = append(errs, fmt.Errorf("source and destination are equal %q:%q", image.Source, image.Destination))
		}

		match := image.Match
		if !match.AllTags && len(match.Tags) == 0 && match.Semver == nil && match.Last == nil {
			errs = append(errs, fmt.Errorf("no image.match criteria given"))
		}

		if image.Match.Semver != nil {
			_, err := semver.NewConstraint(*image.Match.Semver)
			if err != nil {
				errs = append(errs, fmt.Errorf("image.match.semver is invalid, image source:%q, semver:%q %w", image.Source, *image.Match.Semver, err))
			}
		}

		if image.Purge != nil && image.Purge.Semver != nil {
			_, err := semver.NewConstraint(*image.Match.Semver)
			if err != nil {
				errs = append(errs, fmt.Errorf("image.purge.semver is invalid, image source:%q, semver:%q %w", image.Source, *image.Purge.Semver, err))
			}
		}

		srcRef, err := name.ParseReference(image.Source)
		if err != nil {
			errs = append(errs, err)
		} else {
			if !strings.Contains(srcRef.Name(), ":latest") {
				errs = append(errs, fmt.Errorf("image source contains a tag:%q", srcRef.Name()))
			}
		}

		if strings.HasPrefix(image.Destination, "http://") {
			image.Destination = strings.ReplaceAll(image.Destination, "http://", "")
		}

		dstRef, err := name.ParseReference(image.Destination)
		if err != nil {
			errs = append(errs, err)
		} else {
			if !strings.Contains(dstRef.Name(), ":latest") {
				errs = append(errs, fmt.Errorf("image destination contains a tag:%q", dstRef.Name()))
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
