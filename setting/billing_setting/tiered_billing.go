package billing_setting

import (
	"fmt"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/samber/lo"
)

const (
	BillingModeRatio       = "ratio"
	BillingModeTieredExpr  = "tiered_expr"
	BillingModePerSecond   = "per-second"
	BillingModeField       = "billing_mode"
	BillingExprField       = "billing_expr"
	DurationBillingField   = "duration_billing"
)

// BillingSetting is managed by config.GlobalConfig.Register.
// DB keys: billing_setting.billing_mode, billing_setting.billing_expr, billing_setting.duration_billing
type BillingSetting struct {
	BillingMode     map[string]string `json:"billing_mode"`
	BillingExpr     map[string]string `json:"billing_expr"`
	DurationBilling map[string]int    `json:"duration_billing"`
}

var billingSetting = BillingSetting{
	BillingMode:     make(map[string]string),
	BillingExpr:     make(map[string]string),
	DurationBilling: make(map[string]int),
}

// durationBillingFallbacks holds hardcoded per-second models registered by
// individual adaptors at init time.  Admin DB settings override these.
var durationBillingFallbacks map[string]int

// RegisterDurationBillingFallbacks is called by adaptors (e.g., doubao) to
// register a hardcoded set of per-second billing models and their default
// seconds.  Must be called during package init.
func RegisterDurationBillingFallbacks(models map[string]int) {
	if durationBillingFallbacks == nil {
		durationBillingFallbacks = make(map[string]int)
	}
	for k, v := range models {
		durationBillingFallbacks[k] = v
	}
}

func init() {
	config.GlobalConfig.Register("billing_setting", &billingSetting)
}

// ---------------------------------------------------------------------------
// Read accessors (hot path, must be fast)
// ---------------------------------------------------------------------------

func GetBillingMode(model string) string {
	if mode, ok := billingSetting.BillingMode[model]; ok {
		return mode
	}
	return BillingModeRatio
}

func GetBillingExpr(model string) (string, bool) {
	expr, ok := billingSetting.BillingExpr[model]
	return expr, ok
}

func GetBillingModeCopy() map[string]string {
	return lo.Assign(billingSetting.BillingMode)
}

func GetBillingExprCopy() map[string]string {
	return lo.Assign(billingSetting.BillingExpr)
}

func GetDurationBillingCopy() map[string]int {
	return lo.Assign(billingSetting.DurationBilling)
}

// IsDurationBillingModel returns true if the model uses per-second billing
// (either configured via admin settings, or registered as a hardcoded fallback).
func IsDurationBillingModel(model string) bool {
	if billingSetting.BillingMode[model] == BillingModePerSecond {
		return true
	}
	_, inFallback := durationBillingFallbacks[model]
	return inFallback
}

// GetDurationBillingDefault returns the configured default seconds for a
// per-second model, and whether it is configured.
// Priority: DB setting > fallback.
func GetDurationBillingDefault(model string) (int, bool) {
	if sec, ok := billingSetting.DurationBilling[model]; ok && sec > 0 {
		return sec, true
	}
	if sec, ok := durationBillingFallbacks[model]; ok && sec > 0 {
		return sec, true
	}
	return 0, false
}

func GetPricingSyncData(base map[string]any) map[string]any {
	extra := make(map[string]any, 3)
	if modes := GetBillingModeCopy(); len(modes) > 0 {
		extra[BillingModeField] = modes
	}
	if exprs := GetBillingExprCopy(); len(exprs) > 0 {
		extra[BillingExprField] = exprs
	}
	if durs := GetDurationBillingCopy(); len(durs) > 0 {
		extra[DurationBillingField] = durs
	}
	return lo.Assign(base, extra)
}

// ---------------------------------------------------------------------------
// Smoke test (called externally for validation before save)
// ---------------------------------------------------------------------------

func SmokeTestExpr(exprStr string) error {
	return smokeTestExpr(exprStr)
}

func smokeTestExpr(exprStr string) error {
	vectors := []billingexpr.TokenParams{
		{P: 0, C: 0, Len: 0},
		{P: 1000, C: 1000, Len: 1000},
		{P: 100000, C: 100000, Len: 100000},
		{P: 1000000, C: 1000000, Len: 1000000},
	}
	requests := []billingexpr.RequestInput{
		{},
		{
			Headers: map[string]string{
				"anthropic-beta": "fast-mode-2026-02-01",
			},
			Body: []byte(`{"service_tier":"fast","stream_options":{"include_usage":true},"messages":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21]}`),
		},
	}

	for _, v := range vectors {
		for _, request := range requests {
			result, _, err := billingexpr.RunExprWithRequest(exprStr, v, request)
			if err != nil {
				return fmt.Errorf("vector {p=%g, c=%g}: run failed: %w", v.P, v.C, err)
			}
			if result < 0 {
				return fmt.Errorf("vector {p=%g, c=%g}: result %f < 0", v.P, v.C, result)
			}
		}
	}
	return nil
}
