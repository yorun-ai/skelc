package parser

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/parser/analyzer"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

const (
	DiagnosticCodeSyntaxUnexpected   = "syntax.unexpected-token"
	DiagnosticCodeSyntaxEOF          = "syntax.unexpected-eof"
	DiagnosticCodeSyntaxFinalize     = "syntax.invalid-declaration"
	DiagnosticCodeSemanticValidation = analyzer.DiagnosticCodeValidation
	DiagnosticCodeSemanticDuplicate  = analyzer.DiagnosticCodeDuplicate
	DiagnosticCodeSemanticNaming     = analyzer.DiagnosticCodeNaming
	DiagnosticCodeSemanticReference  = analyzer.DiagnosticCodeReference
	DiagnosticCodeSemanticWarning    = "semantic.warning"
	DiagnosticCodeImportMissing      = "import.missing"
	DiagnosticCodeImportCycle        = "import.cycle"
	DiagnosticCodeDomainMissing      = "domain.missing"
	DiagnosticCodeDomainMismatch     = "domain.mismatch"
	DiagnosticCodeDomainFileContent  = "domain.file-content"
	DiagnosticCodeDomainDecorator    = "domain.decorator-location"
	DiagnosticCodeLoaderDirectory    = "loader.ignored-directory"
	DiagnosticCodeLoaderHiddenFile   = "loader.ignored-hidden-file"
	DiagnosticCodeLoaderUnsupported  = "loader.ignored-file"
)

type DiagnosticSeverity string

const (
	DiagnosticSeverityError   DiagnosticSeverity = "error"
	DiagnosticSeverityWarning DiagnosticSeverity = "warning"
)

type SourceRange struct {
	Start model.Position `json:"start"`
	End   model.Position `json:"end"`
}

type DiagnosticRelatedInformation struct {
	Range   SourceRange `json:"range"`
	Message string      `json:"message"`
}

type DiagnosticSuggestion struct {
	Message     string `json:"message"`
	Replacement string `json:"replacement"`
	Replace     bool   `json:"replace,omitempty"`
}

// Diagnostic is a structured compiler diagnostic for an in-memory source.
type Diagnostic struct {
	Code       string                         `json:"code"`
	Severity   DiagnosticSeverity             `json:"severity"`
	Position   model.Position                 `json:"-"`
	Range      SourceRange                    `json:"range"`
	Message    string                         `json:"message"`
	Related    []DiagnosticRelatedInformation `json:"related,omitempty"`
	Suggestion *DiagnosticSuggestion          `json:"suggestion,omitempty"`
}

func (d Diagnostic) Error() string {
	if d.Position.Line <= 0 {
		return d.Message
	}
	return d.Position.String() + " " + d.Message
}

// Diagnostics is an ordered set of independent compiler diagnostics.
type Diagnostics []Diagnostic

func (d Diagnostics) Error() string {
	if len(d) == 0 {
		return ""
	}
	return d[0].Error()
}

func (d Diagnostics) Errors() []error {
	errors := make([]error, 0, len(d))
	for index := range d {
		if d[index].Severity != DiagnosticSeverityWarning {
			errors = append(errors, d[index])
		}
	}
	return errors
}

func (d Diagnostics) HasErrors() bool {
	for _, diagnostic := range d {
		if diagnostic.Severity != DiagnosticSeverityWarning {
			return true
		}
	}
	return false
}

func (d Diagnostics) Failures() Diagnostics {
	failures := Diagnostics{}
	for _, diagnostic := range d {
		if diagnostic.Severity != DiagnosticSeverityWarning {
			failures = append(failures, diagnostic)
		}
	}
	return failures
}

func (d Diagnostics) DiagnosticEntries() Diagnostics {
	return append(Diagnostics{}, d...)
}

func diagnosticFromError(path, fallbackCode string, err error) Diagnostic {
	diagnostic := Diagnostic{
		Code: fallbackCode, Severity: DiagnosticSeverityError,
		Position: model.Position{File: path, Line: 1, Column: 1}, Message: err.Error(),
	}
	var parseError participle.Error
	if errors.As(err, &parseError) {
		parsePosition := parseError.Position()
		diagnostic.Position = workspacePosition(parsePosition)
		diagnostic.Message = parseError.Message()
		return diagnostic
	}
	if sourcePosition, ok := checkutil.Position(err); ok {
		diagnostic.Position = sourcePosition
		diagnostic.Message = strings.TrimPrefix(err.Error(), sourcePosition.String()+" ")
	}
	var failure *checkutil.Failure
	if errors.As(err, &failure) {
		if failure.Code != "" && failure.Code != checkutil.CodeValidation {
			diagnostic.Code = failure.Code
		}
		for _, related := range failure.Related {
			diagnostic.Related = append(diagnostic.Related, DiagnosticRelatedInformation{
				Range: SourceRange{Start: related.Position, End: related.Position}, Message: related.Message,
			})
		}
		if failure.Suggestion != nil {
			diagnostic.Suggestion = &DiagnosticSuggestion{
				Message: failure.Suggestion.Message, Replacement: failure.Suggestion.Replacement, Replace: failure.Suggestion.Replace,
			}
		}
	}
	var missingImport *analyzer.MissingImportError
	if errors.As(err, &missingImport) {
		diagnostic.Code = DiagnosticCodeImportMissing
	}
	diagnostic.Range = SourceRange{Start: diagnostic.Position, End: diagnostic.Position}
	return diagnostic
}

func completeDiagnostic(diagnostic *Diagnostic, contents map[string][]byte) {
	if diagnostic.Severity == "" {
		diagnostic.Severity = DiagnosticSeverityError
	}
	if diagnostic.Range.Start.Line <= 0 || diagnostic.Range.End.Line <= 0 || diagnostic.Range.End == diagnostic.Range.Start {
		diagnostic.Range = sourceRangeAt(diagnostic.Position, contents[filepath.Clean(diagnostic.Position.File)])
	}
	for index := range diagnostic.Related {
		position := diagnostic.Related[index].Range.Start
		if diagnostic.Related[index].Range.End.Line <= 0 || diagnostic.Related[index].Range.End == position {
			diagnostic.Related[index].Range = sourceRangeAt(position, contents[filepath.Clean(position.File)])
		}
	}
}

func workspacePosition(position lexer.Position) model.Position {
	return model.Position{File: position.Filename, Line: position.Line, Column: position.Column}
}

func compareDiagnostics(left, right Diagnostic) int {
	if compared := strings.Compare(left.Position.File, right.Position.File); compared != 0 {
		return compared
	}
	if left.Position.Line != right.Position.Line {
		return left.Position.Line - right.Position.Line
	}
	if left.Position.Column != right.Position.Column {
		return left.Position.Column - right.Position.Column
	}
	return strings.Compare(left.Message, right.Message)
}
