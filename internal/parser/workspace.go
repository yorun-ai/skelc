package parser

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/loader"
	"go.yorun.ai/skelc/internal/parser/analyzer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

const (
	DiagnosticCodeSyntaxUnexpected   = "syntax.unexpected-token"
	DiagnosticCodeSyntaxEOF          = "syntax.unexpected-eof"
	DiagnosticCodeSyntaxFinalize     = "syntax.invalid-declaration"
	DiagnosticCodeSemanticValidation = "semantic.validation"
	DiagnosticCodeSemanticDuplicate  = "semantic.duplicate"
	DiagnosticCodeSemanticNaming     = "semantic.naming"
	DiagnosticCodeSemanticReference  = "semantic.reference"
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

// Source is an in-memory Skel document used by workspace analysis. Domain is a
// best-effort hint used to suppress cascading diagnostics when syntax is
// temporarily incomplete and the full domain declaration cannot be parsed.
type Source struct {
	Path             string
	Domain           string
	ExpectedDomain   string
	Content          []byte
	Parsed           *grammar.SkelContent
	ParseDiagnostics Diagnostics
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

type _WorkspaceDomain struct {
	name          string
	contents      []*grammar.SkelContent
	invalid       bool
	syntaxInvalid bool
	merged        *grammar.SkelContent
	analysis      *analyzer.Analysis
	state         _WorkspaceDomainState
	sources       []Source
	fingerprint   string
}

type _CachedWorkspaceParse struct {
	hash        [32]byte
	content     *grammar.SkelContent
	diagnostics Diagnostics
}

type _CachedWorkspaceDomain struct {
	fingerprint string
	analysis    *analyzer.Analysis
}

// WorkspaceAnalyzer caches syntax trees and successful domain analyses across
// workspace snapshots. A changed domain invalidates only itself and reverse
// dependents whose import fingerprints consequently change.
type WorkspaceAnalyzer struct {
	mu      sync.Mutex
	parses  map[string]_CachedWorkspaceParse
	domains map[string]_CachedWorkspaceDomain
	stats   WorkspaceAnalysisStats
}

type WorkspaceAnalysisStats struct {
	ParsedSources   int
	ReusedSources   int
	AnalyzedDomains int
	ReusedDomains   int
}

func NewWorkspaceAnalyzer() *WorkspaceAnalyzer {
	return &WorkspaceAnalyzer{parses: map[string]_CachedWorkspaceParse{}, domains: map[string]_CachedWorkspaceDomain{}}
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
	return NewWorkspaceAnalyzer().Analyze(sources)
}

func (w *WorkspaceAnalyzer) Analyze(sources []Source) []Diagnostic {
	diagnostics, _ := w.AnalyzeContext(context.Background(), sources)
	return diagnostics
}

func (w *WorkspaceAnalyzer) AnalyzeContext(ctx context.Context, sources []Source) ([]Diagnostic, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stats = WorkspaceAnalysisStats{}
	return w.analyze(ctx, sources, false)
}

func (w *WorkspaceAnalyzer) Stats() WorkspaceAnalysisStats {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.stats
}

func (w *WorkspaceAnalyzer) analyze(ctx context.Context, sources []Source, allowMissingImports bool) ([]Diagnostic, error) {
	ordered := append([]Source{}, sources...)
	slices.SortFunc(ordered, func(left, right Source) int {
		return strings.Compare(left.Path, right.Path)
	})

	domains := map[string]*_WorkspaceDomain{}
	diagnostics := []Diagnostic{}
	for _, source := range ordered {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		content, syntaxDiagnostics := w.parseWorkspaceSource(source)
		diagnostics = append(diagnostics, syntaxDiagnostics...)
		if content == nil {
			if source.Domain != "" {
				domain := workspaceDomain(domains, source.Domain)
				domain.invalid = true
			}
			continue
		}
		if content.Domain == nil || content.Domain.Name == nil || content.Domain.Name.String() == "" {
			position := model.Position{File: source.Path, Line: 1, Column: 1}
			diagnostics = append(diagnostics, Diagnostic{
				Code: DiagnosticCodeDomainMissing, Severity: DiagnosticSeverityError, Position: position, Range: sourceRangeAt(position, source.Content),
				Message: "missing domain declaration",
			})
			continue
		}
		name := content.Domain.Name.String()
		if source.ExpectedDomain != "" && name != source.ExpectedDomain {
			position := workspacePosition(content.Domain.Name.Pos)
			diagnostics = append(diagnostics, Diagnostic{
				Code: DiagnosticCodeDomainMismatch, Severity: DiagnosticSeverityError, Position: position, Range: sourceRangeAt(position, source.Content),
				Message: fmt.Sprintf("domain mismatch: found=%s, expected=%s", name, source.ExpectedDomain),
			})
			workspaceDomain(domains, source.ExpectedDomain).invalid = true
			continue
		}
		domain := workspaceDomain(domains, name)
		if len(syntaxDiagnostics) > 0 {
			domain.syntaxInvalid = true
		}
		domain.contents = append(domain.contents, content)
		domain.sources = append(domain.sources, source)
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
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		w.analyzeWorkspaceDomain(ctx, domains[name], domains, &diagnostics, allowMissingImports)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	contentByPath := make(map[string][]byte, len(ordered))
	for _, source := range ordered {
		contentByPath[filepath.Clean(source.Path)] = source.Content
	}
	for index := range diagnostics {
		completeDiagnostic(&diagnostics[index], contentByPath)
	}
	slices.SortFunc(diagnostics, compareDiagnostics)
	activePaths := make(map[string]bool, len(ordered))
	for _, source := range ordered {
		activePaths[source.Path] = true
	}
	for path := range w.parses {
		if !activePaths[path] {
			delete(w.parses, path)
		}
	}
	for name := range w.domains {
		if domains[name] == nil {
			delete(w.domains, name)
		}
	}
	return diagnostics, nil
}

func (w *WorkspaceAnalyzer) parseWorkspaceSource(source Source) (*grammar.SkelContent, Diagnostics) {
	if source.Parsed != nil || len(source.ParseDiagnostics) > 0 {
		w.stats.ReusedSources++
		if len(source.ParseDiagnostics) > 0 {
			return source.Parsed, append(Diagnostics{}, source.ParseDiagnostics...)
		}
		return source.Parsed, nil
	}
	hash := sha256.Sum256(source.Content)
	if cached, ok := w.parses[source.Path]; ok && cached.hash == hash {
		w.stats.ReusedSources++
		return cached.content, append(Diagnostics{}, cached.diagnostics...)
	}
	w.stats.ParsedSources++
	content, diagnostics := ParseSourceRecovering(source.Path, source.Content)
	w.parses[source.Path] = _CachedWorkspaceParse{hash: hash, content: content, diagnostics: append(Diagnostics{}, diagnostics...)}
	return content, diagnostics
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

func (w *WorkspaceAnalyzer) analyzeWorkspaceDomain(
	ctx context.Context,
	domain *_WorkspaceDomain,
	domains map[string]*_WorkspaceDomain,
	diagnostics *[]Diagnostic,
	allowMissingImports bool,
) bool {
	if ctx.Err() != nil {
		return false
	}
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
			Code: DiagnosticCodeImportCycle, Severity: DiagnosticSeverityError, Position: position, Range: SourceRange{Start: position, End: position},
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
		if ctx.Err() != nil {
			return false
		}
		name := importDecl.Domain.String()
		if seenImports[name] {
			continue
		}
		seenImports[name] = true
		imported := domains[name]
		if imported != nil && (imported.invalid || imported.syntaxInvalid) {
			importsValid = false
			continue
		}
		if imported == nil || imported.merged == nil {
			if allowMissingImports {
				hasMissingImports = true
				continue
			}
			*diagnostics = append(*diagnostics, Diagnostic{
				Code: DiagnosticCodeImportMissing, Severity: DiagnosticSeverityError, Position: workspacePosition(importDecl.Pos),
				Message: fmt.Sprintf("skel import %s not found in the workspace", name),
			})
			importsValid = false
			continue
		}
		if !w.analyzeWorkspaceDomain(ctx, imported, domains, diagnostics, allowMissingImports) {
			importsValid = false
			continue
		}
		imports = append(imports, imported.analysis)
	}
	if !importsValid {
		domain.state = workspaceDomainFailed
		return false
	}

	domain.fingerprint = workspaceDomainFingerprint(domain, domains)
	if cached, ok := w.domains[domain.name]; ok && cached.fingerprint == domain.fingerprint {
		w.stats.ReusedDomains++
		domain.analysis = cached.analysis
		domain.state = workspaceDomainComplete
		return true
	}
	var analysis *analyzer.Analysis
	var analysisErrors []error
	w.stats.AnalyzedDomains++
	if hasMissingImports {
		analysis, analysisErrors = analyzer.AnalyzeImport(domain.merged)
	} else {
		analysis, analysisErrors = analyzer.Analyze(domain.merged, imports)
	}
	if len(analysisErrors) > 0 {
		for _, analysisError := range analysisErrors {
			*diagnostics = append(*diagnostics, diagnosticFromError(domain.merged.Pos.Filename, DiagnosticCodeSemanticValidation, analysisError))
		}
		domain.state = workspaceDomainFailed
		return false
	}
	domain.analysis = analysis
	domain.state = workspaceDomainComplete
	w.domains[domain.name] = _CachedWorkspaceDomain{fingerprint: domain.fingerprint, analysis: analysis}
	return true
}

func workspaceDomainFingerprint(domain *_WorkspaceDomain, domains map[string]*_WorkspaceDomain) string {
	hash := sha256.New()
	ordered := append([]Source{}, domain.sources...)
	slices.SortFunc(ordered, func(left, right Source) int { return strings.Compare(left.Path, right.Path) })
	for _, source := range ordered {
		_, _ = hash.Write([]byte(source.Path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(source.Content)
		_, _ = hash.Write([]byte{0})
	}
	imports := append([]*grammar.ImportDecl{}, domain.merged.Imports...)
	slices.SortFunc(imports, func(left, right *grammar.ImportDecl) int {
		return strings.Compare(left.Domain.String(), right.Domain.String())
	})
	for _, importDecl := range imports {
		name := importDecl.Domain.String()
		_, _ = hash.Write([]byte(name))
		if imported := domains[name]; imported != nil {
			_, _ = hash.Write([]byte(imported.fingerprint))
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
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
	if errors.As(err, &failure) && failure.Code != "" && failure.Code != checkutil.CodeValidation {
		diagnostic.Code = failure.Code
	}
	if errors.As(err, &failure) {
		for _, related := range failure.Related {
			diagnostic.Related = append(diagnostic.Related, DiagnosticRelatedInformation{
				Range: SourceRange{Start: related.Position, End: related.Position}, Message: related.Message,
			})
		}
	}
	var missingImport *analyzer.MissingImportError
	if errors.As(err, &missingImport) {
		diagnostic.Code = DiagnosticCodeImportMissing
	}
	diagnostic.Code = semanticDiagnosticCode(diagnostic.Code, diagnostic.Message)
	diagnostic.Range = SourceRange{Start: diagnostic.Position, End: diagnostic.Position}
	return diagnostic
}

var namingSuggestionPattern = regexp.MustCompile(`expected=([^ ]+)`)

func semanticDiagnosticCode(code, message string) string {
	if code != "" && code != checkutil.CodeValidation && code != DiagnosticCodeSemanticValidation {
		return code
	}
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "duplicated"):
		return DiagnosticCodeSemanticDuplicate
	case strings.Contains(lower, "incorrect case"):
		return DiagnosticCodeSemanticNaming
	case strings.Contains(lower, "unknown "), strings.Contains(lower, "undefined "), strings.Contains(lower, "not found"):
		return DiagnosticCodeSemanticReference
	default:
		return DiagnosticCodeSemanticValidation
	}
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
	if diagnostic.Code == DiagnosticCodeSemanticNaming && diagnostic.Suggestion == nil {
		match := namingSuggestionPattern.FindStringSubmatch(diagnostic.Message)
		if len(match) == 2 {
			diagnostic.Suggestion = &DiagnosticSuggestion{Message: "replace with " + match[1], Replacement: match[1], Replace: true}
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
