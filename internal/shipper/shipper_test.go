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
