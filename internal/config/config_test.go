package config

import "testing"

func TestApplyDefaults(t *testing.T) {
	cfg := Config{}
	cfg = ApplyDefaults(cfg)

	if cfg.IntervalRotateSeconds == 0 {
		t.Fatalf("IntervalRotateSeconds not set")
	}
	if cfg.IntervalShipSeconds == 0 {
		t.Fatalf("IntervalShipSeconds not set")
	}
	if cfg.MaxBytesPerShip == 0 {
		t.Fatalf("MaxBytesPerShip not set")
	}
	if cfg.ShipFormat == "" {
		t.Fatalf("ShipFormat not set")
	}
	if cfg.Defaults.MaxSizeMB == 0 {
		t.Fatalf("Defaults.MaxSizeMB not set")
	}
	if cfg.Defaults.Keep == 0 {
		t.Fatalf("Defaults.Keep not set")
	}
	if cfg.Defaults.Compress == nil {
		t.Fatalf("Defaults.Compress not set")
	}
}
