package ilm

import (
	"testing"
	"time"

	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

func TestToILMRuleExpiryDays(t *testing.T) {
	days := "10"
	opts := LifecycleOptions{
		ID:         "rule-1",
		ExpiryDays: &days,
	}
	rule, err := opts.ToILMRule()
	if err != nil {
		t.Fatalf("ToILMRule() error = %v", err)
	}
	if rule.Expiration.Days != 10 {
		t.Fatalf("Expiration.Days = %d, want 10", rule.Expiration.Days)
	}
}

func TestToILMRuleDisabled(t *testing.T) {
	disabled := false
	opts := LifecycleOptions{
		ID:         "rule-2",
		Status:     &disabled,
		ExpiryDays: strPtr("5"),
	}
	rule, err := opts.ToILMRule()
	if err != nil {
		t.Fatalf("ToILMRule() error = %v", err)
	}
	if rule.Status != "Disabled" {
		t.Fatalf("Status = %q, want Disabled", rule.Status)
	}
}

func TestValidateILMRule(t *testing.T) {
	future := time.Now().Add(72 * time.Hour).Format(defaultILMDateFormat)
	rule := lifecycle.Rule{
		Expiration: lifecycle.Expiration{Days: 10},
	}
	if err := validateILMRule(rule); err != nil {
		t.Fatalf("validateILMRule() error = %v", err)
	}
	rule = lifecycle.Rule{
		Transition: lifecycle.Transition{
			Days:         30,
			StorageClass: "GLACIER",
		},
		Expiration: lifecycle.Expiration{Date: mustParseDate(t, future)},
	}
	if err := validateILMRule(rule); err != nil {
		t.Fatalf("validateILMRule() transition+expiry error = %v", err)
	}
}

func TestApplyRuleFields(t *testing.T) {
	dest := lifecycle.Rule{ID: "x"}
	days := "15"
	opts := LifecycleOptions{ExpiryDays: &days}
	if err := ApplyRuleFields(&dest, opts); err != nil {
		t.Fatalf("ApplyRuleFields() error = %v", err)
	}
	if dest.Expiration.Days != 15 {
		t.Fatalf("Expiration.Days = %d", dest.Expiration.Days)
	}
}

func TestParseExpiry(t *testing.T) {
	days := "20"
	exp, err := parseExpiry(nil, &days, nil, nil)
	if err != nil || exp.Days != 20 {
		t.Fatalf("parseExpiry() = (%+v, %v)", exp, err)
	}
}

func mustParseDate(t *testing.T, s string) lifecycle.ExpirationDate {
	t.Helper()
	d, err := parseExpiryDate(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
