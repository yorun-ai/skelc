package cli

import (
	"runtime"
	"runtime/debug"
	"testing"
)

func TestRunSkelcVersion(t *testing.T) {
	result := Run([]string{"version"})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	info, err := versionInfo()
	if err != nil {
		t.Fatal(err)
	}
	expected := info.TextString() + "\n"
	if result.Stdout != expected {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
	if result.Stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.Stderr)
	}
}

func TestRunSkelcVersionJSON(t *testing.T) {
	result := Run([]string{"version", "--output-format", "json"})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	info, err := versionInfo()
	if err != nil {
		t.Fatal(err)
	}
	json, err := info.JSONString()
	if err != nil {
		t.Fatal(err)
	}
	expected := json + "\n"
	if result.Stdout != expected {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
	if result.Stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.Stderr)
	}
}

func TestRunSkelcVersionRejectsLegacyJSONFlag(t *testing.T) {
	result := Run([]string{"version", "--json"})

	if result.ExitCode != ExitCodeError {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	if result.Stderr == "" {
		t.Fatal("expected stderr")
	}
}

func TestDefaultGoVineVersion(t *testing.T) {
	info, err := versionInfo()
	if err != nil {
		t.Fatal(err)
	}
	version := info.GolangCodeGen.DefaultVineVersion
	if version != "v0.9.3" {
		t.Fatalf("unexpected default go vine version: %q", version)
	}
}

func TestModuleVersion(t *testing.T) {
	for _, test := range []struct {
		name     string
		raw      string
		expected string
	}{
		{name: "empty", expected: devVersion},
		{name: "devel", raw: "(devel)", expected: devVersion},
		{name: "without v prefix", raw: "1.2.3", expected: "v1.2.3"},
		{name: "module version", raw: "v2.3.4", expected: "v2.3.4"},
		{name: "dirty", raw: "v1.1.0-alpha3+dirty", expected: "v1.1.0-alpha3"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := moduleVersion(test.raw)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.expected {
				t.Fatalf("unexpected module version: got %q want %q", got, test.expected)
			}
		})
	}
}

func TestDebugBuildInfo(t *testing.T) {
	setReadBuildInfoForTest(t, func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			GoVersion: "go1.26.0",
			Main:      debug.Module{Version: "v1.1.3"},
		}, true
	})

	info, err := debugBuildInfo()
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != "v1.1.3" {
		t.Fatalf("unexpected version: %q", info.Version)
	}
	if info.Platform != runtime.GOOS+"/"+runtime.GOARCH {
		t.Fatalf("unexpected platform: %q", info.Platform)
	}
	if info.GoVersion != "go1.26.0" {
		t.Fatalf("unexpected go version: %q", info.GoVersion)
	}
}

func TestDebugBuildInfoRejectsMissingBuildInfo(t *testing.T) {
	setReadBuildInfoForTest(t, func() (*debug.BuildInfo, bool) {
		return nil, false
	})

	_, err := debugBuildInfo()
	if err == nil {
		t.Fatal("expected error")
	}
}

func setReadBuildInfoForTest(t *testing.T, read func() (*debug.BuildInfo, bool)) {
	t.Helper()
	original := readBuildInfo
	t.Cleanup(func() {
		readBuildInfo = original
	})
	readBuildInfo = read
}
