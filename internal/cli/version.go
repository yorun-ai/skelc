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
			if cmd.Args().Len() != 0 {
				return fmt.Errorf("unexpected args for %s", commandVersion)
			}
			info, err := versionInfo()
			if err != nil {
				return err
			}
			format, err := commandOutputFormat(cmd)
			if err != nil {
				return err
			}
			if format == outputFormatJSON {
				json, err := info.JSONString()
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(cmd.Root().Writer, json)
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

func versionInfo() (_VersionInfo, error) {
	buildInfo, err := debugBuildInfo()
	if err != nil {
		return _VersionInfo{}, err
	}
	return _VersionInfo{
		Name:      cliName,
		Version:   buildInfo.Version,
		Platform:  buildInfo.Platform,
		GoVersion: buildInfo.GoVersion,
		GolangCodeGen: _VersionGolangCodeGen{
			DefaultVineVersion: golang.DefaultVineVersion,
		},
	}, nil
}

func debugBuildInfo() (_DebugBuildInfo, error) {
	info, ok := readBuildInfo()
	if !ok {
		return _DebugBuildInfo{}, fmt.Errorf("read Go build info failed")
	}
	version, err := moduleVersion(info.Main.Version)
	if err != nil {
		return _DebugBuildInfo{}, err
	}
	return _DebugBuildInfo{
		Version:   version,
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
		GoVersion: info.GoVersion,
	}, nil
}

func moduleVersion(rawVersion string) (string, error) {
	if rawVersion == "" || rawVersion == "(devel)" {
		return devVersion, nil
	}

	version := strings.TrimSuffix(rawVersion, "+dirty")
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	_, err := semver.NewVersion(version)
	if err != nil {
		return "", fmt.Errorf("parse module version %s failed: %w", version, err)
	}
	return version, nil
}

func (info _VersionInfo) TextString() string {
	return info.Name + "\n" +
		"  Version    " + info.Version + "\n" +
		"  Platform   " + info.Platform + "\n" +
		"  GoVersion  " + info.GoVersion + "\n" +
		"Golang CodeGen:\n" +
		"  DefaultVineVersion  " + info.GolangCodeGen.DefaultVineVersion
}

func (info _VersionInfo) JSONString() (string, error) {
	encoded, err := json.Marshal(info)
	if err != nil {
		return "", fmt.Errorf("marshal version info: %w", err)
	}
	return string(encoded), nil
}
