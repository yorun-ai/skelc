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
	"go.yorun.ai/skelc/internal/util/fileutil"
)

const (
	commandFormat = "format"

	flagFormatSkelIn = "skel-in"
)

type _FormattedFile struct {
	path     string
	original []byte
	content  []byte
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
		formattedFiles = append(formattedFiles, _FormattedFile{
			path: sourceFile.FilePath, original: sourceFile.Content, content: formatted,
		})
	}
	replacements := make([]fileutil.Replacement, 0, len(formattedFiles))
	for _, file := range formattedFiles {
		writePath, err := filepath.EvalSymlinks(file.path)
		if err != nil {
			return fmt.Errorf("resolve format target %s: %w", file.path, err)
		}
		info, err := os.Stat(writePath)
		if err != nil {
			return fmt.Errorf("inspect format target %s: %w", writePath, err)
		}
		current, err := os.ReadFile(writePath)
		if err != nil {
			return fmt.Errorf("read format target %s: %w", writePath, err)
		}
		if !bytes.Equal(current, file.original) {
			return fmt.Errorf("format target %s changed while formatting", writePath)
		}
		replacements = append(replacements, fileutil.Replacement{
			Path: writePath, Content: file.content, Mode: info.Mode(),
		})
	}
	return fileutil.ReplaceAll(replacements)
}
