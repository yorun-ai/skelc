package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

const (
	commandSymbol     = "symbol"
	commandSymbolList = "list"
	commandSymbolGet  = "get"

	flagSymbolSkelIn = "skel-in"
)

type _Symbol struct {
	Pub      bool   `json:"pub"`
	Name     string `json:"name"`
	Kind     string `json:"type"`
	SkelName string `json:"skelName"`
}

func newSymbolCommand() *ucli.Command {
	return &ucli.Command{
		Name:               commandSymbol,
		Usage:              "inspect skel symbols",
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
			result := parser.ParseImport(parseSymbolListCommand(cmd).SkelIn)
			symbols := buildSymbols(result.Domain)
			if commandOutputFormat(cmd) == outputFormatJSON {
				output, err := json.MarshalIndent(symbols, "", "  ")
				checkutil.CheckNilError(err, "marshal symbols")
				_, _ = fmt.Fprintf(cmd.Root().Writer, "%s\n", output)
				return nil
			}
			writeSymbols(cmd, symbols)
			printWarnings(cmd, result.Warnings)
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

func parseSymbolListCommand(cmd *ucli.Command) parser.Option {
	checkutil.Check(cmd.Args().Len() == 0, "unexpected args for %s %s", commandSymbol, commandSymbolList)
	parserOption := parser.Option{
		SkelIn: cmd.String(flagSymbolSkelIn),
	}
	normalizeSymbolOption(&parserOption)
	return parserOption
}

func newSymbolGetCommand() *ucli.Command {
	return &ucli.Command{
		Name:      commandSymbolGet,
		Usage:     "get a skel symbol",
		ArgsUsage: "SKEL_NAME",
		Flags:     newSymbolFlags("symbol output format: text/json"),
		Action: func(_ context.Context, cmd *ucli.Command) error {
			skelName := parseSymbolGetCommand(cmd)
			result := parser.ParseImport(parseSymbolGetFlags(cmd).SkelIn)
			for _, symbol := range buildSymbols(result.Domain) {
				if symbol.SkelName != skelName {
					continue
				}
				if commandOutputFormat(cmd) == outputFormatJSON {
					writeSymbolJSON(cmd, symbol)
					return nil
				}
				writeSymbolText(cmd, symbol, len(symbol.Kind))
				printWarnings(cmd, result.Warnings)
				return nil
			}
			return fmt.Errorf("symbol not found: %s", skelName)
		},
	}
}

func parseSymbolGetCommand(cmd *ucli.Command) string {
	checkutil.Check(cmd.Args().Len() <= 1, "unexpected args for %s %s", commandSymbol, commandSymbolGet)
	checkutil.Check(cmd.Args().Len() == 1, "missing skel name")
	skelName := strings.TrimSpace(cmd.Args().First())
	checkutil.Check(skelName != "", "missing skel name")
	return skelName
}

func parseSymbolGetFlags(cmd *ucli.Command) parser.Option {
	parserOption := parser.Option{
		SkelIn: cmd.String(flagSymbolSkelIn),
	}
	normalizeSymbolOption(&parserOption)
	return parserOption
}

func writeSymbols(cmd *ucli.Command, symbols []*_Symbol) {
	kindWidth := maxSymbolKindWidth(symbols)
	for _, symbol := range symbols {
		writeSymbolText(cmd, symbol, kindWidth)
	}
}

func writeSymbolJSON(cmd *ucli.Command, symbol *_Symbol) {
	output, err := json.MarshalIndent(symbol, "", "  ")
	checkutil.CheckNilError(err, "marshal symbol")
	_, _ = fmt.Fprintf(cmd.Root().Writer, "%s\n", output)
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
	symbols := make([]*_Symbol, 0)
	for _, enum := range domain.Enums() {
		symbols = append(symbols, &_Symbol{Pub: enum.Pub, Name: enum.Name, Kind: "enum", SkelName: enum.SkelName})
	}
	for _, data := range domain.Data() {
		symbols = append(symbols, &_Symbol{Pub: data.Pub, Name: data.Name, Kind: "data", SkelName: data.SkelName})
	}
	for _, config := range domain.Configs() {
		symbols = append(symbols, &_Symbol{Pub: config.Pub, Name: config.Name, Kind: "config", SkelName: config.SkelName})
	}
	for _, event := range domain.Events() {
		symbols = append(symbols, &_Symbol{Pub: event.Pub, Name: event.Name, Kind: "event", SkelName: event.SkelName})
	}
	for _, actor := range domain.Actors() {
		symbols = append(symbols, &_Symbol{Pub: actor.Pub, Name: actor.Name, Kind: "actor", SkelName: actor.SkelName})
	}
	for _, service := range domain.Services() {
		symbols = append(symbols, &_Symbol{Pub: service.Pub, Name: service.Name, Kind: "service", SkelName: service.SkelName})
	}
	for _, web := range domain.Webs() {
		symbols = append(symbols, &_Symbol{Name: web.Name, Kind: "web", SkelName: web.SkelName})
	}
	for _, task := range domain.Tasks() {
		symbols = append(symbols, &_Symbol{Name: task.Name, Kind: "task", SkelName: task.SkelName})
	}
	sort.Slice(symbols, func(i int, j int) bool {
		iKindOrder := symbolKindOrder(symbols[i].Kind)
		jKindOrder := symbolKindOrder(symbols[j].Kind)
		if iKindOrder == jKindOrder {
			return symbols[i].SkelName < symbols[j].SkelName
		}
		return iKindOrder < jKindOrder
	})
	return symbols
}

func symbolKindOrder(kind string) int {
	switch kind {
	case "actor":
		return 1
	case "config":
		return 2
	case "data":
		return 3
	case "enum":
		return 4
	case "event":
		return 5
	case "service":
		return 6
	case "task":
		return 7
	case "web":
		return 8
	default:
		return 99
	}
}
