package schema

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type _GitSourceBaseline struct {
	skelIn      string
	archiveRoot string
}

type _GitSourceBaselineError struct {
	cause   error
	message string
}

func (e *_GitSourceBaselineError) Error() string { return e.message }

func (e *_GitSourceBaselineError) Unwrap() error { return e.cause }

func prepareGitSourceBaseline(ctx context.Context, candidateSkelIn string) (*_GitSourceBaseline, error) {
	target, err := filepath.Abs(candidateSkelIn)
	if err != nil {
		return nil, fmt.Errorf("resolve candidate skel input: %w", err)
	}
	target, err = filepath.EvalSymlinks(target)
	if err != nil {
		return nil, fmt.Errorf("resolve candidate skel input %s: %w", target, err)
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("stat candidate skel input %s: %w", target, err)
	}
	probe := target
	if !info.IsDir() {
		probe = filepath.Dir(target)
	}
	repositoryRoot, err := gitOutput(ctx, probe, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, gitHistoryNotFound(target, err)
	}
	repositoryRoot = strings.TrimSpace(repositoryRoot)
	if repositoryRoot == "" {
		return nil, gitHistoryNotFound(target, nil)
	}
	repositoryRoot, err = filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return nil, gitHistoryNotFound(target, err)
	}
	relativeTarget, err := filepath.Rel(repositoryRoot, target)
	if err != nil || pathEscapesRoot(relativeTarget) {
		return nil, gitHistoryNotFound(target, err)
	}
	gitTarget := filepath.ToSlash(relativeTarget)
	archive, err := gitBytes(ctx, repositoryRoot, "archive", "--format=tar", "HEAD", "--", gitTarget)
	if err != nil {
		return nil, gitHistoryNotFound(target, err)
	}
	archiveRoot, err := os.MkdirTemp("", "skelc-schema-baseline-")
	if err != nil {
		return nil, fmt.Errorf("create Git baseline directory: %w", err)
	}
	extracted, err := extractGitArchive(archive, archiveRoot)
	if err != nil {
		_ = os.RemoveAll(archiveRoot)
		return nil, fmt.Errorf("extract Git baseline for %s: %w", target, err)
	}
	if !extracted {
		_ = os.RemoveAll(archiveRoot)
		return nil, gitHistoryNotFound(target, nil)
	}
	baselineSkelIn := archiveRoot
	if relativeTarget != "." {
		baselineSkelIn = filepath.Join(archiveRoot, relativeTarget)
	}
	if _, err := os.Stat(baselineSkelIn); err != nil {
		_ = os.RemoveAll(archiveRoot)
		return nil, gitHistoryNotFound(target, err)
	}
	return &_GitSourceBaseline{skelIn: baselineSkelIn, archiveRoot: archiveRoot}, nil
}

func (b *_GitSourceBaseline) cleanup() {
	if b != nil && b.archiveRoot != "" {
		_ = os.RemoveAll(b.archiveRoot)
	}
}

func (b *_GitSourceBaseline) remapError(err error) error {
	if b == nil || b.archiveRoot == "" || err == nil {
		return err
	}
	message := strings.ReplaceAll(err.Error(), b.archiveRoot+string(filepath.Separator), "HEAD:")
	message = strings.ReplaceAll(message, b.archiveRoot, "HEAD:.")
	return &_GitSourceBaselineError{cause: err, message: message}
}

func (b *_GitSourceBaseline) remapReportPositions(report *Report) {
	if b == nil || report == nil {
		return
	}
	for _, change := range report.Changes {
		if change.Baseline == nil || change.Baseline.File == "" {
			continue
		}
		relativePath, err := filepath.Rel(b.archiveRoot, change.Baseline.File)
		if err != nil || pathEscapesRoot(relativePath) {
			continue
		}
		change.Baseline.File = "HEAD:" + filepath.ToSlash(relativePath)
	}
}

func gitHistoryNotFound(target string, cause error) error {
	message := fmt.Sprintf("git history not found for %s", target)
	if cause != nil {
		return fmt.Errorf("%w: %s: %v", ErrGitHistoryUnavailable, message, cause)
	}
	return fmt.Errorf("%w: %s", ErrGitHistoryUnavailable, message)
}

func extractGitArchive(content []byte, destination string) (bool, error) {
	reader := tar.NewReader(bytes.NewReader(content))
	extracted := false
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return extracted, nil
		}
		if err != nil {
			return false, err
		}
		path := filepath.Join(destination, filepath.FromSlash(header.Name))
		relativePath, err := filepath.Rel(destination, path)
		if err != nil || pathEscapesRoot(relativePath) {
			return false, fmt.Errorf("archive path %q escapes destination", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeXHeader, tar.TypeXGlobalHeader:
			continue
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o755); err != nil {
				return false, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return false, err
			}
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
			if err != nil {
				return false, err
			}
			_, copyErr := io.Copy(file, reader)
			closeErr := file.Close()
			if copyErr != nil {
				return false, copyErr
			}
			if closeErr != nil {
				return false, closeErr
			}
			extracted = true
		default:
			continue
		}
	}
}
