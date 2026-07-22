package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/skelc/internal/parser"
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
	Level    string                    `json:"level"`
	Code     string                    `json:"code,omitempty"`
	Severity parser.DiagnosticSeverity `json:"severity,omitempty"`
	Range    parser.SourceRange        `json:"range,omitempty"`
	Message  string                    `json:"message"`
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
			return ctx, validateLogFormat(cmd)
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

	var stdout strings.Builder
	var stderr strings.Builder

	command.Writer = &stdout
	command.ErrWriter = &stderr
	command.ExitErrHandler = func(_ context.Context, _ *ucli.Command, _ error) {}

	err := command.Run(context.Background(), args)
	if err != nil {
		if diagnostics, ok := err.(interface{ DiagnosticEntries() parser.Diagnostics }); ok {
			return Result{
				ExitCode: ExitCodeError, Stdout: stdout.String(),
				Stderr: formatDiagnostics(diagnostics.DiagnosticEntries(), rawLogFormat),
			}
		}
		if diagnostics, ok := err.(interface{ Errors() []error }); ok {
			return Result{
				ExitCode: ExitCodeError, Stdout: stdout.String(),
				Stderr: formatErrors(diagnostics.Errors(), rawLogFormat),
			}
		}
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

func formatDiagnostics(diagnostics parser.Diagnostics, format string) string {
	if format != logFormatJSONL {
		var output strings.Builder
		for _, diagnostic := range diagnostics {
			level := logutil.LevelError
			if diagnostic.Severity == parser.DiagnosticSeverityWarning {
				level = logutil.LevelWarn
			}
			formatted := logutil.Format(logutil.Entry{Level: level, Message: diagnostic.Error()}, format)
			output.WriteString(formatted)
			if !strings.HasSuffix(formatted, "\n") {
				output.WriteByte('\n')
			}
		}
		return output.String()
	}
	var output strings.Builder
	for _, diagnostic := range diagnostics {
		level := logLevelError
		if diagnostic.Severity == parser.DiagnosticSeverityWarning {
			level = logLevelWarn
		}
		entry := struct {
			Level string `json:"level"`
			parser.Diagnostic
		}{Level: level, Diagnostic: diagnostic}
		content, err := json.Marshal(entry)
		if err != nil {
			return formatErrors([]error{err}, format)
		}
		output.Write(content)
		output.WriteByte('\n')
	}
	return output.String()
}

func formatErrors(errors []error, format string) string {
	var output strings.Builder
	for _, err := range errors {
		formatted := logutil.Format(logutil.Entry{Level: logutil.LevelError, Message: err.Error()}, format)
		output.WriteString(formatted)
		if !strings.HasSuffix(formatted, "\n") {
			output.WriteByte('\n')
		}
	}
	return output.String()
}

func printDiagnostics(cmd *ucli.Command, diagnostics []parser.Diagnostic) {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != parser.DiagnosticSeverityWarning {
			continue
		}
		_, _ = fmt.Fprint(cmd.Root().Writer, formatDiagnostics(parser.Diagnostics{diagnostic}, commandLogFormat(cmd)))
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
	if value == "" {
		return logFormatText
	}
	return value
}

func normalizeLogFormat(value string) (string, error) {
	if value == "" {
		return logFormatText, nil
	}
	if value != logFormatText && value != logFormatJSONL {
		return "", fmt.Errorf("invalid log-format %q, expected text/jsonl", value)
	}
	return value, nil
}

func validateLogFormat(cmd *ucli.Command) error {
	_, err := normalizeLogFormat(commandLogFormat(cmd))
	return err
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

func commandOutputFormat(cmd *ucli.Command) (string, error) {
	value := cmd.String(flagOutputFormat)
	if value == "" {
		return outputFormatText, nil
	}
	if value != outputFormatText && value != outputFormatJSON {
		return "", fmt.Errorf("invalid output-format %q, expected text/json", value)
	}
	return value, nil
}

func parseMappingFlags(values []string, flagName string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	parsed := map[string]string{}
	for _, value := range values {
		key, val, ok := strings.Cut(value, "=")
		if !ok || key == "" || val == "" {
			return nil, fmt.Errorf("invalid flag %s value %q, expected name=value", flagName, value)
		}
		_, duplicated := parsed[key]
		if duplicated {
			return nil, fmt.Errorf("duplicated flag %s key %q", flagName, key)
		}
		parsed[key] = val
	}
	return parsed, nil
}
