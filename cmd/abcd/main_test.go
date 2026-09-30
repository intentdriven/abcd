package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// TestWireArmsTheGitleaksOptIn: the composition root registers the gitleaks
// augmenter, so a scanner built for a repository that armed it in
// .abcd/config/gitleaks.json carries it (here as the not-installed gap, PATH
// holding no gitleaks), and one built for a repository that did not has none.
func TestWireArmsTheGitleaksOptIn(t *testing.T) {
	restore := scanner.SetDefaultAugmenter(nil)
	defer restore()
	wire()
	t.Setenv("PATH", t.TempDir())

	armed := t.TempDir()
	cfg := filepath.Join(armed, ".abcd", "config")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "gitleaks.json"), []byte(`{"schema_version":1,"enabled":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	sc, err := scanner.New(armed)
	if err != nil {
		t.Fatal(err)
	}
	if gap := sc.AugmenterGap(); !strings.Contains(gap, "gitleaks configured but not found") {
		t.Fatalf("an armed repository's scanner does not carry gitleaks: gap %q", gap)
	}

	sc, err = scanner.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if gap := sc.AugmenterGap(); gap != "" {
		t.Fatalf("a repository that did not opt in has a gap: %q", gap)
	}
	if bad, why := sc.Unavailable(); bad {
		t.Fatalf("a repository that did not opt in is degraded: %s", why)
	}
}
