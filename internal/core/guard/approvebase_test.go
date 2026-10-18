package guard

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The approver judges against the registry the run started on: the lane's
// base, or the worktree's HEAD when an interview launched. A change to
// .abcd/guard.json since then is accepted only where it strengthens that
// registry; any weakening, of a warn as much as a blocker, committed in the
// run or not, leaves no registry in force, and every command is refused.

const (
	repoWarn        = `{"schema_version":1,"entries":{"no-make-clean":{"tier":"warn","pattern":{"command":"make","subcommand":"clean"},"why":"w","successor":"s"}}}`
	repoWarnMoved   = `{"schema_version":1,"entries":{"no-make-clean":{"tier":"warn","pattern":{"command":"make","subcommand":"cleanest"},"why":"w","successor":"s"}}}`
	repoWarnReword  = `{"schema_version":1,"entries":{"no-make-clean":{"tier":"warn","pattern":{"command":"make","subcommand":"clean"},"why":"reworded","successor":"s"}}}`
	repoWarnRaised  = `{"schema_version":1,"entries":{"no-make-clean":{"tier":"blocker","pattern":{"command":"make","subcommand":"clean"},"why":"w","successor":"s"}}}`
	repoEmpty       = `{"schema_version":1}`
	repoWarnAndMore = `{"schema_version":1,"entries":{"no-make-clean":{"tier":"warn","pattern":{"command":"make","subcommand":"clean"},"why":"w","successor":"s"},"no-make-dist":{"tier":"blocker","pattern":{"command":"make","subcommand":"dist"},"why":"w","successor":"s"}}}`
)

// baseRepo commits the registry at the run's base and returns the repository
// and the base's commit.
func baseRepo(t *testing.T, atBase string) (*gittest.Repo, string) {
	t.Helper()
	repo := gittest.NewRepo(t)
	repo.Write("README", "x\n")
	if atBase != "" {
		repo.Write(RepoRelPath, atBase)
	}
	repo.Commit("base")
	return repo, strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
}

func TestApproveRefusesARegistryWeakenedSinceTheBase(t *testing.T) {
	for _, tc := range []struct {
		name      string
		atBase    string
		committed string // committed after the base; "" = nothing
		working   string // left in the working tree; "" = nothing
	}{
		{"a warn retiered to nothing is a warn removed, committed in the run", repoWarn, repoEmpty, ""},
		{"a warn removed, uncommitted", repoWarn, "", repoEmpty},
		{"a warn's pattern changed, uncommitted", repoWarn, "", repoWarnMoved},
		{"a warn's pattern changed, committed in the run", repoWarn, repoWarnMoved, ""},
		{"a blocker retiered, committed in the run", repoBlocker, repoRetiered, ""},
		{"a bundled blocker retiered, committed in the run", "", retierBlocker, ""},
		{"the kill switch, committed in the run", "", killSwitch, ""},
		{"a bundled warn retiered back, after the base raised it", `{"schema_version":1,"entries":{"git-clean":{"tier":"blocker"}}}`, repoEmpty, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, base := baseRepo(t, tc.atBase)
			if tc.committed != "" {
				repo.Write(RepoRelPath, tc.committed)
				repo.Commit("in the run")
			}
			if tc.working != "" {
				repo.Write(RepoRelPath, tc.working)
			}
			ld := LoadRepoAt(repo.Root(), base)
			if ld.Posture != LoadRepoDropped || !errors.Is(ld.Err, ErrWeakenedSinceBase) {
				t.Fatalf("LoadRepoAt = posture %v, err %v; want the registry not in force", ld.Posture, ld.Err)
			}
			a := approveShell(ld, NewFiles(repo.Root()), "go test ./...")
			if a.Allow || a.EntryID != registryEntryID {
				t.Errorf("a command was answered %+v under a weakened registry; want %q", a, registryEntryID)
			}
		})
	}
}

func TestApproveTakesStrengtheningChangesSinceTheBase(t *testing.T) {
	for _, tc := range []struct {
		name, atBase, committed, working, blocked string
	}{
		{"a new blocker, uncommitted", "", "", repoBlocker, "make clean"},
		{"a new blocker, committed in the run", repoWarn, repoWarnAndMore, "", "make dist"},
		{"a warn raised to a blocker", repoWarn, "", repoWarnRaised, "make clean"},
		{"a warn reworded", repoWarn, repoWarnReword, "", ""},
		{"nothing changed", repoWarn, "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, base := baseRepo(t, tc.atBase)
			if tc.committed != "" {
				repo.Write(RepoRelPath, tc.committed)
				repo.Commit("in the run")
			}
			if tc.working != "" {
				repo.Write(RepoRelPath, tc.working)
			}
			ld := LoadRepoAt(repo.Root(), base)
			if ld.Posture != LoadClean {
				t.Fatalf("LoadRepoAt = posture %v, err %v; a strengthening change is in force", ld.Posture, ld.Err)
			}
			if a := approveShell(ld, NewFiles(repo.Root()), "go test ./..."); !a.Allow {
				t.Errorf("a clean command was refused: %+v", a)
			}
			if tc.blocked != "" {
				if a := approveShell(ld, NewFiles(repo.Root()), tc.blocked); a.Allow {
					t.Errorf("%q was approved; the strengthened registry is the one in force", tc.blocked)
				}
			}
		})
	}
}

func TestApproveRefusesABaseItCannotRead(t *testing.T) {
	repo, _ := baseRepo(t, "")
	for _, base := range []string{"", "--all", strings.Repeat("0", 40), "not-a-commit"} {
		ld := LoadRepoAt(repo.Root(), base)
		if ld.Posture != LoadRepoDropped || ld.Err == nil {
			t.Errorf("base %q: posture %v, err %v; want the registry not in force", base, ld.Posture, ld.Err)
		}
	}
}
