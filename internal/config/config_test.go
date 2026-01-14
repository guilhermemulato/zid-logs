package config

import "testing"

func TestApplyDefaults(t *testing.T) {
	cfg := Config{}
	cfg = ApplyDefaults(cfg)

	if cfg.RotateAt == "" && cfg.IntervalRotateSeconds == 0 {
		t.Fatalf("Rotate schedule not set")
	}
	if cfg.ShipIntervalHours == 0 && cfg.IntervalShipSeconds == 0 {
		t.Fatalf("Ship interval not set")
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
