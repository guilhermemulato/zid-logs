package shipper

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"zid-logs/internal/config"
	"zid-logs/internal/registry"
	"zid-logs/internal/state"
)

type Payload struct {
	DeviceID    string   `json:"device_id"`
	PFHostname  string   `json:"pf_hostname"`
	Package     string   `json:"package"`
	LogID       string   `json:"log_id"`
	Path        string   `json:"path"`
	Inode       uint64   `json:"inode"`
	OffsetStart int64    `json:"offset_start"`
	OffsetEnd   int64    `json:"offset_end"`
	SentAt      int64    `json:"sent_at"`
	Lines       []string `json:"lines,omitempty"`
	Raw         string   `json:"raw,omitempty"`
}

func ShipOnce(ctx context.Context, input registry.LogInput, cfg config.Config, st *state.State) (*state.Checkpoint, error) {
	if st == nil {
		return nil, errors.New("state nao inicializado")
	}
	if cfg.Endpoint == "" {
		return nil, errors.New("endpoint nao configurado")
	}

	info, err := os.Stat(input.Path)
	if err != nil {
		return nil, err
	}

	cp, exists, err := st.GetCheckpoint(input.Package, input.LogID, input.Path)
	if err != nil {
		return nil, err
	}
	if !exists {
		cp = state.Checkpoint{
			Package: input.Package,
			LogID:   input.LogID,
			Path:    input.Path,
		}
	}

	identity, err := fileIdentity(info)
	if err != nil {
		return nil, err
	}

	if cp.Identity.Inode != 0 && (cp.Identity.Inode != identity.Inode || cp.Identity.Dev != identity.Dev) {
		if cp.LastOffset > info.Size() {
			cp.LastOffset = 0
		}
		cp.Identity = identity
	}
	if cp.Identity.Inode == 0 {
		cp.Identity = identity
	}
	if cp.LastOffset > info.Size() {
		cp.LastOffset = 0
	}

	file, err := os.Open(input.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if _, err := file.Seek(cp.LastOffset, io.SeekStart); err != nil {
		return nil, err
	}

	maxBytes := cfg.MaxBytesPerShip
	if maxBytes <= 0 {
		maxBytes = 256 * 1024
	}

	buf := make([]byte, maxBytes)
	n, readErr := file.Read(buf)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return nil, readErr
	}
	if n == 0 {
		return &cp, nil
	}

	payload, err := buildPayload(input, cfg, cp, buf[:n])
	if err != nil {
		return nil, err
	}
	fillCheckpointWindow(&cp, input, payload)

	cp.LastAttemptAt = time.Now().Unix()
	cp.LastBytesSent = int64(n)
	statusCode, durationMs, err := postPayload(ctx, cfg, payload)
	cp.LastStatusCode = statusCode
	cp.LastDurationMs = durationMs
	if err != nil {
		cp.LastError = err.Error()
		_ = st.SaveCheckpoint(cp)
		return nil, err
	}

	cp.LastOffset += int64(n)
	cp.LastSentAt = time.Now().Unix()
	cp.LastError = ""

	if err := st.SaveCheckpoint(cp); err != nil {
		return nil, err
	}

	return &cp, nil
}

func buildPayload(input registry.LogInput, cfg config.Config, cp state.Checkpoint, data []byte) (Payload, error) {
	hostname, _ := os.Hostname()

	payload := Payload{
		DeviceID:    cfg.DeviceID,
		PFHostname:  hostname,
		Package:     input.Package,
		LogID:       input.LogID,
		Path:        input.Path,
		Inode:       cp.Identity.Inode,
		OffsetStart: cp.LastOffset,
		OffsetEnd:   cp.LastOffset + int64(len(data)),
		SentAt:      time.Now().Unix(),
	}

	switch strings.ToLower(cfg.ShipFormat) {
	case "", "lines":
		lines := strings.Split(string(data), "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		payload.Lines = lines
	case "raw":
		payload.Raw = string(data)
	default:
		return Payload{}, fmt.Errorf("ship_format invalido: %s", cfg.ShipFormat)
	}

	return payload, nil
}

func fillCheckpointWindow(cp *state.Checkpoint, input registry.LogInput, payload Payload) {
	cp.LastLinesSent = 0
	cp.LastWindowStart = 0
	cp.LastWindowEnd = 0
	if len(payload.Lines) == 0 {
		return
	}
	cp.LastLinesSent = len(payload.Lines)
	if input.TimestampLayout == "" {
		return
	}
	start, end := parseTimestampWindow(payload.Lines, input.TimestampLayout)
	cp.LastWindowStart = start
	cp.LastWindowEnd = end
}

func parseTimestampWindow(lines []string, layout string) (int64, int64) {
	var start int64
	var end int64
	for _, line := range lines {
		ts, ok := parseLineTimestamp(line, layout)
		if !ok {
			continue
		}
		if start == 0 {
			start = ts.Unix()
		}
		end = ts.Unix()
	}
	return start, end
}

func parseLineTimestamp(line string, layout string) (time.Time, bool) {
	if layout == "" || len(line) < len(layout) {
		return time.Time{}, false
	}
	prefix := line[:len(layout)]
	ts, err := time.ParseInLocation(layout, prefix, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return ts, true
}

func postPayload(ctx context.Context, cfg config.Config, payload Payload) (int, int64, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, 0, err
	}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(body); err != nil {
		_ = zw.Close()
		return 0, 0, err
	}
	if err := zw.Close(); err != nil {
		return 0, 0, err
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Endpoint, &buf)
	if err != nil {
		return 0, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	if cfg.AuthToken != "" {
		header := strings.TrimSpace(cfg.AuthHeaderName)
		if header == "" {
			header = "x-auth-n8n"
		}
		req.Header.Set(header, cfg.AuthToken)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, time.Since(start).Milliseconds(), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, time.Since(start).Milliseconds(), fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return resp.StatusCode, time.Since(start).Milliseconds(), nil
}

func fileIdentity(info os.FileInfo) (state.FileIdentity, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return state.FileIdentity{}, errors.New("nao foi possivel obter inode")
	}
	return state.FileIdentity{Dev: uint64(stat.Dev), Inode: uint64(stat.Ino)}, nil
}
