package status

import (
	"os"
	"time"

	"zid-logs/internal/registry"
	"zid-logs/internal/state"
)

type InputStatus struct {
	Package     string `json:"package"`
	LogID       string `json:"log_id"`
	Path        string `json:"path"`
	Source      string `json:"source"`
	FileSize    int64  `json:"file_size"`
	Backlog     int64  `json:"backlog"`
	LastOffset  int64  `json:"last_offset"`
	LastSentAt  int64  `json:"last_sent_at"`
	LastError   string `json:"last_error"`
	IdentityDev uint64 `json:"dev"`
	IdentityIno uint64 `json:"inode"`
}

type Status struct {
	GeneratedAt     int64         `json:"generated_at"`
	Inputs          []InputStatus `json:"inputs"`
	LastErrorGlobal string        `json:"last_error_global"`
}

func Build(inputs []registry.LogInput, st *state.State, lastError string) Status {
	status := Status{GeneratedAt: time.Now().Unix(), LastErrorGlobal: lastError}

	for _, input := range inputs {
		item := InputStatus{
			Package: input.Package,
			LogID:   input.LogID,
			Path:    input.Path,
			Source:  input.Source,
		}

		info, err := os.Stat(input.Path)
		if err == nil {
			item.FileSize = info.Size()
		}

		if st != nil {
			cp, ok, err := st.GetCheckpoint(input.Package, input.LogID, input.Path)
			if err == nil && ok {
				item.LastOffset = cp.LastOffset
				item.LastSentAt = cp.LastSentAt
				item.LastError = cp.LastError
				item.IdentityDev = cp.Identity.Dev
				item.IdentityIno = cp.Identity.Inode
			}
		}

		if item.FileSize > 0 {
			item.Backlog = item.FileSize - item.LastOffset
			if item.Backlog < 0 {
				item.Backlog = 0
			}
		}

		status.Inputs = append(status.Inputs, item)
	}

	return status
}
