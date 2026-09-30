package ahoy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core"
)

// TestDetectVersionIgnoresADevSide is the detection half of
// iss-2608241115259170: version.upgrade is a required gap, and with either side
// a dev build it could never settle. A dev-stamped repo run by a release binary
// asked for a re-stamp that the next dev install undid, and the reverse flapped
// the tracked config back; neither is an upgrade.
func TestDetectVersionIgnoresADevSide(t *testing.T) {
	prev := core.Version
	t.Cleanup(func() { core.Version = prev })
	for _, c := range []struct {
		name, recorded, running string
		wantGap                 bool
	}{
		{"release to release is an upgrade", "v1.2.0", "v1.3.0", true},
		{"a dev-stamped repo under a release binary", "dev", "v1.3.0", false},
		{"a release-stamped repo under a dev binary", "v1.2.0", "dev", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, ".abcd"), 0o755); err != nil {
				t.Fatal(err)
			}
			cfg := `{"meta":{"setup_version":"` + c.recorded + `","setup_date":"2026-09-01","schema_version":1}}` + "\n"
			if err := os.WriteFile(filepath.Join(dir, ".abcd", "config.json"), []byte(cfg), 0o644); err != nil {
				t.Fatal(err)
			}
			core.Version = c.running
			got := hasGap(detectVersion(dir), "version.upgrade")
			if got != c.wantGap {
				t.Fatalf("version.upgrade raised = %v, want %v", got, c.wantGap)
			}
		})
	}
}
