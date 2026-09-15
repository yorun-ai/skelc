package module

import "testing"

func TestResolveVineVersion(t *testing.T) {
	for _, test := range []struct {
		name     string
		version  string
		expected string
	}{
		{name: "default", expected: DefaultVineVersion},
		{name: "trimmed default", version: "  ", expected: DefaultVineVersion},
		{name: "explicit minimum", version: "v0.19.0", expected: "v0.19.0"},
		{name: "higher", version: "v1.2.3", expected: "v1.2.3"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveVineVersion(test.version)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.expected {
				t.Fatalf("unexpected Vine version: got %q want %q", got, test.expected)
			}
		})
	}
}

func TestResolveVineVersionRejectsInvalidVersion(t *testing.T) {
	for _, version := range []string{"v0.15.6", "v0.19.0-rc.1", "0.15.7", "v-invalid", "v01.15.7", "v1.2", "v2.0.0"} {
		t.Run(version, func(t *testing.T) {
			if _, err := ResolveVineVersion(version); err == nil {
				t.Fatalf("expected %q to return an error", version)
			}
		})
	}
}

func TestResolveVrpcVersion(t *testing.T) {
	for _, version := range []string{"", "v0.12.0"} {
		got, err := ResolveVrpcVersion(version)
		if err != nil || got != "v0.12.0" {
			t.Fatalf("resolve %q: %q %v", version, got, err)
		}
	}
	for _, version := range []string{"v0.11.0", "v0.12.0-rc.1", "v01.12.0", "v1.2", "v2.0.0"} {
		if _, err := ResolveVrpcVersion(version); err == nil {
			t.Fatalf("accepted unsupported vRPC version %q", version)
		}
	}
}
