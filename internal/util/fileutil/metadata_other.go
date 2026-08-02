//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package fileutil

import (
	"io/fs"
	"os"
)

func copyFileMetadata(_ string, target *os.File, mode fs.FileMode) error { return target.Chmod(mode) }
