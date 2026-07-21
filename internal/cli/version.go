package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/Masterminds/semver/v3"
	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/skelc/internal/codegen/golang"
	"go.yorun.ai/skelc/internal/util/checkutil"
)

const (
	commandVersion = "version"

	cliName    = "Skelc CLI"
	devVersion = "v0.0.0-dev"
)

var readBuildInfo = debug.ReadBuildInfo

func newVersionCommand() *ucli.Command {
	return &ucli.Command{
		Name: commandVersion,
		Flags: []ucli.Flag{
			newOutputFormatFlag("version output format: text/json"),
		},
		Action: func(_ context.Context, cmd *ucli.Command) error {
			checkutil.Check(cmd.Args().Len() == 0, "unexpected args for %s", commandVersion)
			info := versionInfo()
			if commandOutputFormat(cmd) == outputFormatJSON {
				_, _ = fmt.Fprintln(cmd.Root().Writer, info.JSONString())
				return nil
			}
			_, _ = fmt.Fprintln(cmd.Root().Writer, info.TextString())
			return nil
		},
	}
}

type _VersionInfo struct {
	Name          string                `json:"name"`
	Version       string                `json:"version"`
	Platform      string                `json:"platform"`
	GoVersion     string                `json:"goVersion"`
	GolangCodeGen _VersionGolangCodeGen `json:"golangCodeGen"`
}

type _DebugBuildInfo struct {
	Version   string `json:"version"`
	Platform  string `json:"platform"`
	GoVersion string `json:"goVersion"`
}

type _VersionGolangCodeGen struct {
	DefaultVineVersion string `json:"defaultVineVersion"`
}

func versionInfo() _VersionInfo {
	buildInfo := mustDebugBuildInfo()
	return _VersionInfo{
		Name:      cliName,
		Version:   buildInfo.Version,
		Platform:  buildInfo.Platform,
		GoVersion: buildInfo.GoVersion,
		GolangCodeGen: _VersionGolangCodeGen{
			DefaultVineVersion: golang.DefaultVineVersion,
		},
	}
}

func mustDebugBuildInfo() _DebugBuildInfo {
	info, ok := readBuildInfo()
	checkutil.Check(ok, "read Go build info failed")
	return _DebugBuildInfo{
		Version:   moduleVersion(info.Main.Version),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
		GoVersion: info.GoVersion,
	}
}

func moduleVersion(rawVersion string) string {
	if rawVersion == "" || rawVersion == "(devel)" {
		return devVersion
	}

	version := strings.TrimSuffix(rawVersion, "+dirty")
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	_, err := semver.NewVersion(version)
	checkutil.CheckNilError(err, "parse module version %s failed", version)
	return version
}

func (info _VersionInfo) TextString() string {
	return info.Name + "\n" +
		"  Version    " + info.Version + "\n" +
		"  Platform   " + info.Platform + "\n" +
		"  GoVersion  " + info.GoVersion + "\n" +
		"Golang CodeGen:\n" +
		"  DefaultVineVersion  " + info.GolangCodeGen.DefaultVineVersion
}

func (info _VersionInfo) JSONString() string {
	encoded, err := json.Marshal(info)
	checkutil.CheckNilError(err, "marshal version info")
	return string(encoded)
}
