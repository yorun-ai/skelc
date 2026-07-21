package module

import (
	"strings"

	"github.com/Masterminds/semver/v3"
	"go.yorun.ai/skelc/internal/util/checkutil"
)

// DefaultVineVersion is the minimum Vine version targeted by generated Go code.
const DefaultVineVersion = "v0.9.0"

func ResolveVineVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return DefaultVineVersion
	}

	checkutil.Check(strings.HasPrefix(version, "v"), "go-vine-version %s must be v-prefixed semantic version", version)
	parsedVersion, err := semver.NewVersion(version)
	checkutil.CheckNilError(err, "parse go-vine-version %s failed", version)

	parsedDefaultVersion := semver.MustParse(DefaultVineVersion)
	checkutil.Check(parsedVersion.Compare(parsedDefaultVersion) >= 0,
		"go-vine-version %s is lower than default %s", version, DefaultVineVersion)
	return version
}
