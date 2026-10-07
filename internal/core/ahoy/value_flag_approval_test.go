package ahoy

import (
	"slices"
	"strings"
	"testing"
)

// flagOnlyRepo is a repository set up through the retired docs target both
// whose every other setting is saved and whose .gitignore block is current, so
// no settings value is missing: a --docs-target flag is the one change to a
// setting the run is given (iss-2610071538032843).
func flagOnlyRepo(t *testing.T) string {
	t.Helper()
	repo := retiredRepo(t, "both")
	cfg, err := readConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	setSub(cfg, "scan", "deep", false)
	if err := writeConfig(repo, cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := applyVisibilityBlock(repo, "private"); err != nil {
		t.Fatal(err)
	}
	return repo
}

// confirmRecorder approves the categories in yes, declines every other
// approval, and records each approval asked.
type confirmRecorder struct {
	asked *[]string
	yes   []GapCategory
}

func (c confirmRecorder) Confirm(q string) bool {
	*c.asked = append(*c.asked, q)
	for _, cat := range c.yes {
		if askedCategory(q) == cat {
			return true
		}
	}
	return false
}

func (c confirmRecorder) Prompt(_ string, _ []string, def string) string { return def }

// asksCategory reports whether asked holds the approval of the kind of change c.
func asksCategory(asked []string, c GapCategory) bool {
	return slices.ContainsFunc(asked, func(q string) bool { return askedCategory(q) == c })
}

func savedDocsTarget(t *testing.T, repo string) string {
	t.Helper()
	cfg, err := readConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	v, _ := stringVal(subMap(cfg, "docs"), "target")
	return v
}

// TestAValueFlagAloneAsksTheSettingsApproval: a flag that would change a saved
// setting puts the config-change approval, even where no config-change gap
// would put it, because a value flag is held to that approval like any other
// settings change (the product thinker's ruling on iss-2610071538032843).
func TestAValueFlagAloneAsksTheSettingsApproval(t *testing.T) {
	gaps := []Gap{{ID: "rules.missing", Category: SafeAutocreate, Required: true, Resolvable: true}}
	var asked []string
	approved, declined := resolveApproval(gaps, InstallOptions{}, []string{"docs.target changes"}, confirmRecorder{asked: &asked})
	if !asksCategory(asked, ConfigChange) {
		t.Fatalf("the settings approval was not asked for a value flag: %v", asked)
	}
	if approved[ConfigChange] || !slices.Contains(declined, string(ConfigChange)) {
		t.Errorf("a declined settings approval is not declined: approved=%v declined=%v", approved, declined)
	}
	asked = nil
	resolveApproval(gaps, InstallOptions{}, nil, confirmRecorder{asked: &asked})
	if asksCategory(asked, ConfigChange) {
		t.Errorf("the settings approval was asked with no flag and no gap: %v", asked)
	}
}

// TestADeclinedApprovalDropsTheValueFlag is the reporter's run: every asked
// approval answered later. The flag is dropped and named, nothing is saved,
// and the block stays where the saved setting puts it.
func TestADeclinedApprovalDropsTheValueFlag(t *testing.T) {
	setupHermetic(t)
	repo := flagOnlyRepo(t)
	opts := installOptsWithout()
	opts.Yes = false
	opts.ValueOverrides = map[string]string{"docs_target": "agents_md"}
	var asked []string
	res, err := Install(repo, opts, confirmRecorder{asked: &asked})
	if err != nil {
		t.Fatal(err)
	}
	if !asksCategory(asked, ConfigChange) {
		t.Fatalf("the settings approval was not asked: %v", asked)
	}
	if v := savedDocsTarget(t, repo); v != "both" {
		t.Errorf("saved docs.target = %q, want both: a declined approval saves nothing", v)
	}
	var named bool
	for _, n := range res.Notes {
		named = named || strings.HasPrefix(n, "--docs-target agents_md was not applied")
	}
	if !named {
		t.Errorf("no note names the dropped --docs-target agents_md: %v", res.Notes)
	}
	for _, c := range res.Changes {
		if strings.Contains(c, "docs_target") {
			t.Errorf("the receipt reports %q, a change that did not land", c)
		}
	}
	for _, w := range res.Writes {
		if w == ".abcd/config.json" || w == "AGENTS.md" || w == "CLAUDE.md" {
			t.Errorf("a run that declined every approval wrote %s", w)
		}
	}
}

// TestAnApprovedSettingsChangeAppliesTheValueFlag: --yes, or an explicit
// approval of the settings change, applies the flag.
func TestAnApprovedSettingsChangeAppliesTheValueFlag(t *testing.T) {
	for name, opts := range map[string]func(InstallOptions) InstallOptions{
		"--yes": func(o InstallOptions) InstallOptions { o.Yes = true; return o },
		"approve.config-change": func(o InstallOptions) InstallOptions {
			o.Yes = false
			o.ApprovedCategories = map[GapCategory]bool{ConfigChange: true}
			return o
		},
	} {
		t.Run(name, func(t *testing.T) {
			setupHermetic(t)
			repo := flagOnlyRepo(t)
			o := installOptsWithout()
			o.ValueOverrides = map[string]string{"docs_target": "agents_md"}
			if _, err := Install(repo, opts(o), RefusingPrompter{}); err != nil {
				t.Fatal(err)
			}
			if v := savedDocsTarget(t, repo); v != "agents_md" {
				t.Errorf("saved docs.target = %q, want agents_md", v)
			}
		})
	}
}

// TestTheSummaryNeverSaysDeclinedAndSaved: a run that declines the settings
// change while it creates abcd's starter files must not tell the person both
// that they declined saving the settings and that the settings were saved.
func TestTheSummaryNeverSaysDeclinedAndSaved(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, "")
	repo := committedRepo(t)
	opts := installOptsWithout()
	opts.Yes = false
	opts.ValueOverrides = map[string]string{"docs_target": "agents_md"}
	var asked []string
	res, err := Install(repo, opts, confirmRecorder{asked: &asked, yes: []GapCategory{SafeAutocreate}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(res.DeclinedCategories, string(ConfigChange)) {
		t.Fatalf("precondition: the settings change was declined (declined %v)", res.DeclinedCategories)
	}
	if !slices.Contains(res.Writes, ".abcd/config.json") {
		t.Fatalf("precondition: the starter files were created (writes %v)", res.Writes)
	}
	for _, it := range res.Summary {
		if it.What == writeKindHelp[writeSettings].What {
			t.Errorf("the summary says the settings were saved beside their decline: %+v", res.Summary)
		}
	}
}
