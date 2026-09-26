package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func providerGap(gaps []Gap, id string) (Gap, bool) {
	if g := findGap(gaps, id); g != nil {
		return *g, true
	}
	return Gap{}, false
}

// TestAhoyExplainsTheProviderAdapterWhenNoneIsConfigured is criterion 6: a
// repository abcd manages, on a machine with no provider configured, carries
// the optional gap that explains the aggregator, what abcd would use it for,
// that everything works without it, and where the walkthrough is; it is
// advisory, so install neither prompts for it nor counts it as remaining.
func TestAhoyExplainsTheProviderAdapterWhenNoneIsConfigured(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	g, ok := providerGap(det.Gaps, ProviderAdapterGapID)
	if !ok {
		t.Fatalf("no %s gap in %v", ProviderAdapterGapID, gapIDs(det.Gaps))
	}
	if g.Required || g.Resolvable || g.Scope != "machine" {
		t.Fatalf("gap = %+v; want optional, not resolvable by install, machine-scoped", g)
	}
	text := g.Title + " " + g.Detail + " " + g.FixHint
	for _, want := range []string{
		"aggregator",            // what it is
		"decision models",       // what abcd would use it for
		"cheap judgements",      //
		"runs on the host",      // what works without it, and declining says so
		"abcd ahoy --providers", // the walkthrough's explanation
		"abcd ahoy connect",     // the walkthrough's write
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the gap does not say %q:\n%s", want, text)
		}
	}
}

// TestAConfiguredProviderClosesTheGap: once a provider block is on the
// machine, the explanation is not repeated.
func TestAConfiguredProviderClosesTheGap(t *testing.T) {
	home, _ := setupHermetic(t)
	repo := installedRepo(t)
	if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := `{"oracle":{"api":{"openrouter":{"base_url":"https://openrouter.ai/api/v1","key":"openrouter","models":["typesafe/jev-1.13"]}}}}`
	if err := os.WriteFile(filepath.Join(home, ".abcd", "config.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := providerGap(det.Gaps, ProviderAdapterGapID); ok {
		t.Fatal("the no-provider gap persists with a provider configured")
	}
}

// TestARefusedProviderConfigurationIsNamed: a configuration the adapter
// refuses is a gap naming the refusal, never silence and never a crash of the
// detection.
func TestARefusedProviderConfigurationIsNamed(t *testing.T) {
	home, _ := setupHermetic(t)
	repo := installedRepo(t)
	if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := `{"oracle":{"api":{"openrouter":{"base_url":"https://openrouter.ai/api/v1","models":["anthropic/claude-opus-4"]}}}}`
	if err := os.WriteFile(filepath.Join(home, ".abcd", "config.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	g, ok := providerGap(det.Gaps, ProviderAdapterRefusedGapID)
	if !ok || !strings.Contains(g.Detail, "anthropic/*") {
		t.Fatalf("gap = %+v, %v; want the refusal named", g, ok)
	}
	if strings.Contains(g.Detail, home) {
		t.Fatalf("the gap carries the home path: %s", g.Detail)
	}
}
