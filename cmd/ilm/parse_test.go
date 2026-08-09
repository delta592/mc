package ilm

import (
	"testing"
	"time"

	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

func TestGetPrefix(t *testing.T) {
	tests := []struct {
		rule lifecycle.Rule
		want string
	}{
		{lifecycle.Rule{Prefix: "legacy"}, "legacy"},
		{lifecycle.Rule{RuleFilter: lifecycle.Filter{Prefix: "filter"}}, "filter"},
		{lifecycle.Rule{RuleFilter: lifecycle.Filter{And: lifecycle.And{Prefix: "and"}}}, "and"},
		{lifecycle.Rule{}, ""},
	}
	for i, tt := range tests {
		if got := getPrefix(tt.rule); got != tt.want {
			t.Fatalf("%d: getPrefix() = %q, want %q", i+1, got, tt.want)
		}
	}
}

func TestExtractILMTags(t *testing.T) {
	tags := extractILMTags("k1=v1&k2=v2")
	if len(tags) != 2 || tags[0].Key != "k1" || tags[1].Value != "v2" {
		t.Fatalf("extractILMTags() = %+v", tags)
	}
	if len(extractILMTags("")) != 0 {
		t.Fatal("extractILMTags(\"\") should be empty")
	}
	if len(extractILMTags("keyonly")) != 1 || extractILMTags("keyonly")[0].Value != "" {
		t.Fatalf("extractILMTags(keyonly) = %+v", extractILMTags("keyonly"))
	}
}

func TestValidateRuleAction(t *testing.T) {
	if err := validateRuleAction(lifecycle.Rule{}); err == nil {
		t.Fatal("expected error for empty rule")
	}
	rule := lifecycle.Rule{Expiration: lifecycle.Expiration{Days: 1}}
	if err := validateRuleAction(rule); err != nil {
		t.Fatalf("validateRuleAction() error = %v", err)
	}
}

func TestValidateExpiration(t *testing.T) {
	rule := lifecycle.Rule{
		Expiration: lifecycle.Expiration{
			Days:         1,
			Date:         lifecycle.ExpirationDate{Time: time.Now()},
			DeleteMarker: true,
		},
	}
	if err := validateExpiration(rule); err == nil {
		t.Fatal("expected error for multiple expiration params")
	}
}

func TestValidateTransition(t *testing.T) {
	rule := lifecycle.Rule{
		Transition: lifecycle.Transition{
			Days: 1,
			Date: lifecycle.ExpirationDate{Time: time.Now()},
		},
	}
	if err := validateTransition(rule); err == nil {
		t.Fatal("expected error for transition days and date")
	}
}

func TestValidateTranDays(t *testing.T) {
	rule := lifecycle.Rule{Transition: lifecycle.Transition{Days: -1}}
	if err := validateTranDays(rule); err == nil {
		t.Fatal("expected error for negative transition days")
	}
	rule = lifecycle.Rule{Transition: lifecycle.Transition{Days: 10, StorageClass: "STANDARD_IA"}}
	if err := validateTranDays(rule); err == nil {
		t.Fatal("expected error for STANDARD_IA < 30 days")
	}
}

func TestValidateNoncurrentExpiration(t *testing.T) {
	rule := lifecycle.Rule{
		NoncurrentVersionExpiration: lifecycle.NoncurrentVersionExpiration{NoncurrentDays: -1},
	}
	if err := validateNoncurrentExpiration(rule); err == nil {
		t.Fatal("expected error for negative noncurrent days")
	}
}

func TestValidateNoncurrentTransition(t *testing.T) {
	rule := lifecycle.Rule{
		NoncurrentVersionTransition: lifecycle.NoncurrentVersionTransition{NoncurrentDays: -1},
	}
	if err := validateNoncurrentTransition(rule); err == nil {
		t.Fatal("expected error for negative noncurrent transition days")
	}
	rule = lifecycle.Rule{
		NoncurrentVersionTransition: lifecycle.NoncurrentVersionTransition{NoncurrentDays: 5},
	}
	if err := validateNoncurrentTransition(rule); err == nil {
		t.Fatal("expected error for missing storage class")
	}
}

func TestValidateTranExpDate(t *testing.T) {
	now := time.Now()
	rule := lifecycle.Rule{
		Expiration: lifecycle.Expiration{Date: lifecycle.ExpirationDate{Time: now}},
		Transition: lifecycle.Transition{
			Date:         lifecycle.ExpirationDate{Time: now.Add(24 * time.Hour)},
			StorageClass: "GLACIER",
		},
	}
	if err := validateTranExpDate(rule); err == nil {
		t.Fatal("expected error when expiration is before transition")
	}
}

func TestParseExpiryDays(t *testing.T) {
	if _, err := parseExpiryDays("0"); err == nil {
		t.Fatal("expected error for zero expiry days")
	}
	if _, err := parseExpiryDays("abc"); err == nil {
		t.Fatal("expected error for invalid expiry days")
	}
	days, err := parseExpiryDays("10")
	if err != nil || days != 10 {
		t.Fatalf("parseExpiryDays() = (%v, %v)", days, err)
	}
}

func TestParseExpiryDate(t *testing.T) {
	future := time.Now().Add(48 * time.Hour).Format(defaultILMDateFormat)
	if _, err := parseExpiryDate(future); err != nil {
		t.Fatalf("parseExpiryDate() error = %v", err)
	}
	if _, err := parseExpiryDate("bad-date"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseTransition(t *testing.T) {
	days := "30"
	class := "GLACIER"
	transition, err := parseTransition(&class, nil, &days)
	if err != nil || transition.Days != 30 || transition.StorageClass != "GLACIER" {
		t.Fatalf("parseTransition() = (%+v, %v)", transition, err)
	}
}

func TestLifecycleOptionsFilter(t *testing.T) {
	prefix := "logs/"
	tags := "k=v"
	opts := LifecycleOptions{Prefix: &prefix, Tags: &tags}
	filter := opts.Filter()
	if filter.And.Prefix != prefix || len(filter.And.Tags) != 1 {
		t.Fatalf("Filter() with multiple predicates = %+v", filter)
	}

	singlePrefix := "data/"
	opts = LifecycleOptions{Prefix: &singlePrefix}
	filter = opts.Filter()
	if filter.Prefix != singlePrefix {
		t.Fatalf("Filter() single prefix = %+v", filter)
	}
}

func TestRemoveILMRule(t *testing.T) {
	cfg := &lifecycle.Configuration{
		Rules: []lifecycle.Rule{{ID: "a"}, {ID: "b"}},
	}
	out, err := RemoveILMRule(cfg, "a")
	if err != nil || len(out.Rules) != 1 || out.Rules[0].ID != "b" {
		t.Fatalf("RemoveILMRule() = (%+v, %v)", out, err)
	}
	if _, err := RemoveILMRule(cfg, "missing"); err == nil {
		t.Fatal("expected error for missing rule id")
	}
	if _, err := RemoveILMRule(nil, "a"); err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestToTables(t *testing.T) {
	cfg := &lifecycle.Configuration{
		Rules: []lifecycle.Rule{
			{
				ID:         "exp",
				Status:     "Enabled",
				Expiration: lifecycle.Expiration{Days: 7},
			},
			{
				ID:     "tier",
				Status: "Enabled",
				Transition: lifecycle.Transition{
					Days:         30,
					StorageClass: "GLACIER",
				},
			},
		},
	}
	tables := ToTables(cfg)
	if len(tables) == 0 {
		t.Fatal("ToTables() returned no tables")
	}
}

func TestGetExpirationDays(t *testing.T) {
	rule := lifecycle.Rule{Expiration: lifecycle.Expiration{Days: 5}}
	if got := getExpirationDays(rule); got != 5 {
		t.Fatalf("getExpirationDays() = %d, want 5", got)
	}
}

func TestGetTransitionDays(t *testing.T) {
	rule := lifecycle.Rule{Transition: lifecycle.Transition{Days: 15}}
	if got := getTransitionDays(rule); got != 15 {
		t.Fatalf("getTransitionDays() = %d, want 15", got)
	}
}
