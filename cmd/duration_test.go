package cmd

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
)

func TestCmdParseDuration(t *testing.T) {
	tests := []struct {
		in      string
		want    Duration
		wantErr bool
	}{
		{"0", 0, false},
		{"1s", Second, false},
		{"2m", 2 * Minute, false},
		{"1h", Hour, false},
		{"1d", Day, false},
		{"1w", Week, false},
		{"1y", Year, false},
		{"-1h", -Hour, false},
		{"1.5h", Duration(time.Hour + 30*time.Minute), false},
		{"", 0, true},
		{"  ", 0, true},
		{"bad", 0, true},
		{"1x", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseDuration(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("ParseDuration(%q) expected error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseDuration(%q) error = %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("ParseDuration(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestDurationDays(t *testing.T) {
	// Days() returns elapsed hours plus fractional hour from sub-hour remainder.
	require.Equal(t, float64(48), (2 * Day).Days())
}

func TestLeadingInt(t *testing.T) {
	x, rem, err := leadingInt("123abc")
	require.NoError(t, err)
	require.Equal(t, int64(123), x)
	require.Equal(t, "abc", rem)
}

func TestSplitStr(t *testing.T) {
	got := splitStr("a/b/c", "/", 2)
	require.Equal(t, []string{"a", "b/c"}, got)
}

func TestLineTrunc(t *testing.T) {
	require.Equal(t, "hello", lineTrunc("hello", 10))
	require.Contains(t, lineTrunc("hello world foo", 8), "…")
}

func TestCenterText(t *testing.T) {
	got := centerText("x", 5)
	if len(got) != 5 || got[2] != 'x' {
		t.Fatalf("centerText() = %q", got)
	}
}

func TestConservativeFileName(t *testing.T) {
	got := conservativeFileName(`a:b*c?d`)
	require.NotContains(t, got, ":")
	require.NotContains(t, got, "*")
}

func TestGetLookupType(t *testing.T) {
	require.Equal(t, minio.BucketLookupDNS, getLookupType("off"))
	require.Equal(t, minio.BucketLookupPath, getLookupType("on"))
	require.Equal(t, minio.BucketLookupAuto, getLookupType("auto"))
}

func TestIsURLContains(t *testing.T) {
	require.True(t, isURLContains("s3://bucket/prefix", "s3://bucket/prefix/obj", "/"))
	require.False(t, isURLContains("s3://bucket/a", "s3://bucket/b", "/"))
}

func TestIsOlderIsNewer(t *testing.T) {
	now := time.Now()
	require.True(t, isOlder(now.Add(-time.Hour), "2h"))
	require.True(t, isNewer(now.Add(-time.Hour), "30m"))
}

func TestRandString(t *testing.T) {
	src := rand.NewSource(1)
	got := randString(8, src, "pre-")
	require.True(t, len(got) > len("pre-"))
}

func TestUTCNow(t *testing.T) {
	require.Equal(t, time.UTC, UTCNow().Location())
}

func TestIsErrIgnored(t *testing.T) {
	require.False(t, isErrIgnored(nil))
}

func TestRetryMessage(t *testing.T) {
	msg := retryMessage{SourceURL: "src", TargetURL: "dst", Retries: 2}
	require.Contains(t, msg.String(), "src")
	require.Contains(t, msg.JSON(), `"retries"`)
}

func TestNewRetryManager(t *testing.T) {
	ctx := context.Background()
	rm := newRetryManager(ctx, time.Second, 3)
	require.NotNil(t, rm)
	rm.cancelRetry()
}
