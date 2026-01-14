package rotate

import (
	"bufio"
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

func ForceRotate(path string, policy Policy) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if err := rotateFile(path, info, policy); err != nil {
		return false, err
	}

	return true, nil
}

func RotateByTimestampCut(path string, policy Policy, layout string, cutoff time.Time) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if layout == "" {
		return ForceRotate(path, policy)
	}

	cutOffset, err := findCutOffset(path, layout, cutoff)
	if err != nil {
		return false, err
	}
	if cutOffset <= 0 {
		return false, nil
	}
	if cutOffset >= info.Size() {
		return ForceRotate(path, policy)
	}

	return rotateByOffset(path, info, policy, cutOffset)
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

func rotateByOffset(path string, info os.FileInfo, policy Policy, cutOffset int64) (bool, error) {
	if policy.Keep < 1 {
		return false, nil
	}
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	rotateTmp, err := os.CreateTemp(dir, base+".rotate.*")
	if err != nil {
		return false, err
	}
	defer os.Remove(rotateTmp.Name())

	remainTmp, err := os.CreateTemp(dir, base+".remain.*")
	if err != nil {
		rotateTmp.Close()
		return false, err
	}
	defer os.Remove(remainTmp.Name())

	src, err := os.Open(path)
	if err != nil {
		rotateTmp.Close()
		remainTmp.Close()
		return false, err
	}
	defer src.Close()

	if _, err := io.CopyN(rotateTmp, src, cutOffset); err != nil && err != io.EOF {
		rotateTmp.Close()
		remainTmp.Close()
		return false, err
	}
	if _, err := io.Copy(remainTmp, src); err != nil {
		rotateTmp.Close()
		remainTmp.Close()
		return false, err
	}

	if err := rotateTmp.Close(); err != nil {
		remainTmp.Close()
		return false, err
	}
	if err := remainTmp.Close(); err != nil {
		return false, err
	}

	if err := os.Chmod(rotateTmp.Name(), info.Mode().Perm()); err != nil {
		return false, err
	}
	if err := os.Chmod(remainTmp.Name(), info.Mode().Perm()); err != nil {
		return false, err
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		_ = os.Chown(rotateTmp.Name(), int(stat.Uid), int(stat.Gid))
		_ = os.Chown(remainTmp.Name(), int(stat.Uid), int(stat.Gid))
	}

	if err := shiftRotated(path, policy); err != nil {
		return false, err
	}

	if err := moveFile(rotateTmp.Name(), fmt.Sprintf("%s.1", path)); err != nil {
		return false, err
	}
	if err := moveFile(remainTmp.Name(), path); err != nil {
		return false, err
	}

	if policy.Compress {
		if err := compressRotated(path, policy); err != nil {
			return false, err
		}
	}

	return true, nil
}

func findCutOffset(path string, layout string, cutoff time.Time) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var offset int64
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			if ts, ok := parseLineTimestamp(line, layout); ok {
				if !ts.Before(cutoff) {
					return offset, nil
				}
			}
			offset += int64(len(line))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	info, err := file.Stat()
	if err != nil {
		return offset, nil
	}
	return info.Size(), nil
}

func parseLineTimestamp(line string, layout string) (time.Time, bool) {
	if len(line) < len(layout) {
		return time.Time{}, false
	}
	prefix := line[:len(layout)]
	ts, err := time.ParseInLocation(layout, prefix, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return ts, true
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
