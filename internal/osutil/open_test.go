package osutil

import (
	"reflect"
	"testing"
)

func TestOpenCommand(t *testing.T) {
	tests := []struct {
		goos, name string
	}{
		{"darwin", "open"},
		{"windows", "explorer"},
		{"linux", "xdg-open"},
		{"freebsd", "xdg-open"},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			name, args := OpenCommand(tt.goos, "/p")
			if name != tt.name || !reflect.DeepEqual(args, []string{"/p"}) {
				t.Fatalf("got %s %v", name, args)
			}
		})
	}
}
