package cli

import (
	"context"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/internal/util/checkutil"
)

const (
	commandCheck = "check"

	flagCheckSkelIn = "skel-in"
)

func newCheckCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandCheck,
		Usage: "validate skel definition files",
		Flags: newCheckFlags(),
		Action: func(_ context.Context, cmd *ucli.Command) error {
			option := parseCheckCommand(cmd)
			result := parseCheckSource(option)
			printWarnings(cmd, result.Warnings)
			return nil
		},
	}
}

func newCheckFlags() []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{Name: flagCheckSkelIn, Usage: "skeleton input file or directory"},
	}
}

func parseCheckCommand(cmd *ucli.Command) parser.Option {
	checkutil.Check(cmd.Args().Len() == 0, "unexpected args for %s", commandCheck)
	parserOption := parser.Option{
		SkelIn: cmd.String(flagCheckSkelIn),
	}
	normalizeCheckOption(&parserOption)
	return parserOption
}

func parseCheckSource(option parser.Option) (result parser.Result) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		err, ok := recovered.(error)
		if !ok {
			panic(recovered)
		}
		if !parser.IsMissingImportError(err) {
			panic(err)
		}
		result = parser.ParseImport(option.SkelIn)
	}()
	return parser.Parse(option)
}
