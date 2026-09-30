package guard

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// The teaching plane of spc-16 ("Two planes, one registry") renders every
// registry entry as a rule the rules loader injects before shell-heavy work.
// These tests pin the rendering the guard owns: the lesson an entry teaches,
// the command-position description of its pattern, and the recall terms the
// registry offers.

// TestEveryBundledEntryTeachesItsLesson: each bundled entry renders to one
// lesson naming the entry, its tier's consequence, its why and its successor,
// so what an agent is taught up front is what the guard would tell it at the
// moment of refusal.
func TestEveryBundledEntryTeachesItsLesson(t *testing.T) {
	r := Defaults()
	lessons := r.Lessons()
	if len(lessons) != len(r.Entries) {
		t.Fatalf("Lessons() = %d lessons for %d entries: one lesson per entry", len(lessons), len(r.Entries))
	}
	ids := make([]string, 0, len(r.Entries))
	for id := range r.Entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for i, id := range ids {
		e := r.Entries[id]
		got := lessons[i]
		if got != e.Lesson() {
			t.Errorf("lesson %d is not entry %s's (lessons come in id order):\n%s", i, id, got)
		}
		for _, want := range []string{"(" + id + ")", e.Why, "Instead: " + e.Successor, "`" + e.Pattern.Command} {
			if !strings.Contains(got, want) {
				t.Errorf("entry %s lesson does not carry %q:\n%s", id, want, got)
			}
		}
		lead := "Refused by the guard"
		if e.Tier == TierWarn {
			lead = "Warned by the guard"
		}
		if !strings.HasPrefix(got, lead+" ") {
			t.Errorf("entry %s (%s) lesson does not open with %q:\n%s", id, e.Tier, lead, got)
		}
		if strings.Contains(got, "\n") {
			t.Errorf("entry %s lesson spans lines; a rule is one bullet:\n%s", id, got)
		}
	}
}

// TestPatternDescribeReadsAsTheCommand: the description is written over the
// pattern's own fields, so every constraint an entry declares is visible in
// the lesson, in command-position order.
func TestPatternDescribeReadsAsTheCommand(t *testing.T) {
	yes := true
	for _, tc := range []struct {
		name string
		p    Pattern
		want string
	}{
		{"command only", Pattern{Command: "git", Subcommand: "clean"}, "`git clean`"},
		{"two-level subcommand", Pattern{Command: "gh", Subcommand: "repo", Subcommand2: "delete"}, "`gh repo delete`"},
		{"one flag group", Pattern{Command: "git", Subcommand: "reset", Flags: []string{"--hard"}}, "`git reset` with `--hard`"},
		{"alternatives and two groups", Pattern{Command: "rm", Flags: []string{"-r|-R|--recursive", "-f|--force"}, AfterCD: &yes},
			"`rm` with `-r`, `-R` or `--recursive` and `-f` or `--force`, after a `cd`, `pushd` or `popd` earlier in the same chain"},
		{"flag value and path", Pattern{Command: "gh", Subcommand: "api", FlagValues: []FlagValue{{Flag: "-X|--method", Values: []string{"DELETE"}}}, ArgPaths: []PathArg{{Root: "repos", Segments: 3}}},
			"`gh api` with `-X` or `--method` set to `DELETE`, on a `repos/*/*` path"},
		{"operand prefix", Pattern{Command: "git", Subcommand: "push", ArgPrefixes: []string{"+"}}, "`git push` with an operand starting `+`"},
		{"operand count", Pattern{Command: "pkill", MinOperands: 1}, "`pkill` with an operand"},
		{"operand count plural", Pattern{Command: "pkill", MinOperands: 2}, "`pkill` with at least 2 operands"},
		{"args from", Pattern{Command: "kill", ArgsFrom: []Pattern{{Command: "pgrep"}, {Command: "pgrep", MinOperands: 1}, {Command: "pidof"}}},
			"`kill` given pids printed by `pgrep` or `pidof`"},
		{"operand words, capped", Pattern{Command: "rm", Flags: []string{"-r"}, ArgValues: []string{"/", "/*", "~", "~/", "$HOME", "$HOME/", "${HOME}", "${HOME}/"}},
			"`rm` with `-r`, on `/`, `/*`, `~`, `~/`, `$HOME`, `$HOME/` or 2 more spellings like them"},
		{"operand words, few", Pattern{Command: "rm", ArgValues: []string{"*", "."}}, "`rm`, on `*` or `.`"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.p.Describe(); got != tc.want {
				t.Errorf("Describe() =\n  %s\nwant\n  %s", got, tc.want)
			}
		})
	}
}

// TestRecallTermsAreTheCommandHeads: the registry offers its own recall
// vocabulary — the command head each entry matches in command position — so
// an entry for a new command recalls the teaching domain with no second edit.
// The head carries the subcommands, which keeps the recall narrow: `git push`
// recalls it, the bare word "push" does not.
func TestRecallTermsAreTheCommandHeads(t *testing.T) {
	r := Registry{SchemaVersion: SchemaVersion, Entries: map[string]Entry{
		"a": {Pattern: Pattern{Command: "git", Subcommand: "push"}},
		"b": {Pattern: Pattern{Command: "git", Subcommand: "push", Flags: []string{"--no-verify"}}},
		"c": {Pattern: Pattern{Command: "gh", Subcommand: "repo", Subcommand2: "delete"}},
		"d": {Pattern: Pattern{Command: "rm"}},
	}}
	want := []string{"gh repo delete", "git push", "rm"}
	if got := r.RecallTerms(); !reflect.DeepEqual(got, want) {
		t.Fatalf("RecallTerms() = %q, want %q", got, want)
	}
	for _, term := range Defaults().RecallTerms() {
		if strings.TrimSpace(term) == "" {
			t.Fatal("the bundled registry yields an empty recall term")
		}
	}
}

// TestLessonsOverMarkTheRepositorysOwnWords: a repository's .abcd/guard.json
// entries are taught by the same generator as the bundled ones (ruling CK1),
// and every lesson whose words the bundled registry does not teach carries the
// "(repo)" mark, so whose words these are is never invisible: an entry the
// repository added, and a bundled entry it reworded. A change the lesson does
// not show (a fixture) leaves the bundled lesson unmarked.
func TestLessonsOverMarkTheRepositorysOwnWords(t *testing.T) {
	bundled := Defaults()
	if got, want := bundled.LessonsOver(bundled), bundled.Lessons(); !reflect.DeepEqual(got, want) {
		t.Fatalf("the bundled registry over itself marked a lesson:\n got %q\nwant %q", got, want)
	}

	repo := Defaults()
	repo.Entries["deploy-prod"] = Entry{
		ID:        "deploy-prod",
		Pattern:   Pattern{Command: "make", Subcommand: "deploy"},
		Tier:      TierBlocker,
		Why:       "It deploys to production from a laptop.",
		Successor: "Open a release pull request; CI deploys it.",
	}
	clean := repo.Entries["git-clean"]
	clean.Why = "Untracked files here hold the fixtures nobody committed."
	repo.Entries["git-clean"] = clean
	reset := repo.Entries["git-reset-hard"]
	reset.Fixtures.KnownGood = append(reset.Fixtures.KnownGood, "git reset --soft HEAD~1")
	repo.Entries["git-reset-hard"] = reset

	got := map[string]string{}
	for _, l := range repo.LessonsOver(bundled) {
		for id := range repo.Entries {
			if strings.Contains(l, "("+id+")") {
				got[id] = l
			}
		}
	}
	if want := "Refused by the guard (deploy-prod) (repo): `make deploy`. It deploys to production from a laptop. Instead: Open a release pull request; CI deploys it."; got["deploy-prod"] != want {
		t.Errorf("the repository's own entry teaches\n %q\nwant\n %q", got["deploy-prod"], want)
	}
	if !strings.HasPrefix(got["git-clean"], "Warned by the guard (git-clean) (repo): `git clean`.") ||
		!strings.Contains(got["git-clean"], clean.Why) {
		t.Errorf("a bundled entry the repository reworded is not marked as the repository's: %q", got["git-clean"])
	}
	if want := bundled.Entries["git-reset-hard"].Lesson(); got["git-reset-hard"] != want {
		t.Errorf("a fixture-only change marked the bundled lesson:\n got %q\nwant %q", got["git-reset-hard"], want)
	}
	if n := len(repo.LessonsOver(bundled)); n != len(repo.Entries) {
		t.Errorf("LessonsOver gave %d lessons for %d entries", n, len(repo.Entries))
	}
}

// TestLessonsUnderADisabledRegistrySayTheGuardIsOff: a committed
// "disabled": true registry refuses and warns about nothing, so a lesson that
// opened "Refused by the guard" or "Warned by the guard" would teach a false
// sentence. The hazard is still real and still taught (the switches stay
// independent, spc-16), under a lead that says the guard is off — for a
// repository's own entry and for a bundled one alike.
func TestLessonsUnderADisabledRegistrySayTheGuardIsOff(t *testing.T) {
	bundled := Defaults()
	off := Defaults()
	off.Disabled = true
	off.Entries["deploy-prod"] = Entry{
		Pattern:   Pattern{Command: "make", Subcommand: "deploy"},
		Tier:      TierBlocker,
		Why:       "It deploys to production from a laptop.",
		Successor: "Open a release pull request; CI deploys it.",
	}
	got := map[string]string{}
	for _, l := range off.LessonsOver(bundled) {
		if strings.HasPrefix(l, "Refused by the guard") || strings.HasPrefix(l, "Warned by the guard") {
			t.Errorf("a disabled registry teaches the guard as enforcing: %q", l)
		}
		for _, id := range []string{"deploy-prod", "git-push-force", "git-clean"} {
			if strings.Contains(l, "("+id+")") {
				got[id] = l
			}
		}
	}
	if want := "Hazard (guard off) (deploy-prod) (repo): `make deploy`. It deploys to production from a laptop. Instead: Open a release pull request; CI deploys it."; got["deploy-prod"] != want {
		t.Errorf("the repository's own entry under a disabled registry teaches\n %q\nwant\n %q", got["deploy-prod"], want)
	}
	for _, id := range []string{"git-push-force", "git-clean"} {
		if !strings.HasPrefix(got[id], "Hazard (guard off) ("+id+"): ") {
			t.Errorf("the bundled entry %s under a disabled registry teaches %q", id, got[id])
		}
	}
	if n := len(off.Lessons()); n != len(off.Entries) {
		t.Errorf("a disabled registry taught %d lessons for %d entries: the hazards are still taught", n, len(off.Entries))
	}
}
