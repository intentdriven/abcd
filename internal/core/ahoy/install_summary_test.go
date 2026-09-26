package ahoy

import (
	"regexp"
	"strings"
	"testing"
)

// envName matches a raw environment-variable name (ABCD_PLUGIN_ROOT,
// CLAUDE_PLUGIN_ROOT): an all-caps word joined by underscores.
var envName = regexp.MustCompile(`\b[A-Z][A-Z0-9]*_[A-Z0-9_]+\b`)

// insiderWords are the implementer's vocabulary iss-164 found leading the
// completion summary. A person reading what the install did should meet none of
// it: each names a mechanism, not what changed for them.
var insiderWords = []string{"managed repo", "marker block", "canonical", "identity gate", "plugin root", "machine-scope", "gap", "history index", "registry"}

// assertPlainItem holds one summary item to iss-164's frame: what this is, why
// it matters, what (if anything) to do — each present, and each free of raw
// environment names and insider vocabulary.
func assertPlainItem(t *testing.T, it SummaryItem) {
	t.Helper()
	for field, text := range map[string]string{"what": it.What, "why": it.Why, "action": it.Action} {
		if len(strings.Fields(text)) < 3 {
			t.Errorf("item %q: %s is missing or too thin: %q", it.What, field, text)
		}
		if m := envName.FindString(text); m != "" {
			t.Errorf("item %q: %s names the raw environment variable %s: %q", it.What, field, m, text)
		}
		for _, w := range insiderWords {
			if strings.Contains(strings.ToLower(text), w) {
				t.Errorf("item %q: %s uses insider vocabulary %q: %q", it.What, field, w, text)
			}
		}
	}
}

// refsOf collects every reference the summary explains.
func refsOf(items []SummaryItem) map[string]bool {
	out := map[string]bool{}
	for _, it := range items {
		for _, r := range it.Refs {
			out[r] = true
		}
	}
	return out
}

// TestInstallSummaryExplainsEveryWrite is iss-164's detector for the write
// half: every artefact a first install reports writing is explained by a
// summary item in plain language, and the result leads with a sentence saying
// what the run amounted to.
func TestInstallSummaryExplainsEveryWrite(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	idMustGit(t, repo, "init")
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Writes) == 0 {
		t.Fatal("precondition: a first install writes something")
	}
	if len(strings.Fields(res.Headline)) < 4 {
		t.Errorf("headline %q does not say what the run amounted to", res.Headline)
	}
	refs := refsOf(res.Summary)
	for _, w := range res.Writes {
		if !refs[w] {
			t.Errorf("write %q is reported with no plain-language explanation", w)
		}
	}
	for _, id := range res.OptionalSkipped {
		if !refs[id] {
			t.Errorf("optional step %q left undone is reported with no explanation", id)
		}
	}
	for _, it := range res.Summary {
		assertPlainItem(t, it)
	}
}

// TestInstallSummaryExplainsDeclinedAndRemainingWork covers the other half: a
// run the person declined says, per declined kind of change, what was not done
// and how to do it later, and the required work still outstanding is named as
// such rather than as a bare list of identifiers.
func TestInstallSummaryExplainsDeclinedAndRemainingWork(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	idMustGit(t, repo, "init")
	adopt := true
	res, err := Install(repo, InstallOptions{Adopt: &adopt}, stubPrompter{confirm: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.DeclinedCategories) == 0 || len(res.Remaining) == 0 {
		t.Fatalf("precondition: declining every question leaves declined and remaining work; got %+v", res)
	}
	refs := refsOf(res.Summary)
	for _, c := range res.DeclinedCategories {
		if !refs[c] {
			t.Errorf("declined category %q is reported with no explanation", c)
		}
	}
	for _, id := range res.Remaining {
		if !refs[id] {
			t.Errorf("outstanding step %q is reported with no explanation", id)
		}
	}
	for _, it := range res.Summary {
		assertPlainItem(t, it)
	}
}

// TestEveryWriteKindIsExplained keeps the table whole: a kind of write added
// without an explanation would reach the person as a bare path.
func TestEveryWriteKindIsExplained(t *testing.T) {
	for _, k := range allWriteKinds {
		it, ok := writeKindHelp[k]
		if !ok {
			t.Errorf("write kind %q has no explanation", k)
			continue
		}
		assertPlainItem(t, it)
	}
	for _, c := range []GapCategory{SafeAutocreate, ConfigChange, PluginOwned, Dependency, UserState, StatusLine, OracleRouting} {
		it, ok := declinedCategoryHelp[c]
		if !ok {
			t.Errorf("category %q has no explanation for when it is declined", c)
			continue
		}
		assertPlainItem(t, it)
	}
	for _, id := range []string{OptionalPinGapID, StatusLineOfferGapID, OracleRoutingMachineGapID, OracleRoutingRepoGapID} {
		it, ok := optionalSkippedHelp[id]
		if !ok {
			t.Errorf("optional step %q has no explanation", id)
			continue
		}
		assertPlainItem(t, it)
	}
}

// TestPluginFilesMissingGapSpeaksPlainly is the detection half iss-164 quotes:
// the gap a person meets when abcd cannot find its plugin files led with
// "plugin root not resolvable" and two raw environment names. The title and
// detail say what is wrong in plain language; the fix hint may name the
// setting, but only with what it is.
func TestPluginFilesMissingGapSpeaksPlainly(t *testing.T) {
	g := detectPluginRoot(false)[0]
	for field, text := range map[string]string{"title": g.Title, "detail": g.Detail} {
		if m := envName.FindString(text); m != "" {
			t.Errorf("%s names the raw environment variable %s: %q", field, m, text)
		}
		if strings.Contains(strings.ToLower(text), "plugin root") {
			t.Errorf("%s uses insider vocabulary: %q", field, text)
		}
	}
}
