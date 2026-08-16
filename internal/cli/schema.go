package cli

import (
	"context"
	"encoding/json"
	"fmt"
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
				return err
			}
			document, err := loadQuerySchema(cmd)
			if err != nil {
				return err
			}
			entries := filterSchemaEntries(schemas.Entries(document), kind)
			return writeIndentedJSON(cmd, entries, "schema declarations")
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
			document, err := loadQuerySchema(cmd)
			if err != nil {
				return err
			}
			declaration := schemas.Find(document, kind, skelName)
			if declaration == nil {
				return fmt.Errorf("schema declaration not found: %s %s", kind, skelName)
			}
			return writeIndentedJSON(cmd, declaration, "schema declaration")
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
				return fmt.Errorf("unexpected args for %s %s", commandSchema, commandSchemaSnapshot)
			}
			document, err := loadSourceSchema(flagSchemaSkelIn, cmd.String(flagSchemaSkelIn))
			if err != nil {
				return err
			}
			return schemas.Encode(cmd.Root().Writer, document)
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
				return fmt.Errorf("unexpected args for %s %s", commandSchema, commandSchemaDiff)
			}
			candidateSkelIn := cmd.String(flagSchemaSkelIn)
			candidate, err := loadSourceSchema(flagSchemaSkelIn, candidateSkelIn)
			if err != nil {
				return err
			}
			baselineSkelIn := strings.TrimSpace(cmd.String(flagSchemaBaselineSkelIn))
			var gitBaseline *_GitBaseline
			if baselineSkelIn == "" {
				gitBaseline, err = prepareGitBaseline(ctx, candidateSkelIn)
				if err != nil {
					return err
				}
				defer gitBaseline.cleanup()
				baselineSkelIn = gitBaseline.skelIn
			}
			baseline, err := loadSourceSchema(flagSchemaBaselineSkelIn, baselineSkelIn)
			if err != nil {
				return err
			}
			report, err := schemas.Diff(baseline, candidate)
			if err != nil {
				return err
			}
			if gitBaseline != nil {
				gitBaseline.remapReportPositions(report)
			}
			return writeIndentedJSON(cmd, report, "schema diff")
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
		if entry.Kind == kind {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func loadQuerySchema(cmd *ucli.Command) (*schemas.Document, error) {
	option := compiler.Option{SkelIn: cmd.String(flagSchemaSkelIn)}
	if err := normalizeCompilerOption(&option); err != nil {
		return nil, err
	}
	result, err := compiler.CompileImport(option.SkelIn)
	if err != nil {
		return nil, err
	}
	return schemas.Project(result.Domain, result.ImportAliases)
}

func loadSourceSchema(flagName, skelIn string) (*schemas.Document, error) {
	if strings.TrimSpace(skelIn) == "" {
		return nil, fmt.Errorf("missing flag %s", flagName)
	}
	option := compiler.Option{SkelIn: skelIn}
	if err := normalizeCompilerOption(&option); err != nil {
		return nil, err
	}
	shallow, err := compiler.CompileImport(option.SkelIn)
	if err != nil {
		return nil, err
	}
	document, err := schemas.Project(shallow.Domain, shallow.ImportAliases)
	if err != nil {
		return nil, err
	}
	return document, nil
}

func writeIndentedJSON(cmd *ucli.Command, value any, context string) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", context, err)
	}
	_, _ = fmt.Fprintf(cmd.Root().Writer, "%s\n", content)
	return nil
}
