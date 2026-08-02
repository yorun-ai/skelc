package fileutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

// Replacement describes one file to atomically create or replace.
type Replacement struct {
	Path    string
	Content []byte
	Mode    fs.FileMode
}

type _PreparedReplacement struct {
	path       string
	stagedPath string
	backupPath string
	existed    bool
}

var syncReplacementDirectory = syncDirectory

// Replace atomically replaces one file.
func Replace(replacement Replacement) error {
	prepared, err := prepare(replacement)
	if err != nil {
		return err
	}
	defer removePrepared(prepared)
	if err := os.Rename(prepared.stagedPath, prepared.path); err != nil {
		return fmt.Errorf("replace file %s: %w", replacement.Path, err)
	}
	if err := syncDirectories([]_PreparedReplacement{prepared}); err != nil {
		rollbackErrors := rollback([]_PreparedReplacement{prepared}, os.Rename)
		if rollbackSyncErr := syncDirectories([]_PreparedReplacement{prepared}); rollbackSyncErr != nil {
			rollbackErrors = append(rollbackErrors, rollbackSyncErr)
		}
		return replacementFailure(err, rollbackErrors)
	}
	return nil
}

// ReplaceAll stages every replacement before atomically replacing the files.
// If a replacement fails, files already replaced by the batch are restored.
func ReplaceAll(replacements []Replacement) error {
	return replaceAll(replacements, os.Rename)
}

func replaceAll(replacements []Replacement, replace func(string, string) error) error {
	prepared := make([]_PreparedReplacement, 0, len(replacements))
	defer func() {
		for _, replacement := range prepared {
			removePrepared(replacement)
		}
	}()

	paths := map[string]bool{}
	for _, replacement := range replacements {
		cleaned := filepath.Clean(replacement.Path)
		if paths[cleaned] {
			return fmt.Errorf("cannot replace file %s more than once", replacement.Path)
		}
		paths[cleaned] = true
		item, err := prepare(replacement)
		if err != nil {
			return err
		}
		prepared = append(prepared, item)
	}

	for index, replacement := range prepared {
		if err := replace(replacement.stagedPath, replacement.path); err != nil {
			commitErr := fmt.Errorf("replace file %s: %w", replacement.path, err)
			rollbackErrors := rollback(prepared[:index], replace)
			if syncErr := syncDirectories(prepared[:index]); syncErr != nil {
				rollbackErrors = append(rollbackErrors, syncErr)
			}
			return replacementFailure(commitErr, rollbackErrors)
		}
	}
	if err := syncDirectories(prepared); err != nil {
		rollbackErrors := rollback(prepared, replace)
		if rollbackSyncErr := syncDirectories(prepared); rollbackSyncErr != nil {
			rollbackErrors = append(rollbackErrors, rollbackSyncErr)
		}
		return replacementFailure(err, rollbackErrors)
	}
	return nil
}

func prepare(replacement Replacement) (_PreparedReplacement, error) {
	if err := validateTarget(replacement.Path); err != nil {
		return _PreparedReplacement{}, err
	}
	info, err := os.Lstat(replacement.Path)
	if errors.Is(err, os.ErrNotExist) {
		stagedPath, stageErr := stage(replacement, "")
		return _PreparedReplacement{path: replacement.Path, stagedPath: stagedPath}, stageErr
	}
	if err != nil {
		return _PreparedReplacement{}, fmt.Errorf("inspect file %s: %w", replacement.Path, err)
	}
	stagedPath, err := stage(replacement, replacement.Path)
	if err != nil {
		return _PreparedReplacement{}, err
	}
	prepared := _PreparedReplacement{path: replacement.Path, stagedPath: stagedPath}
	content, err := os.ReadFile(replacement.Path)
	if err != nil {
		_ = os.Remove(stagedPath)
		return _PreparedReplacement{}, fmt.Errorf("read file %s: %w", replacement.Path, err)
	}
	backupPath, err := stage(Replacement{Path: replacement.Path, Content: content, Mode: info.Mode()}, replacement.Path)
	if err != nil {
		_ = os.Remove(stagedPath)
		return _PreparedReplacement{}, fmt.Errorf("back up file %s: %w", replacement.Path, err)
	}
	prepared.backupPath = backupPath
	prepared.existed = true
	return prepared, nil
}

func removePrepared(replacement _PreparedReplacement) {
	_ = os.Remove(replacement.stagedPath)
	_ = os.Remove(replacement.backupPath)
}

func replacementFailure(commitErr error, rollbackErrors []error) error {
	if len(rollbackErrors) == 0 {
		return commitErr
	}
	return errors.Join(commitErr, fmt.Errorf("roll back file replacements: %w", errors.Join(rollbackErrors...)))
}

func rollback(replacements []_PreparedReplacement, replace func(string, string) error) []error {
	errs := []error{}
	for index := len(replacements) - 1; index >= 0; index-- {
		replacement := replacements[index]
		if !replacement.existed {
			if err := os.Remove(replacement.path); err != nil && !errors.Is(err, os.ErrNotExist) {
				errs = append(errs, fmt.Errorf("remove new file %s: %w", replacement.path, err))
			}
			continue
		}
		if err := replace(replacement.backupPath, replacement.path); err != nil {
			errs = append(errs, fmt.Errorf("restore file %s: %w", replacement.path, err))
		}
	}
	return errs
}

func validateTarget(path string) error {
	if path == "" {
		return errors.New("replacement path is empty")
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect file %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("replacement target %s is not a regular file", path)
	}
	return nil
}

func stage(replacement Replacement, metadataPath string) (string, error) {
	temporary, err := os.CreateTemp(filepath.Dir(replacement.Path), ".skelc-file-*")
	if err != nil {
		return "", fmt.Errorf("create staged file for %s: %w", replacement.Path, err)
	}
	temporaryPath := temporary.Name()
	cleanup := func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}
	if _, err := temporary.Write(replacement.Content); err != nil {
		cleanup()
		return "", fmt.Errorf("write staged file for %s: %w", replacement.Path, err)
	}
	if metadataPath != "" {
		if err := copyFileMetadata(metadataPath, temporary, replacement.Mode); err != nil {
			cleanup()
			return "", fmt.Errorf("copy file metadata for %s: %w", replacement.Path, err)
		}
	} else if err := temporary.Chmod(replacement.Mode); err != nil {
		cleanup()
		return "", fmt.Errorf("set staged file mode for %s: %w", replacement.Path, err)
	}
	if err := temporary.Sync(); err != nil {
		cleanup()
		return "", fmt.Errorf("sync staged file for %s: %w", replacement.Path, err)
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return "", fmt.Errorf("close staged file for %s: %w", replacement.Path, err)
	}
	return temporaryPath, nil
}

func syncDirectories(replacements []_PreparedReplacement) error {
	directories := map[string]bool{}
	for _, replacement := range replacements {
		if replacement.path != "" {
			directories[filepath.Dir(replacement.path)] = true
		}
	}
	for _, directory := range sortedPaths(directories) {
		if err := syncReplacementDirectory(directory); err != nil {
			return fmt.Errorf("sync replacement directory %s: %w", directory, err)
		}
	}
	return nil
}

func sortedPaths(paths map[string]bool) []string {
	values := make([]string, 0, len(paths))
	for path := range paths {
		values = append(values, path)
	}
	slices.Sort(values)
	return values
}
