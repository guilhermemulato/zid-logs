package shipper

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"zid-logs/internal/config"
	"zid-logs/internal/registry"
	"zid-logs/internal/state"
)

type captured struct {
	Payload Payload
}

func TestShipOnceInodeChangeResetsOffset(t *testing.T) {
	var received []captured
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			defer gz.Close()
			data, _ := io.ReadAll(gz)
			var payload Payload
			if err := json.Unmarshal(data, &payload); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			received = append(received, captured{Payload: payload})
		} else {
			data, _ := io.ReadAll(r.Body)
			var payload Payload
			_ = json.Unmarshal(data, &payload)
			received = append(received, captured{Payload: payload})
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")

	data1 := []byte("line1\nline2\n")
	if err := os.WriteFile(logPath, data1, 0644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	st, err := state.Open(filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatalf("state open: %v", err)
	}
	defer st.Close()

	cfg := config.Config{
		Enabled:         true,
		Endpoint:        server.URL,
		AuthToken:       "token",
		DeviceID:        "dev",
		ShipFormat:      "lines",
		MaxBytesPerShip: 1024,
	}

	input := registry.LogInput{Package: "zid-proxy", LogID: "main", Path: logPath}

	if _, err := ShipOnce(context.Background(), input, cfg, st); err != nil {
		t.Fatalf("ShipOnce error: %v", err)
	}

	if len(received) != 1 {
		t.Fatalf("expected 1 payload, got %d", len(received))
	}
	if received[0].Payload.OffsetStart != 0 {
		t.Fatalf("expected offset_start 0")
	}
	if received[0].Payload.OffsetEnd != int64(len(data1)) {
		t.Fatalf("unexpected offset_end")
	}

	if err := os.Rename(logPath, logPath+".1"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	data2 := []byte("new\n")
	if err := os.WriteFile(logPath, data2, 0644); err != nil {
		t.Fatalf("write new log: %v", err)
	}

	if _, err := ShipOnce(context.Background(), input, cfg, st); err != nil {
		t.Fatalf("ShipOnce error: %v", err)
	}

	if len(received) != 2 {
		t.Fatalf("expected 2 payloads, got %d", len(received))
	}
	if received[1].Payload.OffsetStart != 0 {
		t.Fatalf("expected offset_start reset after inode change")
	}
	if received[1].Payload.OffsetEnd != int64(len(data2)) {
		t.Fatalf("unexpected offset_end after inode change")
	}
}

func TestShipOnceUpdatesWindowAndLines(t *testing.T) {
	var received []captured
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			defer gz.Close()
			data, _ := io.ReadAll(gz)
			var payload Payload
			if err := json.Unmarshal(data, &payload); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			received = append(received, captured{Payload: payload})
		} else {
			data, _ := io.ReadAll(r.Body)
			var payload Payload
			_ = json.Unmarshal(data, &payload)
			received = append(received, captured{Payload: payload})
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	layout := "2006-01-02T15:04:05Z07:00"
	lines := []string{
		"2026-01-20T09:59:04-03:00 | a",
		"2026-01-20T10:00:05-03:00 | b",
	}
	data := []byte(lines[0] + "\n" + lines[1] + "\n")
	if err := os.WriteFile(logPath, data, 0644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	st, err := state.Open(filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatalf("state open: %v", err)
	}
	defer st.Close()

	cfg := config.Config{
		Enabled:         true,
		Endpoint:        server.URL,
		AuthToken:       "token",
		DeviceID:        "dev",
		ShipFormat:      "lines",
		MaxBytesPerShip: 1024,
	}
	input := registry.LogInput{
		Package:         "zid-proxy",
		LogID:           "main",
		Path:            logPath,
		TimestampLayout: layout,
	}

	if _, err := ShipOnce(context.Background(), input, cfg, st); err != nil {
		t.Fatalf("ShipOnce error: %v", err)
	}

	cp, ok, err := st.GetCheckpoint(input.Package, input.LogID, input.Path)
	if err != nil {
		t.Fatalf("get checkpoint: %v", err)
	}
	if !ok {
		t.Fatalf("expected checkpoint to exist")
	}
	if cp.LastLinesSent != 2 {
		t.Fatalf("expected 2 lines sent, got %d", cp.LastLinesSent)
	}

	startTs, _ := time.ParseInLocation(layout, "2026-01-20T09:59:04-03:00", time.Local)
	endTs, _ := time.ParseInLocation(layout, "2026-01-20T10:00:05-03:00", time.Local)
	if cp.LastWindowStart != startTs.Unix() {
		t.Fatalf("unexpected window start: %d", cp.LastWindowStart)
	}
	if cp.LastWindowEnd != endTs.Unix() {
		t.Fatalf("unexpected window end: %d", cp.LastWindowEnd)
	}
	if len(received) != 1 {
		t.Fatalf("expected 1 payload, got %d", len(received))
	}
	if len(received[0].Payload.Lines) != 2 {
		t.Fatalf("expected payload with 2 lines, got %d", len(received[0].Payload.Lines))
	}
}
