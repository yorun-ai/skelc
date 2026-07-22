package parser

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/loader"
	"go.yorun.ai/skelc/internal/parser/analyzer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

const (
	DiagnosticCodeSyntax      = "syntax"
	DiagnosticCodeSemantic    = "semantic"
	DiagnosticCodeImport      = "missing-import"
	DiagnosticCodeImportCycle = "import-cycle"
)

// Source is an in-memory Skel document used by workspace analysis. Domain is a
// best-effort hint used to suppress cascading diagnostics when syntax is
// temporarily incomplete and the full domain declaration cannot be parsed.
type Source struct {
	Path           string
	Domain         string
	ExpectedDomain string
	Content        []byte
}

// Diagnostic is a structured compiler diagnostic for an in-memory source.
type Diagnostic struct {
	Code     string
	Position model.Position
	Message  string
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
	errors := make([]error, len(d))
	for index := range d {
		errors[index] = d[index]
	}
	return errors
}

type _WorkspaceDomain struct {
	name     string
	contents []*grammar.SkelContent
	invalid  bool
	merged   *grammar.SkelContent
	analysis *analyzer.Analysis
	state    _WorkspaceDomainState
}

type _WorkspaceDomainState uint8

const (
	workspaceDomainPending _WorkspaceDomainState = iota
	workspaceDomainVisiting
	workspaceDomainComplete
	workspaceDomainFailed
)

// AnalyzeWorkspace performs syntax and semantic analysis over an in-memory
// workspace. Independent failures in the same domain are collected up to the
// analyzer's diagnostic limit. Domains that depend on a syntactically or
// semantically invalid domain are skipped to avoid cascading errors.
func AnalyzeWorkspace(sources []Source) []Diagnostic {
	return analyzeWorkspace(sources, false)
}

func analyzeWorkspace(sources []Source, allowMissingImports bool) []Diagnostic {
	ordered := append([]Source{}, sources...)
	slices.SortFunc(ordered, func(left, right Source) int {
		return strings.Compare(left.Path, right.Path)
	})

	domains := map[string]*_WorkspaceDomain{}
	diagnostics := []Diagnostic{}
	for _, source := range ordered {
		content, err := parseWorkspaceSource(source)
		if err != nil {
			diagnostics = append(diagnostics, diagnosticFromError(source.Path, DiagnosticCodeSyntax, err))
			if source.Domain != "" {
				domain := workspaceDomain(domains, source.Domain)
				domain.invalid = true
			}
			continue
		}
		if content.Domain == nil || content.Domain.Name == nil || content.Domain.Name.String() == "" {
			diagnostics = append(diagnostics, Diagnostic{
				Code: DiagnosticCodeSyntax, Position: model.Position{File: source.Path, Line: 1, Column: 1},
				Message: "missing domain declaration",
			})
			continue
		}
		name := content.Domain.Name.String()
		if source.ExpectedDomain != "" && name != source.ExpectedDomain {
			diagnostics = append(diagnostics, Diagnostic{
				Code: DiagnosticCodeSemantic, Position: workspacePosition(content.Domain.Name.Pos),
				Message: fmt.Sprintf("domain mismatch: found=%s, expected=%s", name, source.ExpectedDomain),
			})
			workspaceDomain(domains, source.ExpectedDomain).invalid = true
			continue
		}
		domain := workspaceDomain(domains, name)
		domain.contents = append(domain.contents, content)
	}

	names := make([]string, 0, len(domains))
	for name, domain := range domains {
		if len(domain.contents) > 0 {
			domain.merged = mergeWorkspaceContents(domain.contents)
		}
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		analyzeWorkspaceDomain(domains[name], domains, &diagnostics, allowMissingImports)
	}
	slices.SortFunc(diagnostics, compareDiagnostics)
	return diagnostics
}

func parseWorkspaceSource(source Source) (content *grammar.SkelContent, err error) {
	var parseErr error
	captureErr := checkutil.Capture(func() {
		content, parseErr = ParseSource(source.Path, source.Content)
	})
	if captureErr != nil {
		return nil, captureErr
	}
	return content, parseErr
}

func workspaceDomain(domains map[string]*_WorkspaceDomain, name string) *_WorkspaceDomain {
	domain := domains[name]
	if domain == nil {
		domain = &_WorkspaceDomain{name: name}
		domains[name] = domain
	}
	return domain
}

func mergeWorkspaceContents(contents []*grammar.SkelContent) *grammar.SkelContent {
	ordered := append([]*grammar.SkelContent{}, contents...)
	slices.SortFunc(ordered, func(left, right *grammar.SkelContent) int {
		return strings.Compare(left.Pos.Filename, right.Pos.Filename)
	})
	domainContent := ordered[0].Domain
	for _, content := range ordered {
		if filepath.Base(content.Pos.Filename) == loader.DomainFileName {
			domainContent = content.Domain
			break
		}
	}
	merged := &grammar.SkelContent{Pos: domainContent.Pos, Domain: domainContent}
	for _, content := range ordered {
		merged.Imports = append(merged.Imports, content.Imports...)
		merged.Entries = append(merged.Entries, content.Entries...)
	}
	return merged
}

func analyzeWorkspaceDomain(
	domain *_WorkspaceDomain,
	domains map[string]*_WorkspaceDomain,
	diagnostics *[]Diagnostic,
	allowMissingImports bool,
) bool {
	if domain == nil || domain.invalid || domain.merged == nil {
		return false
	}
	switch domain.state {
	case workspaceDomainComplete:
		return true
	case workspaceDomainFailed:
		return false
	case workspaceDomainVisiting:
		position := workspacePosition(domain.merged.Domain.Name.Pos)
		*diagnostics = append(*diagnostics, Diagnostic{
			Code: DiagnosticCodeImportCycle, Position: position,
			Message: fmt.Sprintf("cyclic domain import involving %s", domain.name),
		})
		domain.state = workspaceDomainFailed
		return false
	}

	domain.state = workspaceDomainVisiting
	imports := make([]*analyzer.Analysis, 0, len(domain.merged.Imports))
	seenImports := map[string]bool{}
	importsValid := true
	hasMissingImports := false
	for _, importDecl := range domain.merged.Imports {
		name := importDecl.Domain.String()
		if seenImports[name] {
			continue
		}
		seenImports[name] = true
		imported := domains[name]
		if imported != nil && imported.invalid {
			importsValid = false
			continue
		}
		if imported == nil || imported.merged == nil {
			if allowMissingImports {
				hasMissingImports = true
				continue
			}
			*diagnostics = append(*diagnostics, Diagnostic{
				Code: DiagnosticCodeImport, Position: workspacePosition(importDecl.Pos),
				Message: fmt.Sprintf("skel import %s not found in the workspace", name),
			})
			importsValid = false
			continue
		}
		if !analyzeWorkspaceDomain(imported, domains, diagnostics, allowMissingImports) {
			importsValid = false
			continue
		}
		imports = append(imports, imported.analysis)
	}
	if !importsValid {
		domain.state = workspaceDomainFailed
		return false
	}

	var analysis *analyzer.Analysis
	var analysisErrors []error
	err := checkutil.Capture(func() {
		if hasMissingImports {
			analysis, analysisErrors = analyzer.AnalyzeImport(domain.merged)
		} else {
			analysis, analysisErrors = analyzer.Analyze(domain.merged, imports)
		}
	})
	if err != nil {
		*diagnostics = append(*diagnostics, diagnosticFromError(domain.merged.Pos.Filename, DiagnosticCodeSemantic, err))
		domain.state = workspaceDomainFailed
		return false
	}
	if len(analysisErrors) > 0 {
		for _, analysisError := range analysisErrors {
			*diagnostics = append(*diagnostics, diagnosticFromError(domain.merged.Pos.Filename, DiagnosticCodeSemantic, analysisError))
		}
		domain.state = workspaceDomainFailed
		return false
	}
	domain.analysis = analysis
	domain.state = workspaceDomainComplete
	return true
}

func diagnosticFromError(path, fallbackCode string, err error) Diagnostic {
	diagnostic := Diagnostic{
		Code: fallbackCode, Position: model.Position{File: path, Line: 1, Column: 1}, Message: err.Error(),
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
	if errors.As(err, &failure) && failure.Code != "" && failure.Code != checkutil.CodeValidation {
		diagnostic.Code = failure.Code
	}
	var missingImport *analyzer.MissingImportError
	if errors.As(err, &missingImport) {
		diagnostic.Code = DiagnosticCodeImport
	}
	return diagnostic
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
