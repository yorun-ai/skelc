package module

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"golang.org/x/mod/module"
)

const DefaultVrpcVersion = "v0.12.0"
const MinimumVrpcVersion = "v0.12.0"

func ResolveVrpcVersion(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return DefaultVrpcVersion, nil
	}
	parsed, err := semver.StrictNewVersion(strings.TrimPrefix(version, "v"))
	if err != nil || !strings.HasPrefix(version, "v") {
		return "", fmt.Errorf("go-vrpc-version must be a v-prefixed semantic version")
	}
	if parsed.Compare(semver.MustParse(MinimumVrpcVersion)) < 0 {
		return "", fmt.Errorf("go-vrpc-version must be at least %s", MinimumVrpcVersion)
	}
	if err := module.Check("go.yorun.ai/vrpc", version); err != nil {
		return "", err
	}
	return version, nil
}
