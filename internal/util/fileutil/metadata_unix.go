//go:build darwin || freebsd || linux || netbsd

package fileutil

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

func copyFileMetadata(sourcePath string, target *os.File, mode fs.FileMode) error {
	var sourceStat unix.Stat_t
	if err := unix.Stat(sourcePath, &sourceStat); err != nil {
		return fmt.Errorf("stat source metadata: %w", err)
	}
	var targetStat unix.Stat_t
	if err := unix.Fstat(int(target.Fd()), &targetStat); err != nil {
		return fmt.Errorf("stat staged metadata: %w", err)
	}
	if sourceStat.Uid != targetStat.Uid || sourceStat.Gid != targetStat.Gid {
		if err := unix.Fchown(int(target.Fd()), int(sourceStat.Uid), int(sourceStat.Gid)); err != nil {
			return fmt.Errorf("preserve ownership: %w", err)
		}
	}
	if err := target.Chmod(mode); err != nil {
		return fmt.Errorf("preserve mode: %w", err)
	}
	if err := copyExtendedAttributes(sourcePath, int(target.Fd())); err != nil {
		return err
	}
	return nil
}

func copyExtendedAttributes(sourcePath string, targetFD int) error {
	names, err := listExtendedAttributes(sourcePath)
	if err != nil {
		return err
	}
	for _, name := range names {
		value, err := getExtendedAttribute(sourcePath, name)
		if err != nil {
			return fmt.Errorf("read extended attribute %s: %w", name, err)
		}
		if err := unix.Fsetxattr(targetFD, name, value, 0); err != nil {
			return fmt.Errorf("preserve extended attribute %s: %w", name, err)
		}
	}
	return nil
}

func listExtendedAttributes(path string) ([]string, error) {
	size, err := unix.Listxattr(path, nil)
	if errors.Is(err, unix.ENOTSUP) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list extended attributes: %w", err)
	}
	if size == 0 {
		return nil, nil
	}
	buffer := make([]byte, size)
	size, err = unix.Listxattr(path, buffer)
	if err != nil {
		return nil, fmt.Errorf("list extended attributes: %w", err)
	}
	names := make([]string, 0)
	for _, name := range bytes.Split(buffer[:size], []byte{0}) {
		if len(name) > 0 {
			names = append(names, string(name))
		}
	}
	return names, nil
}

func getExtendedAttribute(path, name string) ([]byte, error) {
	size, err := unix.Getxattr(path, name, nil)
	if err != nil {
		return nil, err
	}
	value := make([]byte, size)
	if size == 0 {
		return value, nil
	}
	size, err = unix.Getxattr(path, name, value)
	if err != nil {
		return nil, err
	}
	return value[:size], nil
}
