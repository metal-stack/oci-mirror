package v1

// SyncConfig defines which images should be synced
type SyncConfig struct {
	// Images is a list of repositories to sync
	Images []ImageSync `json:"images,omitempty"`
	// Registries defines registries with authentication
	Registries map[string]Registry `json:"registries,omitempty"`
}

// Registry Defines a registry which requires authentication
type Registry struct {
	Auth RegistryAuth `json:"auth,omitempty"`
}

// RegistryAuth is the authentication for a registry
type RegistryAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// ImageSync defines the sync configuration for a single Repo
type ImageSync struct {
	// Source defines from which repo the images should pulled from
	Source string `json:"source,omitempty"`
	// Destination defines the new image repo the Source should be rewritten
	Destination string `json:"destination,omitempty"`
	// Match defines which images to sync
	Match Match `json:"match,omitempty"`
}

type Match struct {
	// AllTags copies all images if true
	AllTags bool `json:"all_tags,omitempty"`
	// Tags is a exact list of tags to sync from
	Tags []string `json:"tags,omitempty"`
	// Pattern defines a pattern of tags to sync
	Pattern *string `json:"pattern,omitempty"`
	// Last defines how many of the latest tags should be synced
	Last *int64 `json:"last,omitempty"`
}
