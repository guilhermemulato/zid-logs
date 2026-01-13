package rotate

import (
	"zid-logs/internal/config"
	"zid-logs/internal/registry"
)

func ResolvePolicy(defaults config.RotateDefaults, input registry.InputPolicy) Policy {
	policy := Policy{
		MaxSizeMB:  defaults.MaxSizeMB,
		Keep:       defaults.Keep,
		Compress:   boolValue(defaults.Compress, true),
		MaxAgeDays: 0,
	}

	if input.MaxSizeMB > 0 {
		policy.MaxSizeMB = input.MaxSizeMB
	}
	if input.Keep > 0 {
		policy.Keep = input.Keep
	}
	if input.Compress != nil {
		policy.Compress = *input.Compress
	}
	if input.MaxAgeDays > 0 {
		policy.MaxAgeDays = input.MaxAgeDays
	}

	return policy
}

func boolValue(ptr *bool, fallback bool) bool {
	if ptr == nil {
		return fallback
	}
	return *ptr
}
