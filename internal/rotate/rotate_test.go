package rotate

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRotateKeepAndCompress(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")

	payload := bytes.Repeat([]byte("x"), 2*1024*1024)
	if err := os.WriteFile(path, payload, 0644); err != nil {
		t.Fatalf("write base: %v", err)
	}

	if err := os.WriteFile(path+".1", []byte("old"), 0644); err != nil {
		t.Fatalf("write .1: %v", err)
	}

	policy := Policy{MaxSizeMB: 1, Keep: 2, Compress: true}
	rotated, err := RotateIfNeeded(path, policy)
	if err != nil {
		t.Fatalf("RotateIfNeeded error: %v", err)
	}
	if !rotated {
		t.Fatalf("expected rotation")
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected new file, err: %v", err)
	}

	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("expected .1 file: %v", err)
	}

	if _, err := os.Stat(path + ".2.gz"); err != nil {
		t.Fatalf("expected .2.gz file: %v", err)
	}
	if _, err := os.Stat(path + ".2"); err == nil {
		t.Fatalf("expected .2 removed after gzip")
	}
}
