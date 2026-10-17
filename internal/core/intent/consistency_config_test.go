package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// consistency_config_test.go covers consistency.max_request_bytes
// (itd-2609291923559186, A4): the request limit the emit measures the corpus
// against, read through the one layered resolver.

// cxRoots lays a checkout and a home under one temp dir, writing the
// repository and machine config files when given (0o600, which the machine
// layer's guard admits). Nothing reads the real home.
func cxRoots(t *testing.T, repoJSON, machineJSON string) layered.Roots {
	t.Helper()
	base := t.TempDir()
	r := layered.Roots{Repo: filepath.Join(base, "repo"), Home: filepath.Join(base, "home")}
	put := func(p, body string) {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range []string{r.Repo, r.Home} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if repoJSON != "" {
		put(filepath.Join(r.Repo, ".abcd", "config.json"), repoJSON)
	}
	if machineJSON != "" {
		put(abcdhome.Path(r.Home, "config.json"), machineJSON)
	}
	return r
}

// TestConsistencyConfigReadsTheLimitThroughTheLayers: the repository value
// wins, the machine value applies when the repository is silent, and the
// bundled 512 KiB applies when neither sets it, each with its origin.
func TestConsistencyConfigReadsTheLimitThroughTheLayers(t *testing.T) {
	cases := []struct {
		name, repo, machine string
		want                int
		origin              string
	}{
		{"bundled", `{"docs":{"target":"agents_md"}}`, "", 524288, "bundled"},
		{"machine", "", `{"consistency":{"max_request_bytes":131072}}`, 131072, abcdhome.Display("config.json")},
		{"repo over machine", `{"consistency":{"max_request_bytes":65536}}`, `{"consistency":{"max_request_bytes":131072}}`, 65536, ".abcd/config.json"},
		{"upper bound", `{"consistency":{"max_request_bytes":1048576}}`, "", 1048576, ".abcd/config.json"},
		// on_finding is the pre-commit gate's key (step 4): claimed, so a
		// repository that sets it is not refused, and not read here.
		{"on_finding claimed", `{"consistency":{"on_finding":"warn"}}`, "", 524288, "bundled"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := LoadConsistencyConfig(cxRoots(t, tc.repo, tc.machine))
			if err != nil {
				t.Fatal(err)
			}
			if c.MaxRequestBytes != tc.want || c.MaxRequestBytesOrigin != tc.origin {
				t.Fatalf("limit %d from %q, want %d from %q", c.MaxRequestBytes, c.MaxRequestBytesOrigin, tc.want, tc.origin)
			}
		})
	}
}

// TestConsistencyConfigRefusesWhatWouldSilentlyDoLess: a misspelt key and a
// value outside 64 KiB to 1 MiB are refused naming their file, never replaced
// by the default.
func TestConsistencyConfigRefusesWhatWouldSilentlyDoLess(t *testing.T) {
	cases := []struct {
		name, repo, machine string
		want                []string
	}{
		{"misspelt", `{"consistency":{"max_request_byte":65536}}`, "", []string{".abcd/config.json", "consistency.max_request_byte"}},
		{"below 64 KiB", `{"consistency":{"max_request_bytes":65535}}`, "", []string{".abcd/config.json", "65535", "65536"}},
		{"above 1 MiB", "", `{"consistency":{"max_request_bytes":1048577}}`, []string{abcdhome.Display("config.json"), "1048577", "1048576"}},
		{"not a number", `{"consistency":{"max_request_bytes":"big"}}`, "", []string{".abcd/config.json", "max_request_bytes"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadConsistencyConfig(cxRoots(t, tc.repo, tc.machine))
			if err == nil {
				t.Fatal("loaded; want a refusal")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("refusal %q does not name %q", err, w)
				}
			}
		})
	}
}
