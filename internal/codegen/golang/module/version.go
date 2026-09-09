package module

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// MinimumVineVersion is the minimum Vine version supported by generated Go code.
const MinimumVineVersion = "v0.15.5"

// DefaultVineVersion is the Vine version used when generation does not select one.
const DefaultVineVersion = "v0.15.5"

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

// MinimumApiServiceVineVersion is required by schemas for explicit API services.
const MinimumApiServiceVineVersion = "v0.15.5"

func ResolveServiceVineVersion(version string, hasApiService bool) (string, error) {
	if hasApiService && strings.TrimSpace(version) == "" {
		return MinimumApiServiceVineVersion, nil
	}
	resolved, err := ResolveVineVersion(version)
	if err != nil || !hasApiService {
		return resolved, err
	}
	if semver.MustParse(resolved).Compare(semver.MustParse(MinimumApiServiceVineVersion)) < 0 {
		return "", fmt.Errorf("api service schemas require Vine %s or later; got %s", MinimumApiServiceVineVersion, resolved)
	}
	return resolved, nil
}
