package cli

import (
	"bytes"
	"context"
	"encoding/json"
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
	flagFormatCheck  = "check"
)

type _FormatOption struct {
	skelIn       string
	check        bool
	outputFormat string
}

type _FormatResult struct {
	Changed bool     `json:"changed"`
	Files   []string `json:"files"`
}

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
			&ucli.BoolFlag{Name: flagFormatCheck, Usage: "check formatting without modifying files"},
			newOutputFormatFlag("format result output format: text/json"),
		},
		Action: func(_ context.Context, cmd *ucli.Command) error {
			option, err := parseFormatCommand(cmd)
			if err != nil {
				return err
			}
			result, err := formatFiles(option)
			if err != nil {
				return err
			}
			if err := writeFormatResult(cmd, option, result); err != nil {
				return err
			}
			if option.check && result.Changed {
				return fmt.Errorf("%d Skel file(s) require formatting", len(result.Files))
			}
			return nil
		},
	}
}

func parseFormatCommand(cmd *ucli.Command) (_FormatOption, error) {
	if cmd.Args().Len() != 0 {
		return _FormatOption{}, fmt.Errorf("unexpected args for %s", commandFormat)
	}
	skelIn := cmd.String(flagFormatSkelIn)
	if skelIn == "" {
		return _FormatOption{}, fmt.Errorf("missing flag skel-in")
	}
	path, err := filepath.Abs(skelIn)
	if err != nil {
		return _FormatOption{}, fmt.Errorf("resolve path %s: %w", skelIn, err)
	}
	outputFormat, err := commandOutputFormat(cmd)
	if err != nil {
		return _FormatOption{}, err
	}
	return _FormatOption{skelIn: path, check: cmd.Bool(flagFormatCheck), outputFormat: outputFormat}, nil
}

func formatFiles(option _FormatOption) (_FormatResult, error) {
	loadResult, err := loader.Load(option.skelIn)
	if err != nil {
		return _FormatResult{}, err
	}
	sourceFiles := loadResult.Files
	formattedFiles := make([]_FormattedFile, 0, len(sourceFiles))
	for _, sourceFile := range sourceFiles {
		if err := parser.ValidateSource(sourceFile.FilePath, sourceFile.Content); err != nil {
			return _FormatResult{}, err
		}
		formatted, err := formatter.Source(sourceFile.Content)
		if err != nil {
			return _FormatResult{}, fmt.Errorf("format %s: %w", sourceFile.FilePath, err)
		}
		if err := parser.ValidateSource(sourceFile.FilePath, formatted); err != nil {
			return _FormatResult{}, err
		}
		if bytes.Equal(sourceFile.Content, formatted) {
			continue
		}
		formattedFiles = append(formattedFiles, _FormattedFile{
			path: sourceFile.FilePath, original: sourceFile.Content, content: formatted,
		})
	}
	result := _FormatResult{Changed: len(formattedFiles) > 0, Files: make([]string, 0, len(formattedFiles))}
	for _, file := range formattedFiles {
		result.Files = append(result.Files, file.path)
	}
	if option.check {
		return result, nil
	}
	replacements := make([]fileutil.Replacement, 0, len(formattedFiles))
	for _, file := range formattedFiles {
		writePath, err := filepath.EvalSymlinks(file.path)
		if err != nil {
			return _FormatResult{}, fmt.Errorf("resolve format target %s: %w", file.path, err)
		}
		info, err := os.Stat(writePath)
		if err != nil {
			return _FormatResult{}, fmt.Errorf("inspect format target %s: %w", writePath, err)
		}
		current, err := os.ReadFile(writePath)
		if err != nil {
			return _FormatResult{}, fmt.Errorf("read format target %s: %w", writePath, err)
		}
		if !bytes.Equal(current, file.original) {
			return _FormatResult{}, fmt.Errorf("format target %s changed while formatting", writePath)
		}
		replacements = append(replacements, fileutil.Replacement{
			Path: writePath, Content: file.content, Mode: info.Mode(),
		})
	}
	if err := fileutil.ReplaceAll(replacements); err != nil {
		return _FormatResult{}, err
	}
	return result, nil
}

func writeFormatResult(cmd *ucli.Command, option _FormatOption, result _FormatResult) error {
	if option.outputFormat == outputFormatJSON {
		content, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal format result: %w", err)
		}
		_, _ = fmt.Fprintf(cmd.Root().Writer, "%s\n", content)
		return nil
	}
	if !option.check {
		return nil
	}
	for _, path := range result.Files {
		_, _ = fmt.Fprintln(cmd.Root().Writer, path)
	}
	return nil
}
