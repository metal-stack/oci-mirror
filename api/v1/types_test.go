package v1

import (
	"testing"

	"github.com/metal-stack/metal-lib/pkg/pointer"
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
				{Source: "abc", Destination: "cde"},
				{Source: "abc", Destination: "efg"},
			},
			wantErr: true,
		},
		{
			name: "duplicate destination",
			Images: []ImageMirror{
				{Source: "cde", Destination: "abc"},
				{Source: "efg", Destination: "abc"},
			},
			wantErr: true,
		},
		{
			name: "source and destination are equal",
			Images: []ImageMirror{
				{Source: "abc", Destination: "abc"},
			},
			wantErr: true,
		},
		{
			name: "source empty",
			Images: []ImageMirror{
				{Source: "", Destination: "abc"},
			},
			wantErr: true,
		},
		{
			name: "destination empty",
			Images: []ImageMirror{
				{Source: "abc", Destination: ""},
			},
			wantErr: true,
		},
		{
			name: "invalid semver",
			Images: []ImageMirror{
				{
					Source:      "abc",
					Destination: "abc",
					Match: Match{
						Semver: pointer.Pointer("abc"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "image cde is used in two images",
			Images: []ImageMirror{
				{Source: "abc", Destination: "cde"},
				{Source: "cde", Destination: "efg"},
			},
			wantErr: true,
		},
		{
			name: "valid images mirror spec",
			Images: []ImageMirror{
				{Source: "abc", Destination: "cde"},
				{Source: "efg", Destination: "ihj"},
			},
			wantErr: false,
		},
		{
			name: "valid insecure destination",
			Images: []ImageMirror{
				{Source: "abc", Destination: "http://cde"},
			},
			wantErr: false,
		},
		{
			name: "image source contains tag",
			Images: []ImageMirror{
				{Source: "abc:v1.0.0", Destination: "cde"},
			},
			wantErr: true,
		},
		{
			name: "image destination contains tag",
			Images: []ImageMirror{
				{Source: "abc", Destination: "cde:v1.0.0"},
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
