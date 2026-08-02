//go:build darwin || freebsd || linux || netbsd

package fileutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestReplaceAllPreservesOwnershipAndExtendedAttributes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "domain.skel")
	writeTestFile(t, path, "old")
	const attribute = "user.skelc-test"
	if err := unix.Setxattr(path, attribute, []byte("metadata"), 0); err != nil {
		if errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EPERM) {
			t.Skipf("extended attributes unavailable: %v", err)
		}
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	var beforeStat unix.Stat_t
	if err := unix.Stat(path, &beforeStat); err != nil {
		t.Fatal(err)
	}

	if err := ReplaceAll([]Replacement{{Path: path, Content: []byte("new"), Mode: before.Mode()}}); err != nil {
		t.Fatal(err)
	}
	var afterStat unix.Stat_t
	if err := unix.Stat(path, &afterStat); err != nil {
		t.Fatal(err)
	}
	if beforeStat.Uid != afterStat.Uid || beforeStat.Gid != afterStat.Gid {
		t.Fatalf("ownership changed: before=%d:%d after=%d:%d", beforeStat.Uid, beforeStat.Gid, afterStat.Uid, afterStat.Gid)
	}
	value := make([]byte, 32)
	size, err := unix.Getxattr(path, attribute, value)
	if err != nil {
		t.Fatal(err)
	}
	if string(value[:size]) != "metadata" {
		t.Fatalf("extended attribute = %q", value[:size])
	}
}
