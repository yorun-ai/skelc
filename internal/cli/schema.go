package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/skelc"
	"go.yorun.ai/skelc/internal/compiler"
	schemas "go.yorun.ai/skelc/internal/schema"
	"go.yorun.ai/skelc/internal/util/fileutil"
)

const (
	commandSchema        = "schema"
	commandSchemaList    = "list"
	commandSchemaGet     = "get"
	commandSchemaExport  = "export"
	commandSchemaCompare = "compare"

	flagSchemaSkelIn            = "skel-in"
	flagSchemaSkelImport        = "skel-import"
	flagSchemaScope             = "scope"
	flagSchemaOut               = "schema-out"
	flagSchemaIn                = "schema-in"
	flagSchemaAgainst           = "against"
	flagSchemaAgainstSkelIn     = "against-skel-in"
	flagSchemaAgainstSkelImport = "against-skel-import"
	flagSchemaFailOn            = "fail-on"

	failOnBreaking  = "breaking"
	failOnDangerous = "dangerous"
	failOnAnyChange = "any-change"
	failOnNone      = "none"
)

type _SchemaSourceResult struct {
	document    *schemas.Document
	diagnostics []skelc.Diagnostic
}

type _ExitError int

func (e _ExitError) Error() string { return "" }
func (e _ExitError) ExitCode() int { return int(e) }

func newSchemaCommand() *ucli.Command {
	return &ucli.Command{
		Name:               commandSchema,
		Usage:              "inspect, export, and compare skel schemas",
		HideHelpCommand:    true,
		CustomHelpTemplate: groupCommandHelpTemplate,
		Commands: []*ucli.Command{
			newSchemaListCommand(),
			newSchemaGetCommand(),
			newSchemaExportCommand(),
			newSchemaCompareCommand(),
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
				return err
			}
			document, diagnostics, err := loadQuerySchema(cmd)
			if err != nil {
				return err
			}
			format, err := commandOutputFormat(cmd)
			if err != nil {
				return err
			}
			entries := filterSchemaEntries(schemas.Entries(document), kind)
			if format == outputFormatJSON {
				if err := writeIndentedJSON(cmd, entries, "schema declarations"); err != nil {
					return err
				}
				return nil
			}
			writeSymbols(cmd, entries)
			printDiagnostics(cmd, diagnostics)
			return nil
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
				return err
			}
			document, diagnostics, err := loadQuerySchema(cmd)
			if err != nil {
				return err
			}
			declaration := schemas.Find(document, kind, skelName)
			if declaration == nil {
				return fmt.Errorf("schema declaration not found: %s %s", kind, skelName)
			}
			format, err := commandOutputFormat(cmd)
			if err != nil {
				return err
			}
			if format == outputFormatJSON {
				return writeIndentedJSON(cmd, declaration, "schema declaration")
			}
			if err := schemas.WriteDeclarationText(cmd.Root().Writer, declaration); err != nil {
				return err
			}
			printDiagnostics(cmd, diagnostics)
			return nil
		},
	}
}

func newSchemaExportCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandSchemaExport,
		Usage: "export a normalized schema document",
		Flags: []ucli.Flag{
			&ucli.StringFlag{Name: flagSchemaSkelIn, Usage: "skeleton input file or directory"},
			&ucli.StringSliceFlag{Name: flagSchemaSkelImport, Usage: "skel dependency mapping in domain=path form; repeat for transitive imports"},
			newSchemaScopeFlag(schemas.ScopePublic),
			&ucli.StringFlag{Name: flagSchemaOut, Usage: "schema JSON output file; writes to stdout when omitted"},
		},
		Action: func(_ context.Context, cmd *ucli.Command) error {
			if cmd.Args().Len() != 0 {
				return fmt.Errorf("unexpected args for %s %s", commandSchema, commandSchemaExport)
			}
			scope, err := commandSchemaScope(cmd)
			if err != nil {
				return err
			}
			imports, err := parseMappingFlags(cmd.StringSlice(flagSchemaSkelImport), flagSchemaSkelImport)
			if err != nil {
				return err
			}
			result, err := loadSourceSchema(cmd.String(flagSchemaSkelIn), imports, scope)
			if err != nil {
				return err
			}
			var content bytes.Buffer
			if err := schemas.Encode(&content, result.document); err != nil {
				return err
			}
			outputPath := strings.TrimSpace(cmd.String(flagSchemaOut))
			if outputPath == "" {
				_, _ = cmd.Root().Writer.Write(content.Bytes())
				return nil
			}
			outputPath, err = filepath.Abs(outputPath)
			if err != nil {
				return fmt.Errorf("resolve schema output path: %w", err)
			}
			if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
				return fmt.Errorf("create schema output directory: %w", err)
			}
			if err := fileutil.Replace(fileutil.Replacement{Path: outputPath, Content: content.Bytes(), Mode: 0o644}); err != nil {
				return err
			}
			printDiagnostics(cmd, result.diagnostics)
			return nil
		},
	}
}

func newSchemaCompareCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandSchemaCompare,
		Usage: "compare a baseline and candidate schema",
		Flags: []ucli.Flag{
			&ucli.StringFlag{Name: flagSchemaSkelIn, Usage: "candidate skeleton input file or directory"},
			&ucli.StringSliceFlag{Name: flagSchemaSkelImport, Usage: "candidate skel dependency mapping in domain=path form"},
			&ucli.StringFlag{Name: flagSchemaIn, Usage: "candidate schema JSON file"},
			&ucli.StringFlag{Name: flagSchemaAgainst, Usage: "baseline schema JSON file"},
			&ucli.StringFlag{Name: flagSchemaAgainstSkelIn, Usage: "baseline skeleton input file or directory"},
			&ucli.StringSliceFlag{Name: flagSchemaAgainstSkelImport, Usage: "baseline skel dependency mapping in domain=path form"},
			newSchemaScopeFlag(schemas.ScopePublic),
			newOutputFormatFlag("schema comparison output format: text/json"),
			&ucli.StringFlag{Name: flagSchemaFailOn, Usage: "failure threshold: breaking/dangerous/any-change/none", Value: failOnBreaking},
		},
		Action: func(_ context.Context, cmd *ucli.Command) error {
			if cmd.Args().Len() != 0 {
				return fmt.Errorf("unexpected args for %s %s", commandSchema, commandSchemaCompare)
			}
			scope, err := commandSchemaScope(cmd)
			if err != nil {
				return err
			}
			format, err := commandOutputFormat(cmd)
			if err != nil {
				return err
			}
			failOn, err := normalizeFailOn(cmd.String(flagSchemaFailOn))
			if err != nil {
				return err
			}
			candidateImports, err := parseMappingFlags(cmd.StringSlice(flagSchemaSkelImport), flagSchemaSkelImport)
			if err != nil {
				return err
			}
			baselineImports, err := parseMappingFlags(cmd.StringSlice(flagSchemaAgainstSkelImport), flagSchemaAgainstSkelImport)
			if err != nil {
				return err
			}
			candidate, err := loadComparisonSchema(
				"candidate", cmd.String(flagSchemaSkelIn), cmd.String(flagSchemaIn), candidateImports, scope,
			)
			if err != nil {
				return err
			}
			baseline, err := loadComparisonSchema(
				"baseline", cmd.String(flagSchemaAgainstSkelIn), cmd.String(flagSchemaAgainst), baselineImports, scope,
			)
			if err != nil {
				return err
			}
			report, err := schemas.Compare(baseline.document, candidate.document)
			if err != nil {
				return err
			}
			if format == outputFormatJSON {
				if err := writeIndentedJSON(cmd, report, "schema comparison"); err != nil {
					return err
				}
			} else {
				writeSchemaComparison(cmd, report)
				printDiagnostics(cmd, baseline.diagnostics)
				printDiagnostics(cmd, candidate.diagnostics)
			}
			if schemaComparisonFails(report, failOn) {
				return _ExitError(ExitCodeIncompatible)
			}
			return nil
		},
	}
}

func newSchemaListFlags() []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{Name: flagSchemaSkelIn, Usage: "skeleton input file or directory"},
		newSchemaScopeFlag(schemas.ScopeAll),
		newOutputFormatFlag("schema declaration summary output format: text/json"),
	}
}

func newSchemaGetFlags() []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{Name: flagSchemaSkelIn, Usage: "skeleton input file or directory"},
		newSchemaScopeFlag(schemas.ScopeAll),
		newOutputFormatFlag("schema declaration output format: text/json"),
	}
}

func newSchemaScopeFlag(defaultScope schemas.Scope) ucli.Flag {
	return &ucli.StringFlag{Name: flagSchemaScope, Usage: "schema scope: all/public", Value: string(defaultScope)}
}

func commandSchemaScope(cmd *ucli.Command) (schemas.Scope, error) {
	scope := schemas.Scope(strings.TrimSpace(cmd.String(flagSchemaScope)))
	if err := schemas.ValidateScope(scope); err != nil {
		return "", err
	}
	return scope, nil
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
		if entry.Kind == kind {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func loadQuerySchema(cmd *ucli.Command) (*schemas.Document, compiler.Diagnostics, error) {
	option := compiler.Option{SkelIn: cmd.String(flagSchemaSkelIn)}
	if err := normalizeCompilerOption(&option); err != nil {
		return nil, nil, err
	}
	scope, err := commandSchemaScope(cmd)
	if err != nil {
		return nil, nil, err
	}
	result, err := compiler.CompileImport(option.SkelIn)
	if err != nil {
		return nil, nil, err
	}
	document, err := schemas.Project(result.Domain, scope)
	return document, result.Diagnostics, err
}

func loadSourceSchema(skelIn string, imports map[string]string, scope schemas.Scope) (_SchemaSourceResult, error) {
	if strings.TrimSpace(skelIn) == "" {
		return _SchemaSourceResult{}, fmt.Errorf("missing flag skel-in")
	}
	parsed, err := skelc.Parse(skelc.Input{SkelIn: skelIn, SkelImports: imports})
	if err != nil {
		return _SchemaSourceResult{}, err
	}
	document, err := schemas.Project(parsed.Domain, scope)
	if err != nil {
		return _SchemaSourceResult{}, err
	}
	return _SchemaSourceResult{document: document, diagnostics: parsed.Diagnostics}, nil
}

func loadComparisonSchema(label, skelIn, schemaIn string, imports map[string]string, scope schemas.Scope) (_SchemaSourceResult, error) {
	skelIn = strings.TrimSpace(skelIn)
	schemaIn = strings.TrimSpace(schemaIn)
	if (skelIn == "") == (schemaIn == "") {
		return _SchemaSourceResult{}, fmt.Errorf("%s requires exactly one of skel input or schema input", label)
	}
	if skelIn != "" {
		return loadSourceSchema(skelIn, imports, scope)
	}
	if len(imports) != 0 {
		return _SchemaSourceResult{}, fmt.Errorf("%s skel imports require skel input", label)
	}
	path, err := filepath.Abs(schemaIn)
	if err != nil {
		return _SchemaSourceResult{}, fmt.Errorf("resolve %s schema path: %w", label, err)
	}
	file, err := os.Open(path)
	if err != nil {
		return _SchemaSourceResult{}, fmt.Errorf("open %s schema %s: %w", label, path, err)
	}
	defer file.Close()
	document, err := schemas.Decode(file)
	if err != nil {
		return _SchemaSourceResult{}, fmt.Errorf("read %s schema %s: %w", label, path, err)
	}
	if document.Scope != scope {
		return _SchemaSourceResult{}, fmt.Errorf("%s schema scope is %s, expected %s", label, document.Scope, scope)
	}
	return _SchemaSourceResult{document: document}, nil
}

func writeIndentedJSON(cmd *ucli.Command, value any, context string) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", context, err)
	}
	_, _ = fmt.Fprintf(cmd.Root().Writer, "%s\n", content)
	return nil
}

func writeSchemaComparison(cmd *ucli.Command, report *schemas.Report) {
	for _, change := range report.Changes {
		_, _ = fmt.Fprintf(cmd.Root().Writer, "%-10s  %-36s  %s\n", strings.ToUpper(string(change.Impact)), change.Code, change.Symbol)
		_, _ = fmt.Fprintf(cmd.Root().Writer, "  %s\n", change.Message)
		if change.Baseline != nil {
			_, _ = fmt.Fprintf(cmd.Root().Writer, "  baseline: %s\n", change.Baseline)
		}
		if change.Candidate != nil {
			_, _ = fmt.Fprintf(cmd.Root().Writer, "  candidate: %s\n", change.Candidate)
		}
	}
	status := "compatible"
	if !report.Compatible {
		status = "incompatible"
	}
	_, _ = fmt.Fprintf(cmd.Root().Writer, "%s: %d breaking, %d dangerous, %d compatible changes\n",
		status, report.Summary.Breaking, report.Summary.Dangerous, report.Summary.Compatible)
}

func normalizeFailOn(value string) (string, error) {
	value = strings.TrimSpace(value)
	switch value {
	case failOnBreaking, failOnDangerous, failOnAnyChange, failOnNone:
		return value, nil
	default:
		return "", fmt.Errorf("invalid fail-on %q, expected breaking/dangerous/any-change/none", value)
	}
}

func schemaComparisonFails(report *schemas.Report, failOn string) bool {
	switch failOn {
	case failOnBreaking:
		return report.Summary.Breaking > 0
	case failOnDangerous:
		return report.Summary.Breaking > 0 || report.Summary.Dangerous > 0
	case failOnAnyChange:
		return len(report.Changes) > 0
	default:
		return false
	}
}
