package mirror

import "net/url"

// registryTarget defines if the Registry is a source or destination registry
type registryTarget string

const (
	// sourceRegistry is a registry where images are pulled from
	sourceRegistry = registryTarget("source")
	// destinationRegistry is a registry where images are pushed to
	destinationRegistry = registryTarget("destination")
)

// affectedRegistries returns a slice of all registries of sources and destinations
func (m *mirror) affectedRegistries(target registryTarget) ([]string, error) {
	var result []string
	for _, image := range m.config.Images {
		registry := image.Destination
		if target == sourceRegistry {
			registry = image.Source
		}
		parsed, err := url.Parse(registry)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed.Host)
	}
	return result, nil
}
