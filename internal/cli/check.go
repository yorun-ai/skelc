package cli

import (
	"context"
	"fmt"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/skelc/internal/command"
	"go.yorun.ai/skelc/internal/compiler"
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
				return commandFailure(command.ErrorCodeInvalidArgument, err)
			}
			result, err := compiler.Check(option)
			if err != nil {
				return commandFailure(command.ErrorCodeCompilationFailed, err)
			}
			diagnostics := result.Diagnostics
			if diagnostics == nil {
				diagnostics = compiler.Diagnostics{}
			}
			valid := !diagnostics.HasErrors()
			if err := writeJSONResult(cmd, command.CheckResult{Valid: valid, Diagnostics: diagnostics}, "check result"); err != nil {
				return commandFailure(command.ErrorCodeCommandFailed, err)
			}
			if !valid {
				return commandUnsatisfied()
			}
			return nil
		},
	}
}

func parseCheckCommand(cmd *ucli.Command) (compiler.Option, error) {
	if cmd.Args().Len() != 0 {
		return compiler.Option{}, fmt.Errorf("unexpected args for %s", commandCheck)
	}
	compilerOption := compiler.Option{
		SkelIn: cmd.String(flagCheckSkelIn),
	}
	return compilerOption, normalizeCompilerOption(&compilerOption)
}
