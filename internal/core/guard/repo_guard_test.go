package guard

import (
	"path/filepath"
	"sort"
	"testing"
)

// repoHookVerbEntries are the entries abcd's own .abcd/guard.json adds: one per
// git verb that runs the repository's hooks and takes --no-verify to skip
// them, beyond the bundled commit and push entries. They are this repository's
// rule (AGENTS.md: never --no-verify), not a bundled default, since another
// repository may use the flag on purpose (iss-2610301330557789).
//
// git cherry-pick has no entry: git has no --no-verify for it (its -n is
// --no-commit), so an entry would teach a flag that does not exist. -n is
// matched only where git reads it as --no-verify: on am, as on commit, and
// never on merge, rebase or pull, where it drops the diffstat.
var repoHookVerbEntries = []string{
	"git-am-no-verify",
	"git-merge-no-verify",
	"git-pull-no-verify",
	"git-rebase-no-verify",
}

// repoRegistry is the registry this checkout's hook enforces: the bundled
// defaults merged with the repository's own .abcd/guard.json.
func repoRegistry(t *testing.T) Registry {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	r, err := load(root)
	if err != nil {
		t.Fatalf("this repository's %s does not load: %v", RepoRelPath, err)
	}
	return r
}

// TestRepoGuardEntriesPassAdmissionGate holds the entries the repository's
// own guard file adds to the bar the bundled ones meet: every known-bad
// fixture fires its entry as a blocker, every known-good fires nothing, and
// known-good is at least 40% of each entry's corpus.
func TestRepoGuardEntriesPassAdmissionGate(t *testing.T) {
	r := repoRegistry(t)
	bundled := Defaults().Entries
	var added []string
	for id := range r.Entries {
		if _, ok := bundled[id]; !ok {
			added = append(added, id)
		}
	}
	sort.Strings(added)
	for _, id := range repoHookVerbEntries {
		if !contains(added, id) {
			t.Errorf("%s adds no %s entry (it adds %v)", RepoRelPath, id, added)
		}
	}
	for _, id := range added {
		e := r.Entries[id]
		t.Run(id, func(t *testing.T) {
			bad, good := len(e.Fixtures.KnownBad), len(e.Fixtures.KnownGood)
			if bad == 0 || good == 0 {
				t.Fatalf("entry %s ships %d known-bad and %d known-good fixtures; it needs both", id, bad, good)
			}
			if 10*good < 4*(good+bad) {
				t.Fatalf("entry %s: known-good is %d of %d fixtures, below the 40%% floor", id, good, good+bad)
			}
			if e.Tier != TierBlocker || e.Why == "" || e.Successor == "" {
				t.Fatalf("entry %s: tier %q, why %q, successor %q; it must be a blocker that explains itself", id, e.Tier, e.Why, e.Successor)
			}
			for _, cmd := range e.Fixtures.KnownBad {
				d, err := r.Check(cmd)
				if err != nil {
					t.Fatalf("known-bad %q: unexpected error: %v", cmd, err)
				}
				if !contains(d.Matches, id) || d.Verdict != VerdictBlock {
					t.Fatalf("known-bad %q: verdict %q, matches %v; want %s to block", cmd, d.Verdict, d.Matches, id)
				}
			}
			for _, cmd := range e.Fixtures.KnownGood {
				d, err := r.Check(cmd)
				if err != nil {
					t.Fatalf("known-good %q: unexpected error: %v", cmd, err)
				}
				if d.Verdict != VerdictAllow {
					t.Fatalf("known-good %q fired %v (verdict %q)", cmd, d.Matches, d.Verdict)
				}
			}
		})
	}
}

// TestRepoGuardRefusesEveryHookSkip pins the verdicts themselves, apart from
// the fixtures the file ships, so the file cannot pass by weakening its own
// corpus: each verb that takes --no-verify is refused with it, the bundled
// commit and push entries still answer for those two verbs, and each -n that
// means something else, and the flag's name inside text, is allowed.
func TestRepoGuardRefusesEveryHookSkip(t *testing.T) {
	r := repoRegistry(t)
	for _, tc := range []struct {
		cmd   string
		entry string // "" when the command must be allowed
	}{
		{"git merge --no-verify origin/main", "git-merge-no-verify"},
		{"git -C ../other merge --no-verify topic", "git-merge-no-verify"},
		{"git rebase --no-verify origin/main", "git-rebase-no-verify"},
		{"git am --no-verify 0001-fix.patch", "git-am-no-verify"},
		{"git am -n 0001-fix.patch", "git-am-no-verify"},
		{"git pull --no-verify origin main", "git-pull-no-verify"},
		{"git commit --no-verify -m wip", "git-commit-no-verify"},
		{"git commit -n -m wip", "git-commit-no-verify"},
		{"git push --no-verify origin main", "git-push-no-verify"},

		{"git merge -n origin/main", ""},
		{"git merge --no-verify-signatures origin/main", ""},
		{"git rebase -n origin/main", ""},
		{"git pull -n origin main", ""},
		{"git push -n origin main", ""},
		{"git cherry-pick -n abc1234", ""},
		{"git am 0001-fix.patch", ""},
		{`git commit -m "no-verify in text"`, ""},
	} {
		d, err := r.Check(tc.cmd)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", tc.cmd, err)
		}
		switch {
		case tc.entry == "" && d.Verdict != VerdictAllow:
			t.Errorf("%q fired %v (verdict %q); want it allowed", tc.cmd, d.Matches, d.Verdict)
		case tc.entry != "" && (d.Verdict != VerdictBlock || !contains(d.Matches, tc.entry)):
			t.Errorf("%q: verdict %q, matches %v; want %s to block", tc.cmd, d.Verdict, d.Matches, tc.entry)
		}
	}
}
