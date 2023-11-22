package v1

// SyncConfig defines which images should be synced
type SyncConfig struct {
	// Images is a list of repositories to sync
	Images []ImageSync
}

// ImageSync defines the sync configuration for a single Repo
type ImageSync struct {
	// Source defines from which repo the images should pulled from
	Source string
	// Destination defines the new image repo the Source should be rewritten
	Destination string
	// Match defines which images to sync
	Match Match
}

type Match struct {
	// AllTags copies all images if true
	AllTags bool
	// Tags is a exact list of tags to sync from
	Tags []string
	// Pattern defines a pattern of tags to sync
	Pattern *string
	// Last defines how many of the latest count tags should be synced
	Last *int64
}
