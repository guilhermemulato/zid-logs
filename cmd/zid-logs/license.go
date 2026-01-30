package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"zid-logs/internal/licensing"
)

func requireLicense(pkg string) error {
	_, err := licensing.Check(pkg)
	return err
}

func startLicenseMonitor(ctx context.Context, pkg string, interval time.Duration) <-chan error {
	out := make(chan error, 1)
	if interval <= 0 {
		interval = licenseCheckInterval
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := requireLicense(pkg); err != nil {
					select {
					case out <- err:
					default:
					}
					return
				}
			}
		}
	}()

	return out
}

func formatLicenseError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, licensing.ErrBadSignature) {
		return "bad_sig"
	}
	if errors.Is(err, licensing.ErrIPCUnavailable) {
		return "ipc indisponivel"
	}
	if errors.Is(err, licensing.ErrInvalidLicense) {
		msg := err.Error()
		msg = strings.TrimSpace(strings.TrimPrefix(msg, licensing.ErrInvalidLicense.Error()+": "))
		if msg == "" || msg == licensing.ErrInvalidLicense.Error() {
			return "licenca invalida"
		}
		return msg
	}
	return fmt.Sprintf("erro licenca: %v", err)
}
