package ahoy

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// valueKeyPrompter approves everything and answers each value question from
// answers, else with its default, keeping the config value keys it was asked
// in order.
type valueKeyPrompter struct {
	answers map[string]string
	keys    []string
}

func (p *valueKeyPrompter) Confirm(string) bool { return true }

func (p *valueKeyPrompter) Prompt(key string, _ []string, def string) string {
	if _, config := fixedValueChoices[key]; config && key != emDashPromptKey {
		p.keys = append(p.keys, key)
	}
	if v, ok := p.answers[key]; ok {
		return v
	}
	return def
}

// treeOf lists every path under root with its size, for a wrote-nothing check.
func treeOf(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil {
			rel, _ := filepath.Rel(root, p)
			fmt.Fprintf(&b, "%s %d\n", rel, info.Size())
		}
		return nil
	})
	return b.String()
}

// trufflehogResolves makes PATH resolve trufflehog, or not, whatever the
// machine has installed: the deep-scan question is asked only where it does.
func trufflehogResolves(t *testing.T, on bool) {
	t.Helper()
	var kept []string
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if _, err := os.Stat(filepath.Join(dir, "trufflehog")); err != nil {
			kept = append(kept, dir)
		}
	}
	if on {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "trufflehog"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		kept = append([]string{dir}, kept...)
	}
	t.Setenv("PATH", strings.Join(kept, string(os.PathListSeparator)))
}

// TestWalkPutsTheValueQuestionsTheInstallAsks holds the walk to the install
// it runs ahead of: given the same answers, it puts the same config value
// questions in the same order, stopping where the install's collection
// stops, and writes nothing.
func TestWalkPutsTheValueQuestionsTheInstallAsks(t *testing.T) {
	adopt := true
	for name, c := range map[string]struct {
		answers    map[string]string
		overrides  map[string]string
		trufflehog bool
		want       []string
	}{
		"every value":              {answers: map[string]string{"visibility": "private", "docs_target": "agents_md"}, want: []string{"visibility", "docs_target"}},
		"deep scan asked":          {answers: map[string]string{"visibility": "private"}, trufflehog: true, want: []string{"visibility", "docs_target", "scan_deep"}},
		"visibility decided later": {answers: map[string]string{"visibility": ""}, want: []string{"visibility"}},
		"a value it does not take": {answers: map[string]string{"visibility": "public", "docs_target": "both"}, want: []string{"visibility", "docs_target"}},
		"a flag gives visibility":  {overrides: map[string]string{"visibility": "public"}, want: []string{"docs_target"}},
	} {
		t.Run(name, func(t *testing.T) {
			setupHermetic(t)
			trufflehogResolves(t, c.trufflehog)
			repo := t.TempDir()
			idMustGit(t, repo, "init")
			opts := InstallOptions{Adopt: &adopt, Yes: true, ValueOverrides: c.overrides}
			before := treeOf(t, repo)
			walked := &valueKeyPrompter{answers: c.answers}
			if err := WalkConfigValueQuestions(repo, opts, func(string) bool { return false }, walked.Prompt); err != nil {
				t.Fatal(err)
			}
			if after := treeOf(t, repo); after != before {
				t.Fatalf("the walk wrote into the repository:\n--- before\n%s--- after\n%s", before, after)
			}
			installed := &valueKeyPrompter{answers: c.answers}
			if _, err := Install(repo, opts, installed); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(walked.keys, c.want) || !slices.Equal(installed.keys, c.want) {
				t.Fatalf("walked %v, the install asked %v; want %v", walked.keys, installed.keys, c.want)
			}
		})
	}
}

// TestWalkPutsNothingTheInstallWouldNotAsk: a declined adoption, an
// unapproved settings change, or a repository with every value saved puts
// no value question; the approvals answered by approves do.
func TestWalkPutsNothingTheInstallWouldNotAsk(t *testing.T) {
	setupHermetic(t)
	trufflehogResolves(t, false)
	repo := t.TempDir()
	idMustGit(t, repo, "init")
	no, yes := false, true
	walk := func(opts InstallOptions, approved ...string) []string {
		t.Helper()
		p := &valueKeyPrompter{answers: map[string]string{"visibility": "public"}}
		if err := WalkConfigValueQuestions(repo, opts, func(id string) bool { return slices.Contains(approved, id) }, p.Prompt); err != nil {
			t.Fatal(err)
		}
		return p.keys
	}
	for name, got := range map[string][]string{
		"adoption refused by flag":   walk(InstallOptions{Adopt: &no, Yes: true}),
		"adoption declined":          walk(InstallOptions{Yes: true}),
		"settings change declined":   walk(InstallOptions{Adopt: &yes}),
		"settings change not chosen": walk(InstallOptions{Adopt: &yes, ApprovedCategories: map[GapCategory]bool{SafeAutocreate: true}}, "approve.config-change"),
	} {
		if len(got) != 0 {
			t.Errorf("%s: put %v", name, got)
		}
	}
	if got := walk(InstallOptions{}, "adopt", "approve.config-change"); !slices.Equal(got, []string{"visibility", "docs_target"}) {
		t.Fatalf("adoption and settings change approved: put %v", got)
	}
	if _, err := Install(repo, InstallOptions{Adopt: &yes, Yes: true}, &valueKeyPrompter{answers: map[string]string{"visibility": "public"}}); err != nil {
		t.Fatal(err)
	}
	if got := walk(InstallOptions{Yes: true}); len(got) != 0 {
		t.Fatalf("a repository with every value saved: put %v", got)
	}
	// A flag that turns the saved visibility private makes the deep-scan
	// question askable with no gap for it, as the install asks it, once the
	// settings change the flag puts is approved (iss-2610071538032843).
	trufflehogResolves(t, true)
	private := InstallOptions{Adopt: &yes, ValueOverrides: map[string]string{"visibility": "private"}}
	if got := walk(private); len(got) != 0 {
		t.Fatalf("a flag changing a saved value with the settings change not approved: put %v", got)
	}
	got := walk(private, "approve.config-change")
	installed := &valueKeyPrompter{}
	if _, err := Install(repo, private, installed); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"scan_deep"}) || !slices.Equal(installed.keys, got) {
		t.Fatalf("a flag turning visibility private: walked %v, the install asked %v; want [scan_deep]", got, installed.keys)
	}
}

// TestSetupFixedValuesAreTheQuestionsOwn: the values an answers file may give
// a question with a fixed set are exactly the values its question offers.
func TestSetupFixedValuesAreTheQuestionsOwn(t *testing.T) {
	for id, want := range map[string]string{
		"visibility":                "private|public|later",
		"docs_target":               "agents_md|skip|later",
		"scan_deep":                 "true|false|later",
		"adopt":                     "yes|no|later",
		"approve.config-change":     "yes|no|later",
		StatusLineOfferGapID:        "yes|no|later",
		OracleRoutingMachineGapID:   "yes|no|later",
		elementPromptPrefix + "git": "on|off|later",
	} {
		got, ok := SetupFixedValues(id)
		if !ok || strings.Join(got, "|") != want {
			t.Errorf("%s: %v, %v; want %s", id, got, ok, want)
		}
	}
	for _, id := range []string{artefactKindKey, "Q7", "visibilty"} {
		if got, ok := SetupFixedValues(id); ok {
			t.Errorf("%s: fixed %v", id, got)
		}
	}
	if !SetupAskedBeforeWriting("adopt") || !SetupAskedBeforeWriting("approve.dependency") || SetupAskedBeforeWriting("visibility") {
		t.Fatal("SetupAskedBeforeWriting names the wrong questions")
	}
}
