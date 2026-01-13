package state

import (
	"path/filepath"
	"testing"
)

func TestStateSaveAndGet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.db")

	st, err := Open(path)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer st.Close()

	cp := Checkpoint{
		Package:    "zid-proxy",
		LogID:      "main",
		Path:       "/var/log/zid.log",
		LastOffset: 123,
		LastSentAt: 456,
		LastError:  "",
	}

	if err := st.SaveCheckpoint(cp); err != nil {
		t.Fatalf("SaveCheckpoint error: %v", err)
	}

	got, ok, err := st.GetCheckpoint(cp.Package, cp.LogID, cp.Path)
	if err != nil {
		t.Fatalf("GetCheckpoint error: %v", err)
	}
	if !ok {
		t.Fatalf("expected checkpoint")
	}
	if got.LastOffset != cp.LastOffset {
		t.Fatalf("expected offset %d, got %d", cp.LastOffset, got.LastOffset)
	}
}
