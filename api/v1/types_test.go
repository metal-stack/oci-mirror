package v1

import (
	"testing"
)

func TestConfig_Validate(t *testing.T) {

	tests := []struct {
		name       string
		Images     []ImageMirror
		Registries map[string]Registry
		wantErr    bool
	}{
		{
			name: "duplicate source",
			Images: []ImageMirror{
				{Source: "abc", Destination: "cde", Match: Match{Tags: []string{"latest"}}},
				{Source: "abc", Destination: "efg", Match: Match{Tags: []string{"latest"}}},
			},
			wantErr: true,
		},
		{
			name: "duplicate destination",
			Images: []ImageMirror{
				{Source: "cde", Destination: "abc", Match: Match{Tags: []string{"latest"}}},
				{Source: "efg", Destination: "abc", Match: Match{Tags: []string{"latest"}}},
			},
			wantErr: true,
		},
		{
			name: "source and destination are equal",
			Images: []ImageMirror{
				{Source: "abc", Destination: "abc", Match: Match{Tags: []string{"latest"}}},
			},
			wantErr: true,
		},
		{
			name: "source empty",
			Images: []ImageMirror{
				{Source: "", Destination: "abc", Match: Match{Tags: []string{"latest"}}},
			},
			wantErr: true,
		},
		{
			name: "destination empty",
			Images: []ImageMirror{
				{Source: "abc", Destination: "", Match: Match{Tags: []string{"latest"}}},
			},
			wantErr: true,
		},
		{
			name: "invalid match semver",
			Images: []ImageMirror{
				{
					Source:      "abc",
					Destination: "abc",
					Match: Match{
						Semver: new("abc"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid purge semver",
			Images: []ImageMirror{
				{
					Source:      "abc",
					Destination: "abc",
					Purge: &Purge{
						Semver: new("abc"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "image cde is used in two images",
			Images: []ImageMirror{
				{Source: "abc", Destination: "cde", Match: Match{Tags: []string{"latest"}}},
				{Source: "cde", Destination: "efg", Match: Match{Tags: []string{"latest"}}},
			},
			wantErr: true,
		},
		{
			name: "valid images mirror spec",
			Images: []ImageMirror{
				{Source: "abc", Destination: "cde", Match: Match{Tags: []string{"latest"}}},
				{Source: "efg", Destination: "ihj", Match: Match{Tags: []string{"latest"}}},
			},
			wantErr: false,
		},
		{
			name: "valid insecure destination",
			Images: []ImageMirror{
				{Source: "abc", Destination: "http://cde", Match: Match{Tags: []string{"latest"}}},
			},
			wantErr: false,
		},
		{
			name: "image source contains tag",
			Images: []ImageMirror{
				{Source: "abc:v1.0.0", Destination: "cde", Match: Match{Tags: []string{"latest"}}},
			},
			wantErr: true,
		},
		{
			name: "image destination contains tag",
			Images: []ImageMirror{
				{Source: "abc", Destination: "cde:v1.0.0", Match: Match{Tags: []string{"latest"}}},
			},
			wantErr: true,
		},
		{
			name: "no match criteria",
			Images: []ImageMirror{
				{Source: "abc", Destination: "cde:v1.0.0"},
			},
			wantErr: true,
		},
		{
			name: "invalid purge and alltags set",
			Images: []ImageMirror{
				{
					Source:      "abc",
					Destination: "abc",
					Match: Match{
						AllTags: true,
					},
					Purge: &Purge{
						NoMatch: true,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Config{
				Images:     tt.Images,
				Registries: tt.Registries,
			}
			if err := c.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Config.Destination() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
