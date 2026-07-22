package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
)

type _OutputSnapshot struct {
	path       string
	backupPath string
	mode       fs.FileMode
	existed    bool
}

type _OutputCommit struct {
	output        *ManagedOutput
	manifest      _OutputManifest
	previous      _OutputManifest
	manifestBytes []byte
	snapshots     []_OutputSnapshot
	targetExisted bool
}

func (o *ManagedOutput) Commit() error {
	if o == nil || o.stageRoot == "" {
		return errors.New("output staging transaction is closed")
	}
	defer o.Abort()
	commit, err := o.prepareCommit()
	if err != nil {
		return err
	}
	if err := commit.apply(); err != nil {
		if rollbackErr := commit.rollback(); rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("roll back generated output: %w", rollbackErr))
		}
		return err
	}
	return nil
}

// CommitManagedOutputs commits several staged outputs as one transaction. If
// any target fails, every target already modified by the batch is restored.
func CommitManagedOutputs(outputs []*ManagedOutput) error {
	commits := make([]*_OutputCommit, 0, len(outputs))
	targets := map[string]bool{}
	defer func() {
		for _, output := range outputs {
			output.Abort()
		}
	}()
	for _, output := range outputs {
		if output == nil || output.stageRoot == "" {
			return errors.New("output staging transaction is closed")
		}
		if targets[output.targetDir] {
			return fmt.Errorf("cannot commit output directory %s more than once", output.targetDir)
		}
		targets[output.targetDir] = true
		commit, err := output.prepareCommit()
		if err != nil {
			rollbackErrors := []error{}
			for index := len(commits) - 1; index >= 0; index-- {
				if rollbackErr := commits[index].rollback(); rollbackErr != nil {
					rollbackErrors = append(rollbackErrors, rollbackErr)
				}
			}
			if len(rollbackErrors) > 0 {
				return errors.Join(err, fmt.Errorf("clean up prepared outputs: %w", errors.Join(rollbackErrors...)))
			}
			return err
		}
		commits = append(commits, commit)
	}
	for index, commit := range commits {
		if err := commit.apply(); err != nil {
			rollbackErrors := []error{}
			for rollbackIndex := index; rollbackIndex >= 0; rollbackIndex-- {
				if rollbackErr := commits[rollbackIndex].rollback(); rollbackErr != nil {
					rollbackErrors = append(rollbackErrors, rollbackErr)
				}
			}
			if len(rollbackErrors) > 0 {
				return errors.Join(err, fmt.Errorf("roll back generated outputs: %w", errors.Join(rollbackErrors...)))
			}
			return err
		}
	}
	return nil
}

func (o *ManagedOutput) prepareCommit() (*_OutputCommit, error) {
	manifest, err := buildOutputManifest(o.stageDir)
	if err != nil {
		return nil, err
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode output manifest: %w", err)
	}
	manifestBytes = append(manifestBytes, '\n')

	targetExisted := true
	if _, err := os.Lstat(o.targetDir); errors.Is(err, os.ErrNotExist) {
		targetExisted = false
	} else if err != nil {
		return nil, fmt.Errorf("inspect output directory %s: %w", o.targetDir, err)
	}
	if err := os.MkdirAll(o.targetDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory %s: %w", o.targetDir, err)
	}
	cleanupNewTarget := func() {
		if !targetExisted {
			_ = os.Remove(o.targetDir)
		}
	}
	if err := rejectOutputSymlink(o.targetDir, ""); err != nil {
		cleanupNewTarget()
		return nil, err
	}
	if err := rejectOutputSymlink(o.targetDir, outputManifestName); err != nil {
		cleanupNewTarget()
		return nil, err
	}
	previous, err := readOutputManifest(o.targetDir)
	if err != nil {
		cleanupNewTarget()
		return nil, err
	}
	paths := map[string]bool{outputManifestName: true}
	for _, file := range manifest.Files {
		paths[file.Path] = true
		if err := rejectOutputSymlink(o.targetDir, filepath.FromSlash(file.Path)); err != nil {
			cleanupNewTarget()
			return nil, err
		}
	}
	for _, file := range previous.Files {
		paths[file.Path] = true
		if err := rejectOutputSymlink(o.targetDir, filepath.FromSlash(file.Path)); err != nil {
			cleanupNewTarget()
			return nil, err
		}
	}
	snapshots, err := snapshotOutputFiles(o.targetDir, o.stageRoot, paths)
	if err != nil {
		cleanupNewTarget()
		return nil, err
	}
	return &_OutputCommit{
		output: o, manifest: manifest, previous: previous, manifestBytes: manifestBytes,
		snapshots: snapshots, targetExisted: targetExisted,
	}, nil
}

func (commit *_OutputCommit) apply() error {
	for _, file := range commit.manifest.Files {
		if err := commit.output.commitOutputFile(file.Path); err != nil {
			return err
		}
	}
	if err := removeStaleOutputFiles(commit.output.targetDir, commit.previous, commit.manifest); err != nil {
		return err
	}
	manifestPath := filepath.Join(commit.output.targetDir, outputManifestName)
	return commit.output.writeFile(manifestPath, commit.manifestBytes, 0o644)
}

func (commit *_OutputCommit) rollback() error {
	errs := []error{}
	for index := len(commit.snapshots) - 1; index >= 0; index-- {
		snapshot := commit.snapshots[index]
		target := filepath.Join(commit.output.targetDir, filepath.FromSlash(snapshot.path))
		if !snapshot.existed {
			if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
				errs = append(errs, fmt.Errorf("remove new output %s: %w", target, err))
				continue
			}
			removeEmptyOutputParents(filepath.Dir(target), commit.output.targetDir)
			continue
		}
		content, err := os.ReadFile(snapshot.backupPath)
		if err != nil {
			errs = append(errs, fmt.Errorf("read output backup %s: %w", snapshot.backupPath, err))
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			errs = append(errs, fmt.Errorf("restore output parent %s: %w", target, err))
			continue
		}
		if err := atomicWriteFile(target, content, snapshot.mode); err != nil {
			errs = append(errs, fmt.Errorf("restore output %s: %w", target, err))
		}
	}
	if !commit.targetExisted {
		_ = os.Remove(commit.output.targetDir)
	}
	return errors.Join(errs...)
}

func snapshotOutputFiles(targetDir, stageRoot string, paths map[string]bool) ([]_OutputSnapshot, error) {
	relatives := slices.Sorted(maps.Keys(paths))
	backupRoot := filepath.Join(stageRoot, "backup")
	snapshots := make([]_OutputSnapshot, 0, len(relatives))
	for index, relative := range relatives {
		target := filepath.Join(targetDir, filepath.FromSlash(relative))
		info, err := os.Lstat(target)
		if errors.Is(err, os.ErrNotExist) {
			snapshots = append(snapshots, _OutputSnapshot{path: relative})
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect output %s: %w", target, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("output path %s is not a regular file", target)
		}
		content, err := os.ReadFile(target)
		if err != nil {
			return nil, fmt.Errorf("read output %s: %w", target, err)
		}
		backupPath := filepath.Join(backupRoot, fmt.Sprintf("%d", index))
		if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
			return nil, fmt.Errorf("create output backup directory: %w", err)
		}
		if err := os.WriteFile(backupPath, content, info.Mode().Perm()); err != nil {
			return nil, fmt.Errorf("back up output %s: %w", target, err)
		}
		snapshots = append(snapshots, _OutputSnapshot{
			path: relative, backupPath: backupPath, mode: info.Mode().Perm(), existed: true,
		})
	}
	return snapshots, nil
}
