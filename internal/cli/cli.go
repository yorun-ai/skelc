package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/internal/util/logutil"
)

type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

const (
	commandSkelc = "skelc"

	ExitCodeSuccess = 0
	ExitCodeError   = 1

	flagLogFormat = "log-format"

	logFormatText  = "text"
	logFormatJSONL = "jsonl"

	logLevelInfo  = string(logutil.LevelInfo)
	logLevelWarn  = string(logutil.LevelWarn)
	logLevelError = string(logutil.LevelError)

	flagOutputFormat = "output-format"

	outputFormatText = "text"
	outputFormatJSON = "json"
)

type _LogEntry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

func Main() {
	if len(os.Args) == 2 && os.Args[1] == "lsp" {
		os.Exit(runLSP())
	}
	result := Run(os.Args[1:])
	if result.Stdout != "" {
		_, _ = fmt.Fprint(os.Stdout, result.Stdout)
	}
	if result.Stderr != "" {
		_, _ = fmt.Fprint(os.Stderr, result.Stderr)
		if !strings.HasSuffix(result.Stderr, "\n") {
			_, _ = fmt.Fprintln(os.Stderr)
		}
	}
	os.Exit(result.ExitCode)
}

func Run(args []string) Result {
	if len(args) == 0 {
		args = []string{"--help"}
	}
	return runCLICommand(newCommand(), append([]string{"skelc"}, args...))
}

func newCommand() *ucli.Command {
	return &ucli.Command{
		Name:                          commandSkelc,
		Usage:                         "skeleton code generation and checking",
		Suggest:                       true,
		HideHelpCommand:               true,
		CustomHelpTemplate:            groupCommandHelpTemplate,
		CustomRootCommandHelpTemplate: groupCommandHelpTemplate,
		Flags: []ucli.Flag{
			&ucli.StringFlag{Name: flagLogFormat, Usage: "log output format: text/jsonl", Value: logFormatText},
		},
		Before: func(ctx context.Context, cmd *ucli.Command) (context.Context, error) {
			_ = commandLogFormat(cmd)
			return ctx, nil
		},
		Commands: []*ucli.Command{
			newVersionCommand(),
			newLSPCommand(),
			newGenCommand(),
			newSymbolCommand(),
			newCheckCommand(),
			newFormatCommand(),
		},
	}
}

func runCLICommand(command *ucli.Command, args []string) (result Result) {
	rawLogFormat := rawLogFormatFromArgs(args)
	logutil.Clear()
	defer recoverAsErrorResult(&result, rawLogFormat)

	var stdout strings.Builder
	var stderr strings.Builder

	command.Writer = &stdout
	command.ErrWriter = &stderr
	command.ExitErrHandler = func(_ context.Context, _ *ucli.Command, _ error) {}

	err := command.Run(context.Background(), args)
	if err != nil {
		if stderr.Len() > 0 {
			return Result{
				ExitCode: ExitCodeError,
				Stdout:   stdout.String(),
				Stderr:   logutil.Format(logutil.Error("%s", stderr.String()), rawLogFormat),
			}
		}
		return Result{ExitCode: ExitCodeError, Stdout: stdout.String(), Stderr: logutil.Format(logutil.Error("%s", err.Error()), rawLogFormat)}
	}
	if stderr.Len() > 0 {
		return Result{ExitCode: ExitCodeError, Stdout: stdout.String(), Stderr: logutil.Format(logutil.Error("%s", stderr.String()), rawLogFormat)}
	}
	return Result{ExitCode: ExitCodeSuccess, Stdout: stdout.String(), Stderr: stderr.String()}
}

func recoverAsErrorResult(result *Result, rawLogFormat string) {
	if recovered := recover(); recovered != nil {
		err := checkutil.Recover(recovered)
		*result = Result{ExitCode: ExitCodeError, Stderr: logutil.Format(logutil.Error("%s", err), rawLogFormat)}
	}
}

func printWarnings(cmd *ucli.Command, warnings []string) {
	for _, warning := range warnings {
		_, _ = fmt.Fprintln(cmd.Root().Writer, formatWarningLog(cmd, warning))
	}
}

func printLogs(cmd *ucli.Command, entries []logutil.Entry) {
	for _, entry := range entries {
		_, _ = fmt.Fprintln(cmd.Root().Writer, formatLogEntry(cmd, entry))
	}
}

func commandLogFormat(cmd *ucli.Command) string {
	value := cmd.String(flagLogFormat)
	if value == "" {
		value = cmd.Root().String(flagLogFormat)
	}
	return normalizeLogFormat(value)
}

func normalizeLogFormat(value string) string {
	if value == "" {
		return logFormatText
	}
	checkutil.Check(value == logFormatText || value == logFormatJSONL,
		"invalid log-format %q, expected text/jsonl", value)
	return value
}

func rawLogFormatFromArgs(args []string) string {
	for index, arg := range args {
		if arg == "--"+flagLogFormat && index+1 < len(args) {
			return args[index+1]
		}
		if value, ok := strings.CutPrefix(arg, "--"+flagLogFormat+"="); ok {
			return value
		}
	}
	return logFormatText
}

func formatWarningLog(cmd *ucli.Command, message string) string {
	return logutil.Format(logutil.Entry{Level: logutil.LevelWarn, Message: message}, commandLogFormat(cmd))
}

func formatLogEntry(cmd *ucli.Command, entry logutil.Entry) string {
	return logutil.Format(entry, commandLogFormat(cmd))
}

func newOutputFormatFlag(usage string) ucli.Flag {
	return &ucli.StringFlag{Name: flagOutputFormat, Usage: usage, Value: outputFormatText}
}

func commandOutputFormat(cmd *ucli.Command) string {
	value := cmd.String(flagOutputFormat)
	if value == "" {
		return outputFormatText
	}
	checkutil.Check(value == outputFormatText || value == outputFormatJSON,
		"invalid output-format %q, expected text/json", value)
	return value
}

func parseMappingFlags(values []string, flagName string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	parsed := map[string]string{}
	for _, value := range values {
		key, val, ok := strings.Cut(value, "=")
		checkutil.Check(ok && key != "" && val != "", "invalid flag %s value %q, expected name=value", flagName, value)
		_, duplicated := parsed[key]
		checkutil.CheckNot(duplicated, "duplicated flag %s key %q", flagName, key)
		parsed[key] = val
	}
	return parsed
}
