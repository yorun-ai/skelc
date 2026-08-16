package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/skelc/internal/compiler"
	"go.yorun.ai/skelc/internal/model"
	schemas "go.yorun.ai/skelc/internal/schema"
)

const (
	commandSymbol     = "symbol"
	commandSymbolList = "list"
	commandSymbolGet  = "get"

	flagSymbolSkelIn = "skel-in"
)

type _Symbol = schemas.Entry

func newSymbolCommand() *ucli.Command {
	return &ucli.Command{
		Name:               commandSymbol,
		Usage:              "deprecated schema query compatibility commands",
		HideHelpCommand:    true,
		CustomHelpTemplate: groupCommandHelpTemplate,
		Commands: []*ucli.Command{
			newSymbolListCommand(),
			newSymbolGetCommand(),
		},
	}
}

func newSymbolListCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandSymbolList,
		Usage: "list skel symbols",
		Flags: newSymbolFlags("symbol list output format: text/json"),
		Action: func(_ context.Context, cmd *ucli.Command) error {
			option, err := parseSymbolListCommand(cmd)
			if err != nil {
				return err
			}
			result, err := compiler.CompileImport(option.SkelIn)
			if err != nil {
				return err
			}
			symbols := buildSymbols(result.Domain)
			format, err := commandOutputFormat(cmd)
			if err != nil {
				return err
			}
			if format == outputFormatJSON {
				output, err := json.MarshalIndent(symbols, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal symbols: %w", err)
				}
				_, _ = fmt.Fprintf(cmd.Root().Writer, "%s\n", output)
				return nil
			}
			writeSymbols(cmd, symbols)
			printDiagnostics(cmd, result.Diagnostics)
			return nil
		},
	}
}

func newSymbolFlags(outputFormatUsage string) []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{Name: flagSymbolSkelIn, Usage: "skeleton input file or directory"},
		newOutputFormatFlag(outputFormatUsage),
	}
}

func parseSymbolListCommand(cmd *ucli.Command) (compiler.Option, error) {
	if cmd.Args().Len() != 0 {
		return compiler.Option{}, fmt.Errorf("unexpected args for %s %s", commandSymbol, commandSymbolList)
	}
	compilerOption := compiler.Option{
		SkelIn: cmd.String(flagSymbolSkelIn),
	}
	return compilerOption, normalizeCompilerOption(&compilerOption)
}

func newSymbolGetCommand() *ucli.Command {
	return &ucli.Command{
		Name:      commandSymbolGet,
		Usage:     "get a skel symbol",
		ArgsUsage: "SKEL_NAME",
		Flags:     newSymbolFlags("symbol output format: text/json"),
		Action: func(_ context.Context, cmd *ucli.Command) error {
			skelName, err := parseSymbolGetCommand(cmd)
			if err != nil {
				return err
			}
			option, err := parseSymbolGetFlags(cmd)
			if err != nil {
				return err
			}
			result, err := compiler.CompileImport(option.SkelIn)
			if err != nil {
				return err
			}
			for _, symbol := range buildSymbols(result.Domain) {
				if symbol.SkelName != skelName {
					continue
				}
				format, err := commandOutputFormat(cmd)
				if err != nil {
					return err
				}
				if format == outputFormatJSON {
					return writeSymbolJSON(cmd, symbol)
				}
				writeSymbolText(cmd, symbol, len(symbol.Kind))
				printDiagnostics(cmd, result.Diagnostics)
				return nil
			}
			return fmt.Errorf("symbol not found: %s", skelName)
		},
	}
}

func parseSymbolGetCommand(cmd *ucli.Command) (string, error) {
	if cmd.Args().Len() > 1 {
		return "", fmt.Errorf("unexpected args for %s %s", commandSymbol, commandSymbolGet)
	}
	if cmd.Args().Len() != 1 {
		return "", fmt.Errorf("missing skel name")
	}
	skelName := strings.TrimSpace(cmd.Args().First())
	if skelName == "" {
		return "", fmt.Errorf("missing skel name")
	}
	return skelName, nil
}

func parseSymbolGetFlags(cmd *ucli.Command) (compiler.Option, error) {
	compilerOption := compiler.Option{
		SkelIn: cmd.String(flagSymbolSkelIn),
	}
	return compilerOption, normalizeCompilerOption(&compilerOption)
}

func writeSymbols(cmd *ucli.Command, symbols []*_Symbol) {
	kindWidth := maxSymbolKindWidth(symbols)
	for _, symbol := range symbols {
		writeSymbolText(cmd, symbol, kindWidth)
	}
}

func writeSymbolJSON(cmd *ucli.Command, symbol *_Symbol) error {
	output, err := json.MarshalIndent(symbol, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal symbol: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.Root().Writer, "%s\n", output)
	return nil
}

func writeSymbolText(cmd *ucli.Command, symbol *_Symbol, kindWidth int) {
	if symbol.Pub {
		_, _ = fmt.Fprintf(cmd.Root().Writer, "pub  %-*s  %s\n", kindWidth, symbol.Kind, symbol.SkelName)
		return
	}
	_, _ = fmt.Fprintf(cmd.Root().Writer, "---  %-*s  %s\n", kindWidth, symbol.Kind, symbol.SkelName)
}

func maxSymbolKindWidth(symbols []*_Symbol) int {
	maxWidth := 0
	for _, symbol := range symbols {
		if len(symbol.Kind) > maxWidth {
			maxWidth = len(symbol.Kind)
		}
	}
	return maxWidth
}

func buildSymbols(domain *model.Domain) []*_Symbol {
	document, err := schemas.Project(domain, nil)
	if err != nil {
		return nil
	}
	entries := schemas.Entries(document)
	symbols := make([]*_Symbol, 0, len(entries))
	for _, entry := range entries {
		if entry.Kind != "resource" {
			symbols = append(symbols, entry)
		}
	}
	return symbols
}
