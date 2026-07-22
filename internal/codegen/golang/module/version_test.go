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
		{name: "explicit default", version: "v0.9.0", expected: "v0.9.0"},
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
	for _, version := range []string{"0.9.0", "v0.8.0", "v-invalid"} {
		t.Run(version, func(t *testing.T) {
			if _, err := ResolveVineVersion(version); err == nil {
				t.Fatalf("expected %q to return an error", version)
			}
		})
	}
}
