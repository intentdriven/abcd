package ahoy

import (
	"slices"
	"strings"
	"testing"
)

// yesPipe is `yes | abcd ahoy install`: every approval is answered yes and
// every value question is answered with the line "y".
type yesPipe struct{}

func (yesPipe) Confirm(string) bool                    { return true }
func (yesPipe) Prompt(string, []string, string) string { return "y" }

// TestRefusedScanDeepAnswerKeepsTheFlaggedValues is the iss-2610020700263496
// detector, the reporter's run: `yes | abcd ahoy install --adopt --docs-target
// agents_md --visibility private --oracle-backend host-delegated` beside
// trufflehog. The piped "y" to scan_deep is not true or false, so scan.deep is
// not saved; the three values the flags gave are saved all the same, the one
// gap left among the settings is scan.deep's, and a note names the answer, the
// two accepted ones and --scan-deep.
func TestRefusedScanDeepAnswerKeepsTheFlaggedValues(t *testing.T) {
	setupHermetic(t)
	trufflehogResolves(t, true)
	repo := gitRepo(t)
	adopt := true
	opts := InstallOptions{
		Adopt: &adopt,
		ValueOverrides: map[string]string{
			"visibility":     "private",
			"docs_target":    "agents_md",
			"oracle_backend": "host-delegated",
		},
	}
	res, err := Install(repo, opts, yesPipe{})
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := readConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []struct{ section, key, want string }{
		{"repo", "visibility", "private"},
		{"docs", "target", "agents_md"},
		{"oracle", "backend", "host-delegated"},
	} {
		if got, _ := stringVal(subMap(cfg, s.section), s.key); got != s.want {
			t.Errorf("config.json %s.%s = %q, want %q, the value its flag gave", s.section, s.key, got, s.want)
		}
	}
	if _, present := boolVal(subMap(cfg, "scan"), "deep"); present {
		t.Errorf("the refused scan_deep answer \"y\" was saved: %v", cfg)
	}

	var settings []string
	for _, id := range res.Remaining {
		if strings.HasPrefix(id, "config.") || strings.HasPrefix(id, "visibility.") {
			settings = append(settings, id)
		}
	}
	if !slices.Equal(settings, []string{"config.scan_deep_missing"}) {
		t.Errorf("remaining settings gaps = %v, want only config.scan_deep_missing (remaining %v)", settings, res.Remaining)
	}

	var named bool
	for _, n := range res.Notes {
		named = named || strings.Contains(n, `"y"`) && strings.Contains(n, "scan_deep") &&
			strings.Contains(n, "true") && strings.Contains(n, "false") && strings.Contains(n, "--scan-deep")
	}
	if !named {
		t.Errorf("no note names the refused scan_deep answer \"y\", the accepted true and false, and --scan-deep: %q", res.Notes)
	}
	if !hasBlock(t, repo+"/AGENTS.md") {
		t.Errorf("AGENTS.md carries no block, though --docs-target agents_md was saved")
	}
}

// TestRefusedAnswerKeepsEveryFlaggedValue is the sibling sweep: any refused
// answer, not only scan_deep's, used to drop every value the flags gave, since
// the settings were saved in one write after the last question. A typo'd
// docs_target answer now drops only docs.target, and the flags' visibility and
// oracle backend are saved.
func TestRefusedAnswerKeepsEveryFlaggedValue(t *testing.T) {
	dir := t.TempDir()
	a := &applyCtx{
		cwd:      dir,
		approved: map[GapCategory]bool{ConfigChange: true},
		gapPresent: map[string]bool{
			"config.visibility_missing":     true,
			"config.docs_target_missing":    true,
			"config.oracle_backend_missing": true,
		},
		overrides: map[string]string{"visibility": "private", "oracle_backend": "host-delegated"},
		prompter:  stubPrompter{answers: map[string]string{"docs_target": "clade_md"}},
	}
	cfg := a.stepConfigValues()
	if cfg == nil || cfg.Visibility != "private" || cfg.DocsTarget != "" {
		t.Fatalf("stepConfigValues = %+v, want the saved flags' values and no docs target", cfg)
	}
	m, err := readConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := stringVal(subMap(m, "repo"), "visibility"); v != "private" {
		t.Errorf("repo.visibility = %q, want private, the value --visibility gave", v)
	}
	if v, _ := stringVal(subMap(m, "oracle"), "backend"); v != "host-delegated" {
		t.Errorf("oracle.backend = %q, want host-delegated, the value --oracle-backend gave", v)
	}
	if v, present := stringVal(subMap(m, "docs"), "target"); present {
		t.Errorf("the refused docs_target answer was saved as %q", v)
	}
}
