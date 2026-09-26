package guard

import (
	"errors"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// iss-147. The only way to weaken the guard is a committed, reviewable edit to
// .abcd/guard.json (spc-16), and nothing enforced the committed half: Load read
// the working tree, so an agent's uncommitted write of `"disabled": true` — a
// write the guard itself allows — switched the guard off on the very next
// command. A weakening edit now takes effect only once HEAD carries it; until
// then the committed registry stays in force and Load names the refused edit.

const (
	killSwitch    = `{"schema_version":1,"disabled":true}`
	retierBlocker = `{"schema_version":1,"entries":{"git-push-force":{"tier":"warn"}}}`
	repoBlocker   = `{"schema_version":1,"entries":{"no-make-clean":{"tier":"blocker","pattern":{"command":"make","subcommand":"clean"},"why":"w","successor":"s"}}}`
	repoRetiered  = `{"schema_version":1,"entries":{"no-make-clean":{"tier":"warn","pattern":{"command":"make","subcommand":"clean"},"why":"w","successor":"s"}}}`
)

func TestUncommittedWeakeningIsRefused(t *testing.T) {
	cases := []struct {
		name      string
		committed string // "" = no committed guard.json
		working   string
		check     string // a command the committed registry blocks
	}{
		{"an uncommitted kill switch", "", killSwitch, "cd scratch && rm -rf *"},
		{"an uncommitted retier of a bundled blocker", "", retierBlocker, "git push --force origin main"},
		{"an uncommitted retier of a committed repo blocker", repoBlocker, repoRetiered, "make clean"},
		{"an uncommitted kill switch over a committed file", repoBlocker, killSwitch, "make clean"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := gittest.NewRepo(t)
			repo.Write("README", "x\n")
			if tc.committed != "" {
				repo.Write(RepoRelPath, tc.committed)
			}
			repo.Commit("seed")
			repo.Write(RepoRelPath, tc.working)

			r, err := Load(repo.Root())
			if !errors.Is(err, ErrUncommittedOverride) {
				t.Fatalf("Load error = %v, want %v", err, ErrUncommittedOverride)
			}
			if r.Disabled {
				t.Fatal("an uncommitted kill switch disabled the guard")
			}
			d, cerr := r.Check(tc.check)
			if cerr != nil {
				t.Fatal(cerr)
			}
			if d.Verdict != VerdictBlock {
				t.Errorf("Check(%q) = %q via %q: the committed registry must stay in force", tc.check, d.Verdict, d.EntryID)
			}
			ld := LoadRepo(repo.Root())
			if ld.Posture != LoadRepoDropped || !errors.Is(ld.Err, ErrUncommittedOverride) {
				t.Errorf("LoadRepo = posture %v, err %v; want the dropped-repo-layer posture naming the refused edit", ld.Posture, ld.Err)
			}
		})
	}
}

func TestCommittedAndStrengtheningOverridesLoad(t *testing.T) {
	t.Run("a committed kill switch is honoured", func(t *testing.T) {
		repo := gittest.NewRepo(t)
		repo.Write(RepoRelPath, killSwitch)
		repo.Commit("disable the guard")
		r, err := Load(repo.Root())
		if err != nil || !r.Disabled {
			t.Fatalf("Load = disabled %v, err %v; a committed kill switch is the reviewed escape", r.Disabled, err)
		}
	})
	t.Run("a committed retier is honoured", func(t *testing.T) {
		repo := gittest.NewRepo(t)
		repo.Write(RepoRelPath, retierBlocker)
		repo.Commit("retier")
		r, err := Load(repo.Root())
		if err != nil || r.Entries["git-push-force"].Tier != TierWarn {
			t.Fatalf("Load = tier %q, err %v", r.Entries["git-push-force"].Tier, err)
		}
	})
	t.Run("an uncommitted new blocker strengthens and loads", func(t *testing.T) {
		repo := gittest.NewRepo(t)
		repo.Write("README", "x\n")
		repo.Commit("seed")
		repo.Write(RepoRelPath, repoBlocker)
		r, err := Load(repo.Root())
		if err != nil {
			t.Fatalf("a strengthening edit must load uncommitted, got %v", err)
		}
		if d, _ := r.Check("make clean"); d.Verdict != VerdictBlock {
			t.Errorf("the new blocker did not take effect: %+v", d)
		}
	})
	t.Run("an uncommitted retier of a warn to a blocker loads", func(t *testing.T) {
		repo := gittest.NewRepo(t)
		repo.Write("README", "x\n")
		repo.Commit("seed")
		repo.Write(RepoRelPath, `{"schema_version":1,"entries":{"git-clean":{"tier":"blocker"}}}`)
		if _, err := Load(repo.Root()); err != nil {
			t.Fatalf("a strengthening retier must load uncommitted, got %v", err)
		}
	})
}

// TestLoadRepoCarriesTheFailSafePolicy — iss-2608291814576261. The fail-safe
// policy for a broken repo layer was decided in the CLI by counting entries
// after a Load error, so every surface re-derived its own answer. LoadRepo
// decides it once, in core, and names the posture a front door formats.
func TestLoadRepoCarriesTheFailSafePolicy(t *testing.T) {
	t.Run("clean", func(t *testing.T) {
		ld := LoadRepo(t.TempDir())
		if ld.Posture != LoadClean || ld.Err != nil || len(ld.Registry.Entries) == 0 {
			t.Fatalf("LoadRepo(no override) = %+v", ld)
		}
	})
	t.Run("a malformed repo file drops the repo layer and keeps the bundled hazards", func(t *testing.T) {
		ld := LoadRepo(writeOverride(t, `{not json`))
		if ld.Posture != LoadRepoDropped || !errors.Is(ld.Err, ErrMalformedConfig) {
			t.Fatalf("posture %v err %v", ld.Posture, ld.Err)
		}
		if len(ld.Registry.Entries) != len(Defaults().Entries) {
			t.Errorf("the bundled hazards must stay armed, got %d entries", len(ld.Registry.Entries))
		}
	})
	t.Run("no registry at all is unavailable", func(t *testing.T) {
		if got := postureOf(Registry{}, errors.New("x")); got != LoadUnavailable {
			t.Errorf("an empty registry with an error = %v, want LoadUnavailable", got)
		}
		if got := postureOf(Registry{}, nil); got != LoadUnavailable {
			t.Errorf("an empty registry = %v, want LoadUnavailable", got)
		}
	})
}
