package platform

import "testing"

func TestSupportsCoreHost(t *testing.T) {
	tests := []struct {
		name   string
		goos   string
		goarch string
		want   bool
	}{
		{name: "linux amd64", goos: "linux", goarch: "amd64", want: true},
		{name: "linux arm64", goos: "linux", goarch: "arm64", want: false},
		{name: "darwin amd64", goos: "darwin", goarch: "amd64", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := SupportsCoreHost(tc.goos, tc.goarch); got != tc.want {
				t.Fatalf("SupportsCoreHost(%q, %q) = %v, want %v", tc.goos, tc.goarch, got, tc.want)
			}
		})
	}
}
