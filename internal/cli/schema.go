package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/skelc/internal/compiler"
	schemas "go.yorun.ai/skelc/internal/schema"
)

const (
	commandSchema         = "schema"
	commandSchemaList     = "list"
	commandSchemaGet      = "get"
	commandSchemaSnapshot = "snapshot"
	commandSchemaDiff     = "diff"

	flagSchemaSkelIn         = "skel-in"
	flagSchemaBaselineSkelIn = "baseline-skel-in"
)

func newSchemaCommand() *ucli.Command {
	return &ucli.Command{
		Name:               commandSchema,
		Usage:              "inspect, snapshot, and diff skel schemas",
		HideHelpCommand:    true,
		CustomHelpTemplate: groupCommandHelpTemplate,
		Commands: []*ucli.Command{
			newSchemaListCommand(),
			newSchemaGetCommand(),
			newSchemaSnapshotCommand(),
			newSchemaDiffCommand(),
		},
	}
}

func newSchemaListCommand() *ucli.Command {
	return &ucli.Command{
		Name:      commandSchemaList,
		Usage:     "list top-level schema declarations",
		ArgsUsage: "[TYPE]",
		Flags:     newSchemaListFlags(),
		Action: func(_ context.Context, cmd *ucli.Command) error {
			kind, err := parseSchemaListKind(cmd)
			if err != nil {
				return schemaCommandFailure(schemas.ErrorCodeInvalidArgument, err)
			}
			document, err := loadQuerySchema(cmd)
			if err != nil {
				return err
			}
			entries := filterSchemaEntries(schemas.Entries(document), kind)
			return writeSchemaResult(cmd, entries, "schema declarations")
		},
	}
}

func newSchemaGetCommand() *ucli.Command {
	return &ucli.Command{
		Name:      commandSchemaGet,
		Usage:     "get one complete top-level schema declaration",
		ArgsUsage: "TYPE SKEL_NAME",
		Flags:     newSchemaGetFlags(),
		Action: func(_ context.Context, cmd *ucli.Command) error {
			kind, skelName, err := parseSchemaGetArguments(cmd)
			if err != nil {
				return schemaCommandFailure(schemas.ErrorCodeInvalidArgument, err)
			}
			document, err := loadQuerySchema(cmd)
			if err != nil {
				return err
			}
			declaration := schemas.Find(document, kind, skelName)
			return writeSchemaResult(cmd, declaration, "schema declaration")
		},
	}
}

func newSchemaSnapshotCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandSchemaSnapshot,
		Usage: "output a normalized schema snapshot",
		Flags: []ucli.Flag{
			&ucli.StringFlag{Name: flagSchemaSkelIn, Usage: "skeleton input file or directory"},
		},
		Action: func(_ context.Context, cmd *ucli.Command) error {
			if cmd.Args().Len() != 0 {
				return schemaCommandFailure(schemas.ErrorCodeInvalidArgument,
					fmt.Errorf("unexpected args for %s %s", commandSchema, commandSchemaSnapshot))
			}
			document, err := loadSourceSchema(flagSchemaSkelIn, cmd.String(flagSchemaSkelIn))
			if err != nil {
				return err
			}
			if err := schemas.Validate(document); err != nil {
				return schemaCommandFailure(schemas.ErrorCodeCommandFailed, err)
			}
			return writeSchemaResult(cmd, document, "schema snapshot")
		},
	}
}

func newSchemaDiffCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandSchemaDiff,
		Usage: "list all schema changes between baseline and candidate source",
		Flags: []ucli.Flag{
			&ucli.StringFlag{Name: flagSchemaSkelIn, Usage: "candidate skeleton input file or directory"},
			&ucli.StringFlag{Name: flagSchemaBaselineSkelIn, Usage: "baseline skeleton input file or directory; defaults to the candidate path at Git HEAD"},
		},
		Action: func(ctx context.Context, cmd *ucli.Command) error {
			if cmd.Args().Len() != 0 {
				return schemaCommandFailure(schemas.ErrorCodeInvalidArgument,
					fmt.Errorf("unexpected args for %s %s", commandSchema, commandSchemaDiff))
			}
			candidateOption := compiler.Option{SkelIn: cmd.String(flagSchemaSkelIn)}
			if err := normalizeCompilerOption(&candidateOption); err != nil {
				return schemaCommandFailure(schemas.ErrorCodeInvalidArgument, err)
			}
			baselineSkelIn := strings.TrimSpace(cmd.String(flagSchemaBaselineSkelIn))
			if baselineSkelIn != "" {
				baselineOption := compiler.Option{SkelIn: baselineSkelIn}
				if err := normalizeCompilerOption(&baselineOption); err != nil {
					return schemaCommandFailure(schemas.ErrorCodeInvalidArgument, err)
				}
				baselineSkelIn = baselineOption.SkelIn
			}
			report, err := schemas.DiffSource(ctx, candidateOption.SkelIn, schemas.SourceDiffOption{BaselineSkelIn: baselineSkelIn})
			if err != nil {
				if errors.Is(err, schemas.ErrGitHistoryUnavailable) {
					return schemaCommandFailure(schemas.ErrorCodeGitHistoryNotFound,
						fmt.Errorf("%w; pass an explicit --%s", err, flagSchemaBaselineSkelIn))
				}
				return schemaCommandFailure(schemas.ErrorCodeCommandFailed, err)
			}
			return writeSchemaResult(cmd, report, "schema diff")
		},
	}
}

func newSchemaListFlags() []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{Name: flagSchemaSkelIn, Usage: "skeleton input file or directory"},
	}
}

func newSchemaGetFlags() []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{Name: flagSchemaSkelIn, Usage: "skeleton input file or directory"},
	}
}

func parseSchemaListKind(cmd *ucli.Command) (string, error) {
	if cmd.Args().Len() > 1 {
		return "", fmt.Errorf("unexpected args for %s %s", commandSchema, commandSchemaList)
	}
	if cmd.Args().Len() == 0 {
		return "", nil
	}
	kind := strings.TrimSpace(cmd.Args().First())
	if err := schemas.ValidateKind(kind); err != nil {
		return "", err
	}
	return kind, nil
}

func parseSchemaGetArguments(cmd *ucli.Command) (string, string, error) {
	if cmd.Args().Len() > 2 {
		return "", "", fmt.Errorf("unexpected args for %s %s", commandSchema, commandSchemaGet)
	}
	if cmd.Args().Len() < 2 {
		return "", "", fmt.Errorf("missing schema declaration type or skel name; expected TYPE SKEL_NAME")
	}
	kind := strings.TrimSpace(cmd.Args().Get(0))
	if err := schemas.ValidateKind(kind); err != nil {
		return "", "", err
	}
	skelName := strings.TrimSpace(cmd.Args().Get(1))
	if skelName == "" {
		return "", "", fmt.Errorf("missing skel name")
	}
	return kind, skelName, nil
}

func filterSchemaEntries(entries []*schemas.Entry, kind string) []*schemas.Entry {
	if kind == "" {
		return entries
	}
	filtered := make([]*schemas.Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.Kind == schemas.DeclarationType(kind) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func loadQuerySchema(cmd *ucli.Command) (*schemas.Document, error) {
	option := compiler.Option{SkelIn: cmd.String(flagSchemaSkelIn)}
	if err := normalizeCompilerOption(&option); err != nil {
		return nil, schemaCommandFailure(schemas.ErrorCodeInvalidArgument, err)
	}
	result, err := compiler.CompileImport(option.SkelIn)
	if err != nil {
		return nil, schemaCommandFailure(schemas.ErrorCodeCompilationFailed, err)
	}
	document, err := schemas.Project(result.Domain, result.ImportAliases)
	if err != nil {
		return nil, schemaCommandFailure(schemas.ErrorCodeCommandFailed, err)
	}
	return document, nil
}

func loadSourceSchema(flagName, skelIn string) (*schemas.Document, error) {
	if strings.TrimSpace(skelIn) == "" {
		return nil, schemaCommandFailure(schemas.ErrorCodeInvalidArgument, fmt.Errorf("missing flag %s", flagName))
	}
	option := compiler.Option{SkelIn: skelIn}
	if err := normalizeCompilerOption(&option); err != nil {
		return nil, schemaCommandFailure(schemas.ErrorCodeInvalidArgument, err)
	}
	shallow, err := compiler.CompileImport(option.SkelIn)
	if err != nil {
		return nil, schemaCommandFailure(schemas.ErrorCodeCompilationFailed, err)
	}
	document, err := schemas.Project(shallow.Domain, shallow.ImportAliases)
	if err != nil {
		return nil, schemaCommandFailure(schemas.ErrorCodeCommandFailed, err)
	}
	return document, nil
}

func writeIndentedJSON(cmd *ucli.Command, value any, context string) error {
	if err := writeIndentedJSONTo(cmd.Root().Writer, value); err != nil {
		return fmt.Errorf("encode %s: %w", context, err)
	}
	return nil
}

func writeIndentedJSONTo(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeSchemaResult(cmd *ucli.Command, value any, context string) error {
	if err := writeIndentedJSON(cmd, value, context); err != nil {
		return schemaCommandFailure(schemas.ErrorCodeCommandFailed, err)
	}
	return nil
}

type _SchemaCommandFailure struct {
	code  schemas.ErrorCode
	cause error
}

func (e *_SchemaCommandFailure) Error() string {
	return e.cause.Error()
}

func (e *_SchemaCommandFailure) Unwrap() error {
	return e.cause
}

func (e *_SchemaCommandFailure) commandError() *schemas.CommandError {
	return new(schemas.CommandError{Code: e.code, Message: e.cause.Error()})
}

func schemaCommandFailure(code schemas.ErrorCode, err error) error {
	if failure, ok := err.(*_SchemaCommandFailure); ok {
		return failure
	}
	return new(_SchemaCommandFailure{code: code, cause: err})
}

func schemaFailureLogs(failure *_SchemaCommandFailure, format string) string {
	if diagnostics, ok := failure.cause.(interface{ DiagnosticEntries() compiler.Diagnostics }); ok {
		return formatDiagnostics(diagnostics.DiagnosticEntries(), format)
	}
	if diagnostics, ok := failure.cause.(interface{ Errors() []error }); ok {
		return formatErrors(diagnostics.Errors(), format)
	}
	return ""
}

func schemaCommandRequested(args []string) bool {
	for index := 1; index < len(args); index++ {
		arg := args[index]
		if arg == "--"+flagLogFormat {
			index++
			continue
		}
		if strings.HasPrefix(arg, "--"+flagLogFormat+"=") {
			continue
		}
		return arg == commandSchema
	}
	return false
}

func writeInvalidSchemaCommandErrorTo(writer io.Writer, message string) error {
	return writeIndentedJSONTo(writer, new(schemas.CommandError{
		Code:    schemas.ErrorCodeInvalidArgument,
		Message: message,
	}))
}
