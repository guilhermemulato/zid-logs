package shipper

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
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

	if err := postPayload(ctx, cfg, payload); err != nil {
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

func postPayload(ctx context.Context, cfg config.Config, payload Payload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(body); err != nil {
		_ = zw.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Endpoint, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	if cfg.AuthToken != "" {
		req.Header.Set("x-auth-n8n", cfg.AuthToken)
	}

	client, err := httpClient(cfg)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}

func httpClient(cfg config.Config) (*http.Client, error) {
	tlsConfig := &tls.Config{InsecureSkipVerify: cfg.TLS.InsecureSkipVerify}

	if cfg.TLS.CAPath != "" {
		caData, err := os.ReadFile(cfg.TLS.CAPath)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caData) {
			return nil, errors.New("falha ao carregar CA")
		}
		tlsConfig.RootCAs = pool
	}

	if cfg.TLS.ClientCertPath != "" && cfg.TLS.ClientKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLS.ClientCertPath, cfg.TLS.ClientKeyPath)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}, nil
}

func fileIdentity(info os.FileInfo) (state.FileIdentity, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return state.FileIdentity{}, errors.New("nao foi possivel obter inode")
	}
	return state.FileIdentity{Dev: uint64(stat.Dev), Inode: uint64(stat.Ino)}, nil
}
