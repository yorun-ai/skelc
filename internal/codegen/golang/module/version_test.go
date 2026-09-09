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
		{name: "explicit minimum", version: "v0.15.5", expected: "v0.15.5"},
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
	for _, version := range []string{"v0.15.4", "v0.15.3", "v0.15.5-rc.1", "v0.15.1", "v0.15.2", "v0.15.3-rc.1", "v0.15.0", "v0.14.0", "v0.14.1", "v0.15.1-rc.1", "0.15.0", "v0.13.1", "v0.13.0", "v0.12.0", "v0.10.1", "v-invalid"} {
		t.Run(version, func(t *testing.T) {
			if _, err := ResolveVineVersion(version); err == nil {
				t.Fatalf("expected %q to return an error", version)
			}
		})
	}
}

func TestApiServiceVineVersion(t *testing.T) {
	got, err := ResolveServiceVineVersion("", true)
	if err != nil || got != MinimumApiServiceVineVersion {
		t.Fatalf("default API schema version: %s %v", got, err)
	}
	if got, err := ResolveServiceVineVersion("v0.15.5", true); err != nil || got != "v0.15.5" {
		t.Fatalf("minimum API schema version: %s %v", got, err)
	}
	if _, err := ResolveServiceVineVersion("v0.15.3", true); err == nil {
		t.Fatal("accepted Vine without API schema support")
	}
	if got, err := ResolveServiceVineVersion("v1.0.0", true); err != nil || got != "v1.0.0" {
		t.Fatalf("newer API schema version: %s %v", got, err)
	}
}

func TestResolveVrpcVersion(t *testing.T) {
	for _, version := range []string{"", "v0.12.0"} {
		got, err := ResolveVrpcVersion(version)
		if err != nil || got != "v0.12.0" {
			t.Fatalf("resolve %q: %q %v", version, got, err)
		}
	}
	for _, version := range []string{"v0.11.0", "v0.12.0-rc.1"} {
		if _, err := ResolveVrpcVersion(version); err == nil {
			t.Fatalf("accepted unsupported vRPC version %q", version)
		}
	}
}
