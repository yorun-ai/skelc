//go:build aix || dragonfly || openbsd || solaris

package fileutil

import (
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
	return nil
}
