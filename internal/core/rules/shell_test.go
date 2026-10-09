package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The SHELL domain is itd-103's teaching plane (spc-16, "Two planes, one
// registry"; iss-151, ruling J10): the rules loader injects the shell-hazard
// registry's lessons before shell-heavy work, so a host without hook support
// still gets taught what a host with hooks would refuse. Its rules and recall
// terms are GENERATED from the bundled registry the guard reads, never written
// in defaults/rules.json, so the two planes cannot drift.

// TestShellDomainIsGeneratedFromTheGuardRegistry is the drift test: the bundled
// SHELL domain carries exactly one rule per bundled registry entry — the
// entry's lesson, in id order — and recalls on exactly the registry's command
// heads, so a registry entry added or removed changes the domain with no
// second edit, and an edit to the domain that is not an edit to the registry
// fails here.
func TestShellDomainIsGeneratedFromTheGuardRegistry(t *testing.T) {
	d, ok := Defaults().Domains[ShellDomain]
	if !ok {
		t.Fatalf("the %s domain is not bundled: the teaching plane of itd-103 is missing", ShellDomain)
	}
	if d.State == StateDormant {
		t.Fatalf("the %s domain ships dormant: it would teach only on *%s", ShellDomain, ShellDomain)
	}
	reg := guard.Defaults()
	if want := append(reg.Lessons(), reg.ScriptLesson()); !reflect.DeepEqual(d.Rules, want) {
		t.Errorf("%s rules drifted from the guard registry:\n got %q\nwant %q", ShellDomain, d.Rules, want)
	}
	if want := reg.RecallTerms(); !reflect.DeepEqual(d.Recall, want) {
		t.Errorf("%s recall drifted from the guard registry:\n got %q\nwant %q", ShellDomain, d.Recall, want)
	}
	if !reflect.DeepEqual(d.Aliases, shellAliases) {
		t.Errorf("%s aliases = %q, want the fixed list %q", ShellDomain, d.Aliases, shellAliases)
	}
}

// TestShellDomainIsNotHandWritten: the embedded rules.json never declares the
// SHELL domain, so there is no second copy of the registry's words to fall out
// of step with the first.
func TestShellDomainIsNotHandWritten(t *testing.T) {
	var raw struct {
		Domains map[string]json.RawMessage `json:"domains"`
	}
	if err := json.Unmarshal(defaultsJSON, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw.Domains[ShellDomain]; ok {
		t.Fatalf("defaults/rules.json declares %s by hand; it is generated from the guard registry", ShellDomain)
	}
}

// TestShellDomainFollowsRegistryEdits: the generator reads whatever registry
// it is handed, so adding an entry adds its lesson and its recall term, and
// removing one removes both.
func TestShellDomainFollowsRegistryEdits(t *testing.T) {
	reg := guard.Defaults()
	base, ok := shellDomain(reg, reg)
	if !ok {
		t.Fatal("the bundled registry generated no domain")
	}

	added := guard.Defaults()
	added.Entries["zz-shred-disk"] = guard.Entry{
		ID:        "zz-shred-disk",
		Pattern:   guard.Pattern{Command: "shred", Flags: []string{"-u"}},
		Tier:      guard.TierBlocker,
		Successor: "Delete the one file you mean with rm.",
		Why:       "Shredding cannot be undone.",
	}
	grown, _ := shellDomain(added, added)
	if len(grown.Rules) != len(base.Rules)+1 {
		t.Fatalf("adding an entry gave %d rules, want %d", len(grown.Rules), len(base.Rules)+1)
	}
	// The entries' lessons come first, in id order; the guard's script rule
	// closes the list.
	if last := grown.Rules[len(grown.Rules)-2]; !strings.Contains(last, "(zz-shred-disk)") || !strings.Contains(last, "Shredding cannot be undone.") {
		t.Errorf("the added entry's lesson is missing or out of id order: %q", last)
	}
	if !holds(grown.Recall, "shred") {
		t.Errorf("the added entry's command did not become a recall term: %q", grown.Recall)
	}

	removed := guard.Defaults()
	delete(removed.Entries, "git-clean")
	shrunk, _ := shellDomain(removed, removed)
	if len(shrunk.Rules) != len(base.Rules)-1 {
		t.Fatalf("removing an entry gave %d rules, want %d", len(shrunk.Rules), len(base.Rules)-1)
	}
	for _, r := range shrunk.Rules {
		if strings.Contains(r, "(git-clean)") {
			t.Fatalf("the removed entry still teaches: %q", r)
		}
	}
	if holds(shrunk.Recall, "git clean") {
		t.Errorf("the removed entry's recall term survived: %q", shrunk.Recall)
	}

	// An empty registry generates no domain at all, never a heading-only one.
	if _, ok := shellDomain(guard.Registry{SchemaVersion: guard.SchemaVersion}, guard.Defaults()); ok {
		t.Fatal("an empty registry generated a domain; it would render as a heading with nothing under it")
	}
}

func holds(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// TestShellDomainRecallsShellHeavyPrompts: a prompt about shell-heavy work
// recalls the domain; ordinary prompts that merely share a word do not.
func TestShellDomainRecallsShellHeavyPrompts(t *testing.T) {
	rs := Defaults()
	shellHeavy := []string{
		"clean up the scratch directory with rm -rf",
		"cd into build and rm the old output",
		"git push the branch when the tests pass",
		"git reset the worktree to origin",
		"stop it with kill $(pgrep -f make) for now",
		"pgrep -f node | xargs kill",
		// A signal flag between the everyday verb and the substitution.
		"kill -9 $(pgrep make)",
		"kill -9 $(pidof node)",
		"pkill the dev server",
		"write a bash script that loops over the fixtures",
		"run it in the shell",
		"force push the rebased branch",
		"gh repo delete the throwaway fork",
	}
	// An entry whose head is marked an everyday verb recalls only through the
	// shapes its pattern declares, so every command line it is proven to fire
	// on must still recall the domain that teaches it.
	for _, e := range guard.Defaults().Entries {
		if e.EverydayVerb != nil && *e.EverydayVerb {
			shellHeavy = append(shellHeavy, e.Fixtures.KnownBad...)
		}
	}
	for _, prompt := range shellHeavy {
		if !has(rs.Match(prompt), ShellDomain) {
			t.Errorf("shell-heavy prompt %q did not recall %s, got %v", prompt, ShellDomain, names(rs.Match(prompt)))
		}
	}
	for _, prompt := range []string{
		"update the configuration chapter",
		"commit the change and open a pull request",
		"push the release notes to the site",
		"resolve the issue with a note",
		"the reset button on the form",
		// kill is an everyday verb: the registry marks it so, and the bare
		// word no longer recalls the domain (iss-2609300123431381).
		"kill the feature flag, we are not shipping it",
		"kill the stale server process",
	} {
		if has(rs.Match(prompt), ShellDomain) {
			t.Errorf("ordinary prompt %q recalled %s", prompt, ShellDomain)
		}
	}
}

// TestShellDomainObeysTheLoaderContracts: the generated domain is an ordinary
// bundled domain to every loader contract — per-field overrides with
// provenance, dormant, the star command, the kill switch, and the withheld-
// guardrail note.
func TestShellDomainObeysTheLoaderContracts(t *testing.T) {
	prompt := "rm -rf the build directory"

	t.Run("bundled provenance", func(t *testing.T) {
		d, ok := Defaults().Lookup(ShellDomain)
		if !ok || d.Source != SourceBundled {
			t.Fatalf("Lookup(%s) = %+v, %v; want a bundled domain", ShellDomain, d, ok)
		}
		if !strings.HasPrefix(Render([]ResolvedDomain{d}), "# abcd rules — 1 domain(s) active\n## SHELL\n- Refused by the guard") {
			t.Fatalf("the bundled domain does not render bare:\n%s", Render([]ResolvedDomain{d}))
		}
	})

	t.Run("dormant silences, star activates", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoRules(t, dir, `{"schema_version":1,"domains":{"SHELL":{"state":"dormant"}}}`)
		rs := mustLoad(t, dir)
		if has(rs.Match(prompt), ShellDomain) {
			t.Fatal("a dormant SHELL still recalled")
		}
		got := rs.Match("*SHELL " + prompt)
		if !has(got, ShellDomain) {
			t.Fatal("*SHELL did not activate the dormant domain")
		}
		d, _ := rs.Lookup(ShellDomain)
		if d.Source != SourceRepo || len(d.Rules) != len(Defaults().Domains[ShellDomain].Rules) {
			t.Fatalf("a state-only override should keep the generated rules and read as the repo's: %+v", d)
		}
	})

	t.Run("kill switch", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoRules(t, dir, `{"schema_version":1,"disabled":true,"domains":{}}`)
		rs := mustLoad(t, dir)
		if got := rs.Match("*SHELL " + prompt); len(got) != 0 {
			t.Fatalf("the kill switch let %v through", names(got))
		}
	})

	t.Run("rules override names the withheld lessons", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoRules(t, dir, `{"schema_version":1,"domains":{"SHELL":{"rules":["Ask before deleting anything."]}}}`)
		rs := mustLoad(t, dir)
		d, _ := rs.Lookup(ShellDomain)
		if d.Source != SourceRepo || !reflect.DeepEqual(d.Rules, []string{"Ask before deleting anything."}) {
			t.Fatalf("the repo's rules did not replace the generated list: %+v", d)
		}
		if !strings.HasPrefix(Render([]ResolvedDomain{d}), "# abcd rules — 1 domain(s) active\n## SHELL (repo override)\n") {
			t.Fatalf("the overridden domain does not name its layer:\n%s", Render([]ResolvedDomain{d}))
		}
		var note string
		for _, n := range rs.Notes() {
			if strings.Contains(n, `"SHELL"`) && strings.Contains(n, "WITHHOLDS") {
				note = n
			}
		}
		if note == "" || !strings.Contains(note, "(git-push-force)") {
			t.Fatalf("replacing the SHELL lessons withheld them silently; notes: %q", rs.Notes())
		}
	})

	t.Run("render is stable", func(t *testing.T) {
		a, _ := Defaults().Lookup(ShellDomain)
		b, _ := mustLoad(t, t.TempDir()).Lookup(ShellDomain)
		if Signature(a) != Signature(b) {
			t.Fatal("two loads of the generated domain sign differently; dedup would re-inject it every prompt")
		}
	})
}

func mustLoad(t *testing.T, dir string) RuleSet {
	t.Helper()
	rs, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return rs
}

// repoGuardEntry is a repository's own hazard, declared in its
// .abcd/guard.json, and repoGuardLesson the one rule it teaches: the text is
// pinned here so a change to the generator's wording is a change someone saw.
const (
	repoGuardEntry = `{"schema_version":1,"entries":{"deploy-prod":{
		"tier":"blocker",
		"pattern":{"command":"make","subcommand":"deploy"},
		"why":"It deploys to production from a laptop.",
		"successor":"Open a release pull request; CI deploys it."}}}`
	repoGuardLesson = "Refused by the guard (deploy-prod) (repo): `make deploy`. It deploys to production from a laptop. Instead: Open a release pull request; CI deploys it."
)

func writeRepoGuard(t *testing.T, dir, body string) {
	t.Helper()
	abcd := filepath.Join(dir, ".abcd")
	if err := os.MkdirAll(abcd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(abcd, "guard.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestShellDomainTeachesTheRepositorysOwnGuardEntries (ruling CK1): an entry
// a repository adds in its .abcd/guard.json is taught in SHELL by the same
// generator as the bundled ones — its lesson in id order among them, marked
// "(repo)", and its command head a recall term — whether or not the
// repository also carries a rules.json.
func TestShellDomainTeachesTheRepositorysOwnGuardEntries(t *testing.T) {
	bundled := Defaults().Domains[ShellDomain]
	for _, withRules := range []bool{false, true} {
		dir := t.TempDir()
		writeRepoGuard(t, dir, repoGuardEntry)
		if withRules {
			writeRepoRules(t, dir, `{"schema_version":1,"domains":{}}`)
		}
		rs := mustLoad(t, dir)
		d, ok := rs.Lookup(ShellDomain)
		if !ok {
			t.Fatalf("rules.json=%v: no %s domain", withRules, ShellDomain)
		}
		if !holds(d.Rules, repoGuardLesson) {
			t.Errorf("rules.json=%v: the repository's own entry is not taught as\n %q\namong the %d rules", withRules, repoGuardLesson, len(d.Rules))
		}
		if len(d.Rules) != len(bundled.Rules)+1 {
			t.Errorf("rules.json=%v: %d lessons, want the %d bundled ones plus the repository's", withRules, len(d.Rules), len(bundled.Rules))
		}
		for _, l := range bundled.Rules {
			if !holds(d.Rules, l) {
				t.Errorf("rules.json=%v: a bundled lesson is missing or marked: %q", withRules, l)
			}
		}
		if !holds(d.Recall, "make deploy") {
			t.Errorf("rules.json=%v: the entry's command head is not a recall term: %q", withRules, d.Recall)
		}
		if !has(rs.Match("make deploy the docs site"), ShellDomain) {
			t.Errorf("rules.json=%v: a prompt naming the repository's hazard did not recall %s", withRules, ShellDomain)
		}
		if d.Source != SourceBundled {
			t.Errorf("rules.json=%v: source %q; the domain is still the generated one, its repository words marked per lesson", withRules, d.Source)
		}
		if n := rs.Notes(); len(n) != 0 {
			t.Errorf("rules.json=%v: a valid guard.json produced notes: %q", withRules, n)
		}
	}
}

// TestShellDomainRefusesAnInvalidRepoGuardEntryLoudly (ruling CK1): a
// repository guard.json the guard refuses is refused here too, never taught
// and never silently skipped. SHELL teaches the registry the guard enforces in
// its place (the bundled one), the rest of the rule set loads, and a note
// names the file and the reason on every load.
func TestShellDomainRefusesAnInvalidRepoGuardEntryLoudly(t *testing.T) {
	dir := t.TempDir()
	writeRepoGuard(t, dir, `{"schema_version":1,"entries":{"deploy-prod":{
		"tier":"blocker",
		"pattern":{"command":"make","subcommand":"deploy"},
		"successor":"Open a release pull request; CI deploys it."}}}`)
	rs := mustLoad(t, dir)
	d, _ := rs.Lookup(ShellDomain)
	if want := Defaults().Domains[ShellDomain].Rules; !reflect.DeepEqual(d.Rules, want) {
		t.Errorf("an invalid repository entry changed what SHELL teaches: %d rules, want the %d bundled ones", len(d.Rules), len(want))
	}
	for _, r := range d.Rules {
		if strings.Contains(r, "deploy-prod") {
			t.Errorf("the refused entry is taught: %q", r)
		}
	}
	if holds(d.Recall, "make deploy") {
		t.Errorf("the refused entry's head is a recall term: %q", d.Recall)
	}
	if _, ok := rs.Lookup("PII"); !ok {
		t.Error("a refused guard.json took the rest of the rule set with it")
	}
	var note string
	for _, n := range rs.Notes() {
		if strings.Contains(n, ShellDomain) && strings.Contains(n, ".abcd/guard.json") {
			note = n
		}
	}
	if note == "" || !strings.Contains(note, "deploy-prod has no why") || !strings.Contains(note, "refused") {
		t.Fatalf("the refused guard.json is not named loudly; notes: %q", rs.Notes())
	}
}

// guardEntries renders a guard.json with n entries of its own, each teaching a
// why and a successor of size bytes: a registry within every bound that still
// teaches more than the injection budget holds.
func guardEntries(n, size int) string {
	var b strings.Builder
	b.WriteString(`{"schema_version":1,"entries":{`)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `"deploy-%d":{"tier":"blocker","pattern":{"command":"deploy%d"},"why":%q,"successor":%q}`,
			i, i, strings.Repeat("w", size), strings.Repeat("s", size))
	}
	b.WriteString("}}")
	return b.String()
}

// TestShellDomainRefusesAnOverlongRepoLesson: a lesson is injected word for
// word, so a repository entry whose why passes the guard's per-field bound is
// refused like any invalid entry — named on the load's notes, never taught —
// rather than handed to the injection budget to crowd every other domain out.
func TestShellDomainRefusesAnOverlongRepoLesson(t *testing.T) {
	dir := t.TempDir()
	writeRepoGuard(t, dir, fmt.Sprintf(`{"schema_version":1,"entries":{"deploy-prod":{
		"tier":"blocker",
		"pattern":{"command":"make","subcommand":"deploy"},
		"why":%q,
		"successor":"Open a release pull request; CI deploys it."}}}`, strings.Repeat("w", 240*1000)))
	rs := mustLoad(t, dir)
	d, _ := rs.Lookup(ShellDomain)
	if want := Defaults().Domains[ShellDomain].Rules; !reflect.DeepEqual(d.Rules, want) {
		t.Errorf("an overlong repository entry changed what SHELL teaches: %d rules, want the %d bundled ones", len(d.Rules), len(want))
	}
	var note string
	for _, n := range rs.Notes() {
		if strings.Contains(n, ".abcd/guard.json") && strings.Contains(n, "deploy-prod has a why of") {
			note = n
		}
	}
	if note == "" {
		t.Fatalf("the overlong guard.json is not refused loudly; notes: %q", rs.Notes())
	}
}

// TestInjectionBudgetNamesTheFileThatOverflowedIt: when the injection budget
// drops a domain, the trailer names the file whose words filled it — the
// repository's guard.json when SHELL's own lessons did, its rules.json when a
// domain it declared did — never always .abcd/rules.json.
func TestInjectionBudgetNamesTheFileThatOverflowedIt(t *testing.T) {
	const prompt = "commit this in the shell and keep the pii out"
	t.Run("guard.json", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoGuard(t, dir, guardEntries(40, 1000))
		rs := mustLoad(t, dir)
		if n := rs.Notes(); len(n) != 0 {
			t.Fatalf("a guard.json within every bound produced notes: %q", n)
		}
		res := Inject(rs, prompt, SessionState{}, 0)
		trailer := res.Text[strings.LastIndex(res.Text, "\n# abcd rules"):]
		if !strings.Contains(trailer, injectionTruncatedMarker) {
			t.Fatalf("40 long repository lessons did not overflow the budget:\n%.300s", res.Text)
		}
		if !strings.Contains(trailer, guard.RepoRelPath) || strings.Contains(trailer, RepoRelPath) {
			t.Errorf("the trailer does not blame the repository's guard.json alone:\n%s", trailer)
		}
	})
	t.Run("rules.json", func(t *testing.T) {
		dir := t.TempDir()
		big := strings.Repeat("x", 20*1024)
		writeRepoRules(t, dir, fmt.Sprintf(`{"schema_version":1,"domains":{"FLOOD":{"recall":["shell"],"rules":[%q,%q,%q,%q]}}}`, big, big, big, big))
		res := Inject(mustLoad(t, dir), prompt, SessionState{}, 0)
		trailer := res.Text[strings.LastIndex(res.Text, "\n# abcd rules"):]
		if !strings.Contains(trailer, injectionTruncatedMarker) {
			t.Fatalf("an 80 KiB domain did not overflow the budget:\n%.300s", res.Text)
		}
		if !strings.Contains(trailer, RepoRelPath) || strings.Contains(trailer, guard.RepoRelPath) {
			t.Errorf("the trailer does not blame the repository's rules.json alone:\n%s", trailer)
		}
	})
}

// TestShellLessonsUnderACommittedDisabledGuardSayItIsOff: a committed
// "disabled": true guard.json refuses nothing, so SHELL still teaches every
// hazard (the switches are independent, spc-16) but never as "Refused by the
// guard": the bundled lessons and the repository's own alike open with the
// guard-off lead.
func TestShellLessonsUnderACommittedDisabledGuardSayItIsOff(t *testing.T) {
	for name, body := range map[string]string{
		"bundled": `{"schema_version":1,"disabled":true}`,
		"repo": `{"schema_version":1,"disabled":true,"entries":{"deploy-prod":{
		"tier":"blocker",
		"pattern":{"command":"make","subcommand":"deploy"},
		"why":"It deploys to production from a laptop.",
		"successor":"Open a release pull request; CI deploys it."}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			repo := gittest.NewRepo(t)
			repo.Write(guard.RepoRelPath, body)
			repo.Commit("disable the guard")
			rs := mustLoad(t, repo.Root())
			if n := rs.Notes(); len(n) != 0 {
				t.Fatalf("a committed kill switch produced notes: %q", n)
			}
			d, _ := rs.Lookup(ShellDomain)
			if len(d.Rules) < len(Defaults().Domains[ShellDomain].Rules) {
				t.Fatalf("a disabled guard stopped SHELL teaching: %d rules", len(d.Rules))
			}
			for _, r := range d.Rules {
				if !strings.HasPrefix(r, "Hazard (guard off) (") {
					t.Errorf("a lesson under a disabled guard does not say the guard is off: %q", r)
				}
			}
			if name == "repo" && !holds(d.Rules, strings.Replace(repoGuardLesson, "Refused by the guard", "Hazard (guard off)", 1)) {
				t.Errorf("the repository's own lesson is not taught under the guard-off lead: %q", d.Rules)
			}
		})
	}
}

// TestWithheldNoteNamesTheRepositorysOwnLessons: a rules.json SHELL override
// that replaces the rules list withholds every lesson SHELL was built with,
// the repository's own guard entries included, and the note names them. It
// compares against the base this load built, not only the bundled list.
func TestWithheldNoteNamesTheRepositorysOwnLessons(t *testing.T) {
	dir := t.TempDir()
	writeRepoGuard(t, dir, repoGuardEntry)
	writeRepoRules(t, dir, `{"schema_version":1,"domains":{"SHELL":{"rules":["Ask before deleting anything."]}}}`)
	rs := mustLoad(t, dir)
	var note string
	for _, n := range rs.Notes() {
		if strings.Contains(n, `"SHELL"`) && strings.Contains(n, "WITHHOLDS") {
			note = n
		}
	}
	if note == "" {
		t.Fatalf("no withheld note for SHELL; notes: %q", rs.Notes())
	}
	total := len(Defaults().Domains[ShellDomain].Rules) + 1
	if !strings.Contains(note, "(deploy-prod) (repo)") {
		t.Errorf("the note does not name the repository's own withheld lesson: %s", note)
	}
	if !strings.Contains(note, fmt.Sprintf("WITHHOLDS %d of its %d", total, total)) {
		t.Errorf("the note does not count the repository's lesson among the %d withheld: %s", total, note)
	}
	if !strings.Contains(note, guard.RepoRelPath) {
		t.Errorf("the note says only what abcd ships, not the repository's guard.json: %s", note)
	}
}
