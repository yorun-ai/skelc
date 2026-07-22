package module

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// DefaultVineVersion is the minimum Vine version targeted by generated Go code.
const DefaultVineVersion = "v0.9.0"

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
	if parsed.Compare(semver.MustParse(DefaultVineVersion)) < 0 {
		return fmt.Errorf("go-vine-version %s is lower than default %s", version, DefaultVineVersion)
	}
	return nil
}
