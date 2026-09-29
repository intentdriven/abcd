package termsafe

import (
	"strings"
	"testing"
)

func TestDescribeRefusedNeverQuotes(t *testing.T) {
	if got := DescribeRefused(""); got != "an empty value" {
		t.Errorf("DescribeRefused(\"\") = %q", got)
	}
	const v = "zzleak-7f3a"
	got := DescribeRefused(v)
	if strings.Contains(got, v) || !strings.Contains(got, "11-byte") {
		t.Errorf("DescribeRefused(%q) = %q, want the length and not the value", v, got)
	}
}
