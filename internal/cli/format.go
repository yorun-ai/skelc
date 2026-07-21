package cli

import (
	"bytes"
	"context"
	"os"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/skelc/internal/formatter"
	"go.yorun.ai/skelc/internal/loader"
	"go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/internal/util/checkutil"
)

const (
	commandFormat = "format"

	flagFormatSkelIn = "skel-in"
)

type _FormattedFile struct {
	path    string
	content []byte
}

func newFormatCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandFormat,
		Usage: "format skel definition files",
		Flags: newFormatFlags(),
		Action: func(_ context.Context, cmd *ucli.Command) error {
			formatFiles(parseFormatCommand(cmd))
			return nil
		},
	}
}

func newFormatFlags() []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{Name: flagFormatSkelIn, Usage: "skeleton input file or directory"},
	}
}

func parseFormatCommand(cmd *ucli.Command) string {
	checkutil.Check(cmd.Args().Len() == 0, "unexpected args for %s", commandFormat)
	skelIn := cmd.String(flagFormatSkelIn)
	checkutil.Check(skelIn != "", "missing flag skel-in")
	return normalizeRequiredPath(skelIn)
}

func formatFiles(skelIn string) {
	sourceFiles := loader.Load(skelIn).Files
	formattedFiles := make([]_FormattedFile, 0, len(sourceFiles))
	for _, sourceFile := range sourceFiles {
		parser.ValidateSource(sourceFile.FilePath, sourceFile.Content)
		formatted := formatter.Source(sourceFile.Content)
		parser.ValidateSource(sourceFile.FilePath, formatted)
		if bytes.Equal(sourceFile.Content, formatted) {
			continue
		}
		formattedFiles = append(formattedFiles, _FormattedFile{path: sourceFile.FilePath, content: formatted})
	}
	for _, file := range formattedFiles {
		err := os.WriteFile(file.path, file.content, 0o644)
		checkutil.CheckNilError(err, "write %s failed", file.path)
	}
}
