package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadInputsVariants(t *testing.T) {
	dir := t.TempDir()

	arrayPath := filepath.Join(dir, "array.json")
	arrayJSON := `[{"package":"a","log_id":"l1","path":"/tmp/a.log"}]`
	if err := os.WriteFile(arrayPath, []byte(arrayJSON), 0644); err != nil {
		t.Fatalf("write array: %v", err)
	}

	singlePath := filepath.Join(dir, "single.json")
	singleJSON := `{"package":"b","log_id":"l2","path":"/tmp/b.log"}`
	if err := os.WriteFile(singlePath, []byte(singleJSON), 0644); err != nil {
		t.Fatalf("write single: %v", err)
	}

	wrappedPath := filepath.Join(dir, "wrapped.json")
	wrappedJSON := `{"inputs":[{"package":"c","log_id":"l3","path":"/tmp/c.log"}]}`
	if err := os.WriteFile(wrappedPath, []byte(wrappedJSON), 0644); err != nil {
		t.Fatalf("write wrapped: %v", err)
	}

	inputs, err := LoadInputs(dir)
	if err != nil {
		t.Fatalf("LoadInputs error: %v", err)
	}
	if len(inputs) != 3 {
		t.Fatalf("expected 3 inputs, got %d", len(inputs))
	}

	for _, input := range inputs {
		if input.Source == "" {
			t.Fatalf("expected Source set for input")
		}
	}
}
