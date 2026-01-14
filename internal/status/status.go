package status

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"zid-logs/internal/config"
	"zid-logs/internal/registry"
	"zid-logs/internal/state"
)

type InputStatus struct {
	Package        string `json:"package"`
	LogID          string `json:"log_id"`
	Path           string `json:"path"`
	Source         string `json:"source"`
	FileSize       int64  `json:"file_size"`
	Backlog        int64  `json:"backlog"`
	LastOffset     int64  `json:"last_offset"`
	LastSentAt     int64  `json:"last_sent_at"`
	LastError      string `json:"last_error"`
	LastAttemptAt  int64  `json:"last_attempt_at"`
	LastStatusCode int    `json:"last_status_code"`
	LastBytesSent  int64  `json:"last_bytes_sent"`
	LastDurationMs int64  `json:"last_duration_ms"`
	LastRotateAt   int64  `json:"last_rotate_at"`
	IdentityDev    uint64 `json:"dev"`
	IdentityIno    uint64 `json:"inode"`
}

type Status struct {
	GeneratedAt       int64         `json:"generated_at"`
	Inputs            []InputStatus `json:"inputs"`
	LastErrorGlobal   string        `json:"last_error_global"`
	TotalInputs       int           `json:"total_inputs"`
	TotalBacklog      int64         `json:"total_backlog"`
	LastSentAt        int64         `json:"last_sent_at"`
	LastAttemptAt     int64         `json:"last_attempt_at"`
	LastRotateAt      int64         `json:"last_rotate_at"`
	NextRotateAt      int64         `json:"next_rotate_at"`
	ShipIntervalHours int           `json:"ship_interval_hours"`
	RotateAt          string        `json:"rotate_at"`
}

func Build(cfg config.Config, inputs []registry.LogInput, st *state.State, lastError string) Status {
	status := Status{
		GeneratedAt:       time.Now().Unix(),
		LastErrorGlobal:   lastError,
		TotalInputs:       len(inputs),
		ShipIntervalHours: cfg.ShipIntervalHours,
		RotateAt:          cfg.RotateAt,
	}

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
				item.LastAttemptAt = cp.LastAttemptAt
				item.LastStatusCode = cp.LastStatusCode
				item.LastBytesSent = cp.LastBytesSent
				item.LastDurationMs = cp.LastDurationMs
				item.LastRotateAt = cp.LastRotateAt
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

		status.TotalBacklog += item.Backlog
		if item.LastSentAt > status.LastSentAt {
			status.LastSentAt = item.LastSentAt
		}
		if item.LastAttemptAt > status.LastAttemptAt {
			status.LastAttemptAt = item.LastAttemptAt
		}
		if item.LastRotateAt > status.LastRotateAt {
			status.LastRotateAt = item.LastRotateAt
		}
		if status.LastErrorGlobal == "" && item.LastError != "" {
			status.LastErrorGlobal = item.LastError
		}

		status.Inputs = append(status.Inputs, item)
	}

	if cfg.RotateAt != "" {
		if next, err := nextRotateTime(time.Now(), cfg.RotateAt); err == nil {
			status.NextRotateAt = next.Unix()
		}
	}

	return status
}

func nextRotateTime(now time.Time, rotateAt string) (time.Time, error) {
	hour, minute, err := parseRotateAt(rotateAt)
	if err != nil {
		return time.Time{}, err
	}
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next, nil
}

func parseRotateAt(value string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) == 0 || len(parts) > 2 {
		return 0, 0, fmt.Errorf("rotate_at invalido")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("hora invalida")
	}
	minute := 0
	if len(parts) == 2 {
		minute, err = strconv.Atoi(parts[1])
		if err != nil || minute < 0 || minute > 59 {
			return 0, 0, fmt.Errorf("minuto invalido")
		}
	}
	return hour, minute, nil
}
