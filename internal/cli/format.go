package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/skelc/internal/formatter"
	"go.yorun.ai/skelc/internal/loader"
	"go.yorun.ai/skelc/internal/parser"
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
		Flags: []ucli.Flag{
			&ucli.StringFlag{Name: flagFormatSkelIn, Usage: "skeleton input file or directory"},
		},
		Action: func(_ context.Context, cmd *ucli.Command) error {
			path, err := parseFormatCommand(cmd)
			if err != nil {
				return err
			}
			return formatFiles(path)
		},
	}
}

func parseFormatCommand(cmd *ucli.Command) (string, error) {
	if cmd.Args().Len() != 0 {
		return "", fmt.Errorf("unexpected args for %s", commandFormat)
	}
	skelIn := cmd.String(flagFormatSkelIn)
	if skelIn == "" {
		return "", fmt.Errorf("missing flag skel-in")
	}
	path, err := filepath.Abs(skelIn)
	if err != nil {
		return "", fmt.Errorf("resolve path %s: %w", skelIn, err)
	}
	return path, nil
}

func formatFiles(skelIn string) error {
	loadResult, err := loader.Load(skelIn)
	if err != nil {
		return err
	}
	sourceFiles := loadResult.Files
	formattedFiles := make([]_FormattedFile, 0, len(sourceFiles))
	for _, sourceFile := range sourceFiles {
		if err := parser.ValidateSource(sourceFile.FilePath, sourceFile.Content); err != nil {
			return err
		}
		formatted := formatter.Source(sourceFile.Content)
		if err := parser.ValidateSource(sourceFile.FilePath, formatted); err != nil {
			return err
		}
		if bytes.Equal(sourceFile.Content, formatted) {
			continue
		}
		formattedFiles = append(formattedFiles, _FormattedFile{path: sourceFile.FilePath, content: formatted})
	}
	for _, file := range formattedFiles {
		if err := os.WriteFile(file.path, file.content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", file.path, err)
		}
	}
	return nil
}
