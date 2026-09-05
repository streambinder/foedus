package templates

import (
	"os"
	"testing"
)

func TestCartoAPIKey(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T)
		want  string
	}{
		{
			name: "unset",
			setup: func(t *testing.T) {
				t.Helper()
				if err := os.Unsetenv("CARTO_API_KEY"); err != nil {
					t.Fatal(err)
				}
			},
			want: "",
		},
		{
			name: "empty",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("CARTO_API_KEY", "")
			},
			want: "",
		},
		{
			name: "trims surrounding whitespace",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("CARTO_API_KEY", "  abc123  ")
			},
			want: "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)
			if got := cartoAPIKey(); got != tt.want {
				t.Fatalf("cartoAPIKey() = %q, want %q", got, tt.want)
			}
		})
	}
}
