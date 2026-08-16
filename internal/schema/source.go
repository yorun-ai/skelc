package schema

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.yorun.ai/skelc/internal/compiler"
)

// SourceDiffOption configures a source compatibility comparison.
type SourceDiffOption struct {
	// BaselineSkelIn selects an explicit baseline file or directory. An empty
	// value reads the domain's source directory from Git HEAD.
	BaselineSkelIn string
}

type _CachedSourceBaseline struct {
	head           string
	repositoryRoot string
	document       *Document
	err            error
}

type _CachedGitFailure struct {
	expires time.Time
	err     error
}

const gitFailureCacheDuration = 2 * time.Second

// SourceDiffer caches the current immutable Git baseline for each domain so
// repeated editor analysis does not re-read and recompile unchanged history.
type SourceDiffer struct {
	mu          sync.Mutex
	baselines   map[string]_CachedSourceBaseline
	gitFailures map[string]_CachedGitFailure
	now         func() time.Time
}

// NewSourceDiffer creates a source differ with an empty Git baseline cache.
func NewSourceDiffer() *SourceDiffer {
	return &SourceDiffer{
		baselines: map[string]_CachedSourceBaseline{}, gitFailures: map[string]_CachedGitFailure{}, now: time.Now,
	}
}

// DiffWorkspaceDomain compares one successfully analyzed in-memory domain with
// either an explicit source baseline or the same source directory at Git HEAD.
func DiffWorkspaceDomain(ctx context.Context, candidate compiler.WorkspaceDomain, option SourceDiffOption) (*Report, error) {
	return NewSourceDiffer().DiffWorkspaceDomain(ctx, candidate, option)
}

// DiffSource compares a candidate file or directory with either an explicit
// source baseline or the same path at Git HEAD.
func DiffSource(ctx context.Context, candidateSkelIn string, option SourceDiffOption) (*Report, error) {
	candidate, err := projectSource(candidateSkelIn)
	if err != nil {
		return nil, err
	}
	baselineSkelIn := strings.TrimSpace(option.BaselineSkelIn)
	var gitBaseline *_GitSourceBaseline
	if baselineSkelIn == "" {
		gitBaseline, err = prepareGitSourceBaseline(ctx, candidateSkelIn)
		if err != nil {
			return nil, err
		}
		defer gitBaseline.cleanup()
		baselineSkelIn = gitBaseline.skelIn
	}
	baseline, err := projectSource(baselineSkelIn)
	if err != nil {
		if gitBaseline != nil {
			return nil, gitBaseline.remapError(err)
		}
		return nil, err
	}
	report, err := Diff(baseline, candidate)
	if err != nil {
		return nil, err
	}
	if gitBaseline != nil {
		gitBaseline.remapReportPositions(report)
	}
	return report, nil
}

// DiffWorkspaceDomain compares a domain while reusing its unchanged Git
// baseline across calls to the same differ.
func (d *SourceDiffer) DiffWorkspaceDomain(ctx context.Context, candidate compiler.WorkspaceDomain, option SourceDiffOption) (*Report, error) {
	candidateSchema, err := Project(candidate.Model, nil)
	if err != nil {
		return nil, err
	}
	baselineSkelIn := strings.TrimSpace(option.BaselineSkelIn)
	var baselineSchema *Document
	gitRepositoryRoot := ""
	if baselineSkelIn == "" {
		baselineSchema, gitRepositoryRoot, err = projectGitBaseline(ctx, d, candidate)
	} else {
		if !filepath.IsAbs(baselineSkelIn) {
			baselineSkelIn = filepath.Join(candidate.Root, baselineSkelIn)
		}
		baseline, compileErr := compiler.CompileImport(baselineSkelIn)
		if compileErr != nil {
			return nil, fmt.Errorf("compile schema compatibility baseline %s: %w", baselineSkelIn, compileErr)
		}
		baselineSchema, err = Project(baseline.Domain, baseline.ImportAliases)
	}
	if err != nil {
		return nil, err
	}
	report, err := Diff(baselineSchema, candidateSchema)
	if err != nil {
		return nil, err
	}
	if gitRepositoryRoot != "" {
		remapReportBaselinePositions(report, gitRepositoryRoot)
	}
	return report, nil
}

func projectSource(skelIn string) (*Document, error) {
	result, err := compiler.CompileImport(skelIn)
	if err != nil {
		return nil, err
	}
	return Project(result.Domain, result.ImportAliases)
}

func (d *SourceDiffer) cachedBaseline(key, head string) (*Document, string, error, bool) {
	if d == nil {
		return nil, "", nil, false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	baseline := d.baselines[key]
	if baseline.head != head {
		return nil, "", nil, false
	}
	return baseline.document, baseline.repositoryRoot, baseline.err, true
}

func (d *SourceDiffer) storeBaseline(key, head, repositoryRoot string, document *Document, err error) {
	if d == nil || (document == nil && err == nil) {
		return
	}
	d.mu.Lock()
	if d.baselines == nil {
		d.baselines = map[string]_CachedSourceBaseline{}
	}
	d.baselines[key] = _CachedSourceBaseline{
		head: head, repositoryRoot: repositoryRoot, document: document, err: err,
	}
	d.mu.Unlock()
}

func (d *SourceDiffer) cachedGitFailure(root string) error {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	failure, ok := d.gitFailures[root]
	if !ok {
		return nil
	}
	if !d.currentTime().Before(failure.expires) {
		delete(d.gitFailures, root)
		return nil
	}
	return failure.err
}

func (d *SourceDiffer) storeGitFailure(root string, err error) {
	if d == nil || err == nil {
		return
	}
	d.mu.Lock()
	if d.gitFailures == nil {
		d.gitFailures = map[string]_CachedGitFailure{}
	}
	d.gitFailures[root] = _CachedGitFailure{expires: d.currentTime().Add(gitFailureCacheDuration), err: err}
	d.mu.Unlock()
}

func (d *SourceDiffer) clearGitFailure(root string) {
	if d == nil {
		return
	}
	d.mu.Lock()
	delete(d.gitFailures, root)
	d.mu.Unlock()
}

func (d *SourceDiffer) currentTime() time.Time {
	if d != nil && d.now != nil {
		return d.now()
	}
	return time.Now()
}

func remapReportBaselinePositions(report *Report, repositoryRoot string) {
	for _, change := range report.Changes {
		if change.Baseline == nil || change.Baseline.File == "" {
			continue
		}
		relative, err := filepath.Rel(repositoryRoot, change.Baseline.File)
		if err == nil && !pathEscapesRoot(relative) {
			change.Baseline.File = "HEAD:" + filepath.ToSlash(relative)
		}
	}
}
