package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type InputPolicy struct {
	MaxSizeMB   int   `json:"max_size_mb,omitempty"`
	Keep        int   `json:"keep,omitempty"`
	Compress    *bool `json:"compress,omitempty"`
	MaxAgeDays  int   `json:"max_age_days,omitempty"`
	ShipEnabled *bool `json:"ship_enabled,omitempty"`
}

type LogInput struct {
	Package string      `json:"package"`
	LogID   string      `json:"log_id"`
	Path    string      `json:"path"`
	Policy  InputPolicy `json:"policy"`
	Source  string      `json:"-"`
}

type InputFile struct {
	Inputs []LogInput `json:"inputs"`
}

func LoadInputs(dir string) ([]LogInput, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var inputs []LogInput
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		path := filepath.Join(dir, name)
		fileInputs, err := parseInputFile(path)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, fileInputs...)
	}

	return inputs, nil
}

func parseInputFile(path string) ([]LogInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var inputs []LogInput
	if err := json.Unmarshal(data, &inputs); err == nil {
		return withSource(inputs, path), nil
	}

	var single LogInput
	if err := json.Unmarshal(data, &single); err == nil {
		return withSource([]LogInput{single}, path), nil
	}

	var wrapped InputFile
	if err := json.Unmarshal(data, &wrapped); err == nil {
		return withSource(wrapped.Inputs, path), nil
	}

	return nil, fmt.Errorf("arquivo de input invalido: %s", path)
}

func withSource(inputs []LogInput, path string) []LogInput {
	for i := range inputs {
		inputs[i].Source = path
	}
	return inputs
}
