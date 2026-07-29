package module

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// MinimumVineVersion is the minimum Vine version supported by generated Go code.
const MinimumVineVersion = "v0.10.1"

// DefaultVineVersion is the Vine version used when generation does not select one.
const DefaultVineVersion = MinimumVineVersion

func ResolveVineVersion(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return DefaultVineVersion, nil
	}
	if err := ValidateVineVersion(version); err != nil {
		return "", err
	}
	return version, nil
}

func ValidateVineVersion(version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return nil
	}
	if !strings.HasPrefix(version, "v") {
		return fmt.Errorf("go-vine-version %s must be v-prefixed semantic version", version)
	}
	parsed, err := semver.NewVersion(version)
	if err != nil {
		return fmt.Errorf("parse go-vine-version %s failed: %w", version, err)
	}
	if parsed.Compare(semver.MustParse(MinimumVineVersion)) < 0 {
		return fmt.Errorf("go-vine-version %s is lower than minimum %s", version, MinimumVineVersion)
	}
	return nil
}
