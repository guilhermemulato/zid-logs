package rotate

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type Policy struct {
	MaxSizeMB  int
	Keep       int
	Compress   bool
	MaxAgeDays int
}

func RotateIfNeeded(path string, policy Policy) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	if !shouldRotate(info, policy) {
		return false, nil
	}

	if err := rotateFile(path, info, policy); err != nil {
		return false, err
	}

	return true, nil
}

func shouldRotate(info os.FileInfo, policy Policy) bool {
	if policy.MaxSizeMB > 0 {
		if info.Size() >= int64(policy.MaxSizeMB)*1024*1024 {
			return true
		}
	}
	if policy.MaxAgeDays > 0 {
		maxAge := time.Duration(policy.MaxAgeDays) * 24 * time.Hour
		if time.Since(info.ModTime()) >= maxAge {
			return true
		}
	}
	return false
}

func rotateFile(path string, info os.FileInfo, policy Policy) error {
	if policy.Keep < 1 {
		return nil
	}

	if err := shiftRotated(path, policy); err != nil {
		return err
	}

	if err := moveFile(path, fmt.Sprintf("%s.1", path)); err != nil {
		return err
	}

	if err := recreateFile(path, info); err != nil {
		return err
	}

	if policy.Compress {
		if err := compressRotated(path, policy); err != nil {
			return err
		}
	}

	return nil
}

func shiftRotated(path string, policy Policy) error {
	for i := policy.Keep - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", path, i)
		dst := fmt.Sprintf("%s.%d", path, i+1)

		if policy.Compress && i >= 2 {
			if err := moveFile(src+".gz", dst+".gz"); err != nil {
				return err
			}
		}

		if err := moveFile(src, dst); err != nil {
			return err
		}
	}

	return nil
}

func moveFile(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	_ = os.Remove(dst)
	return os.Rename(src, dst)
}

func recreateFile(path string, info os.FileInfo) error {
	perm := info.Mode().Perm()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		_ = os.Chown(path, int(stat.Uid), int(stat.Gid))
	}

	return nil
}

func compressRotated(path string, policy Policy) error {
	for i := 2; i <= policy.Keep; i++ {
		src := fmt.Sprintf("%s.%d", path, i)
		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		if err := compressFile(src); err != nil {
			return err
		}
	}
	return nil
}

func compressFile(path string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}

	dstPath := path + ".gz"
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer dst.Close()

	zw := gzip.NewWriter(dst)
	if _, err := io.Copy(zw, src); err != nil {
		_ = zw.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}

	return os.Remove(path)
}
