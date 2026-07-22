package cli

import (
	"context"

	"fmt"
	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/skelc/internal/parser"
)

const (
	commandCheck = "check"

	flagCheckSkelIn = "skel-in"
)

func newCheckCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandCheck,
		Usage: "validate skel definition files",
		Flags: []ucli.Flag{
			&ucli.StringFlag{Name: flagCheckSkelIn, Usage: "skeleton input file or directory"},
		},
		Action: func(_ context.Context, cmd *ucli.Command) error {
			option, err := parseCheckCommand(cmd)
			if err != nil {
				return err
			}
			result, err := parser.Check(option)
			if err != nil {
				return err
			}
			printDiagnostics(cmd, result.Diagnostics)
			if result.Diagnostics.HasErrors() {
				return result.Diagnostics.Failures()
			}
			return nil
		},
	}
}

func parseCheckCommand(cmd *ucli.Command) (parser.Option, error) {
	if cmd.Args().Len() != 0 {
		return parser.Option{}, fmt.Errorf("unexpected args for %s", commandCheck)
	}
	parserOption := parser.Option{
		SkelIn: cmd.String(flagCheckSkelIn),
	}
	return parserOption, normalizeParserOption(&parserOption)
}
