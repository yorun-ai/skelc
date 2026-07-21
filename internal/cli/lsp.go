package cli

import (
	"context"
	"fmt"
	"os"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/skelc/internal/lsp"
)

func newLSPCommand() *ucli.Command {
	return &ucli.Command{
		Name:  "lsp",
		Usage: "run the Skel language server over standard input and output",
		Action: func(context.Context, *ucli.Command) error {
			return fmt.Errorf("lsp must be run as a stdio command")
		},
	}
}

func runLSP() int {
	if err := lsp.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return ExitCodeError
	}
	return ExitCodeSuccess
}
