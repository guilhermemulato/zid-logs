package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	DefaultConfigPath = "/usr/local/etc/zid-logs/config.json"
	DefaultInputsDir  = "/var/db/zid-logs/inputs.d"
	DeviceIDPath      = "/var/db/zid-logs/device_id"
	StateDBPath       = "/var/db/zid-logs/state.db"
)

type TLSConfig struct {
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
	CAPath             string `json:"ca_path,omitempty"`
	ClientCertPath     string `json:"client_cert_path,omitempty"`
	ClientKeyPath      string `json:"client_key_path,omitempty"`
}

type RotateDefaults struct {
	MaxSizeMB     int   `json:"max_size_mb"`
	Keep          int   `json:"keep"`
	Compress      *bool `json:"compress"`
	RotateOnStart bool  `json:"rotate_on_start"`
}

type Config struct {
	Enabled               bool           `json:"enabled"`
	Endpoint              string         `json:"endpoint"`
	AuthToken             string         `json:"auth_token"`
	DeviceID              string         `json:"device_id"`
	IntervalRotateSeconds int            `json:"interval_rotate_seconds"`
	IntervalShipSeconds   int            `json:"interval_ship_seconds"`
	MaxBytesPerShip       int            `json:"max_bytes_per_ship"`
	ShipFormat            string         `json:"ship_format"`
	TLS                   TLSConfig      `json:"tls"`
	Defaults              RotateDefaults `json:"defaults"`
}

func DefaultConfig() Config {
	defaultCompress := true
	return Config{
		Enabled:               false,
		IntervalRotateSeconds: 300,
		IntervalShipSeconds:   60,
		MaxBytesPerShip:       256 * 1024,
		ShipFormat:            "lines",
		TLS: TLSConfig{
			InsecureSkipVerify: false,
		},
		Defaults: RotateDefaults{
			MaxSizeMB:     50,
			Keep:          10,
			Compress:      &defaultCompress,
			RotateOnStart: false,
		},
	}
}

func ApplyDefaults(cfg Config) Config {
	def := DefaultConfig()
	if cfg.IntervalRotateSeconds <= 0 {
		cfg.IntervalRotateSeconds = def.IntervalRotateSeconds
	}
	if cfg.IntervalShipSeconds <= 0 {
		cfg.IntervalShipSeconds = def.IntervalShipSeconds
	}
	if cfg.MaxBytesPerShip <= 0 {
		cfg.MaxBytesPerShip = def.MaxBytesPerShip
	}
	if cfg.ShipFormat == "" {
		cfg.ShipFormat = def.ShipFormat
	}
	if cfg.Defaults.MaxSizeMB <= 0 {
		cfg.Defaults.MaxSizeMB = def.Defaults.MaxSizeMB
	}
	if cfg.Defaults.Keep <= 0 {
		cfg.Defaults.Keep = def.Defaults.Keep
	}
	if cfg.Defaults.Compress == nil {
		cfg.Defaults.Compress = def.Defaults.Compress
	}
	return cfg
}

func LoadConfig(path string) (Config, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := DefaultConfig()
			return cfg, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	cfg = ApplyDefaults(cfg)
	return cfg, nil
}

func SaveConfig(path string, cfg Config) error {
	if path == "" {
		path = DefaultConfigPath
	}
	cfg = ApplyDefaults(cfg)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func EnsureDeviceID(cfg Config) (Config, error) {
	if cfg.DeviceID != "" {
		return cfg, nil
	}

	data, err := os.ReadFile(DeviceIDPath)
	if err == nil {
		cfg.DeviceID = string(data)
		return cfg, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return cfg, err
	}

	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return cfg, err
	}

	cfg.DeviceID = hex.EncodeToString(buf)

	if err := os.MkdirAll(filepath.Dir(DeviceIDPath), 0755); err != nil {
		return cfg, err
	}

	if err := os.WriteFile(DeviceIDPath, []byte(cfg.DeviceID), 0644); err != nil {
		return cfg, err
	}

	return cfg, nil
}
