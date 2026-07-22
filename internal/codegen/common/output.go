package common

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const outputManifestName = ".skelc-manifest.json"

type _OutputManifest struct {
	Version int                   `json:"version"`
	Files   []_OutputManifestFile `json:"files"`
}

type _OutputManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// ManagedOutput stages a complete generator run and commits only files owned
// by skelc. Files not listed in the previous manifest are never removed.
type ManagedOutput struct {
	targetDir string
	stageRoot string
	stageDir  string
}

func NewManagedOutput(targetDir string) (*ManagedOutput, error) {
	targetDir = filepath.Clean(targetDir)
	parent := filepath.Dir(targetDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return nil, fmt.Errorf("create output parent %s: %w", parent, err)
	}
	stageRoot, err := os.MkdirTemp(parent, "."+filepath.Base(targetDir)+".skelc-stage-")
	if err != nil {
		return nil, fmt.Errorf("create output staging directory for %s: %w", targetDir, err)
	}
	stageDir := filepath.Join(stageRoot, filepath.Base(targetDir))
	if err := os.Mkdir(stageDir, 0o755); err != nil {
		_ = os.RemoveAll(stageRoot)
		return nil, fmt.Errorf("create output staging directory %s: %w", stageDir, err)
	}
	return &ManagedOutput{targetDir: targetDir, stageRoot: stageRoot, stageDir: stageDir}, nil
}

func (o *ManagedOutput) StageDir() string {
	return o.stageDir
}

func (o *ManagedOutput) Abort() {
	if o == nil || o.stageRoot == "" {
		return
	}
	_ = os.RemoveAll(o.stageRoot)
	o.stageRoot = ""
}

func (o *ManagedOutput) Commit() error {
	if o == nil || o.stageRoot == "" {
		return errors.New("output staging transaction is closed")
	}
	defer o.Abort()

	manifest, err := buildOutputManifest(o.stageDir)
	if err != nil {
		return err
	}
	previous, err := readOutputManifest(o.targetDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(o.targetDir, 0o755); err != nil {
		return fmt.Errorf("create output directory %s: %w", o.targetDir, err)
	}
	if err := rejectOutputSymlink(o.targetDir, ""); err != nil {
		return err
	}
	for _, file := range manifest.Files {
		if err := rejectOutputSymlink(o.targetDir, filepath.FromSlash(file.Path)); err != nil {
			return err
		}
	}
	for _, file := range previous.Files {
		if err := rejectOutputSymlink(o.targetDir, filepath.FromSlash(file.Path)); err != nil {
			return err
		}
	}
	for _, file := range manifest.Files {
		if err := commitOutputFile(o.stageDir, o.targetDir, file.Path); err != nil {
			return err
		}
	}
	if err := removeStaleOutputFiles(o.targetDir, previous, manifest); err != nil {
		return err
	}
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode output manifest: %w", err)
	}
	content = append(content, '\n')
	return atomicWriteFile(filepath.Join(o.targetDir, outputManifestName), content, 0o644)
}

func buildOutputManifest(stageDir string) (_OutputManifest, error) {
	manifest := _OutputManifest{Version: 1, Files: []_OutputManifestFile{}}
	err := filepath.WalkDir(stageDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("generated output contains symlink %s", path)
		}
		relative, err := filepath.Rel(stageDir, path)
		if err != nil {
			return err
		}
		relative, err = cleanManifestPath(relative)
		if err != nil {
			return err
		}
		hash, err := fileSHA256(path)
		if err != nil {
			return err
		}
		manifest.Files = append(manifest.Files, _OutputManifestFile{Path: filepath.ToSlash(relative), SHA256: hash})
		return nil
	})
	if err != nil {
		return _OutputManifest{}, fmt.Errorf("inspect staged output %s: %w", stageDir, err)
	}
	slices.SortFunc(manifest.Files, func(left, right _OutputManifestFile) int {
		return strings.Compare(left.Path, right.Path)
	})
	return manifest, nil
}

func readOutputManifest(targetDir string) (_OutputManifest, error) {
	path := filepath.Join(targetDir, outputManifestName)
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return _OutputManifest{Version: 1}, nil
	}
	if err != nil {
		return _OutputManifest{}, fmt.Errorf("read output manifest %s: %w", path, err)
	}
	var manifest _OutputManifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return _OutputManifest{}, fmt.Errorf("decode output manifest %s: %w", path, err)
	}
	if manifest.Version != 1 {
		return _OutputManifest{}, fmt.Errorf("unsupported output manifest version %d in %s", manifest.Version, path)
	}
	for index := range manifest.Files {
		cleaned, err := cleanManifestPath(filepath.FromSlash(manifest.Files[index].Path))
		if err != nil {
			return _OutputManifest{}, fmt.Errorf("invalid output manifest %s: %w", path, err)
		}
		manifest.Files[index].Path = filepath.ToSlash(cleaned)
	}
	return manifest, nil
}

func commitOutputFile(stageDir, targetDir, relative string) error {
	relative, err := cleanManifestPath(filepath.FromSlash(relative))
	if err != nil {
		return err
	}
	source := filepath.Join(stageDir, relative)
	target := filepath.Join(targetDir, relative)
	if err := rejectOutputSymlink(targetDir, relative); err != nil {
		return err
	}
	content, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read staged output %s: %w", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create output directory for %s: %w", target, err)
	}
	if err := atomicWriteFile(target, content, 0o644); err != nil {
		return fmt.Errorf("commit generated output %s: %w", target, err)
	}
	return nil
}

func removeStaleOutputFiles(targetDir string, previous, current _OutputManifest) error {
	currentPaths := make(map[string]bool, len(current.Files))
	for _, file := range current.Files {
		currentPaths[file.Path] = true
	}
	for _, file := range previous.Files {
		if currentPaths[file.Path] {
			continue
		}
		relative, err := cleanManifestPath(filepath.FromSlash(file.Path))
		if err != nil {
			return err
		}
		path := filepath.Join(targetDir, relative)
		if err := rejectOutputSymlink(targetDir, relative); err != nil {
			return err
		}
		hash, err := fileSHA256(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect stale generated output %s: %w", path, err)
		}
		if hash != file.SHA256 {
			continue
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove stale generated output %s: %w", path, err)
		}
		removeEmptyOutputParents(filepath.Dir(path), targetDir)
	}
	return nil
}

func rejectOutputSymlink(root, relative string) error {
	path := root
	parts := []string{}
	if relative != "" {
		parts = strings.Split(filepath.Clean(relative), string(filepath.Separator))
	}
	for index := -1; index < len(parts); index++ {
		if index >= 0 {
			path = filepath.Join(path, parts[index])
		}
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect output path %s: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output path contains symlink %s", path)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("output parent %s is not a directory", path)
		}
	}
	return nil
}

func removeEmptyOutputParents(dir, root string) {
	for dir != root && strings.HasPrefix(dir, root+string(filepath.Separator)) {
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

func cleanManifestPath(path string) (string, error) {
	cleaned := filepath.Clean(path)
	if cleaned == "." || filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe generated output path %q", path)
	}
	if filepath.Base(cleaned) == outputManifestName {
		return "", fmt.Errorf("generated output cannot use reserved path %q", path)
	}
	return cleaned, nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func atomicWriteFile(path string, content []byte, mode fs.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".skelc-write-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
