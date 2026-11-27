package loop

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/gittest"
)

// A drive with no --until runs a set number of days (itd-2611251809580662
// decisions 1 and 9, spc-2611251810508418 step 3): seven, or drive.until_days
// from the repository's file at origin's default branch, 1 to 31, never read
// from a working tree and never from the machine's file.

// TestADriveRunsSevenDaysWithNothingSet: with no file setting it, the drive
// runs abcd's seven days, from the bundled layer, and says nothing more.
func TestADriveRunsSevenDaysWithNothingSet(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	v, note, err := resolveDriveDays(repo.Root(), layered.Roots{Repo: repo.Root(), Home: t.TempDir()})
	if err != nil || v != bundledDriveDays() || v.Value != 7 || note != "" {
		t.Fatalf("with nothing set a drive runs seven days: %+v %q %v", v, note, err)
	}
}

// TestTheRepositorysUntilDaysAtOriginsDefaultBranchSetTheDefault: the
// repository's drive.until_days at origin's default branch sets the days,
// named with its layer and the file at the branch; the figure on another of
// origin's branches is not read.
func TestTheRepositorysUntilDaysAtOriginsDefaultBranchSetTheDefault(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	roots := layered.Roots{Repo: repo.Root(), Home: t.TempDir()}
	ceilingOnBranch(t, repo, "develop", `{"drive": {"until_days": 2}}`)
	if v, _, err := resolveDriveDays(repo.Root(), roots); err != nil || v != bundledDriveDays() {
		t.Fatalf("another branch on origin sets nothing: %+v %v", v, err)
	}
	ceilingOnOrigin(t, repo, `{"drive": {"until_days": 14}}`)
	v, note, err := resolveDriveDays(repo.Root(), roots)
	if err != nil || note != "" || v != (PaceValue{Value: 14, Layer: "repo", Origin: ".abcd/config.json at origin/main"}) {
		t.Fatalf("the repository's figure at origin's default branch sets the days: %+v %q %v", v, note, err)
	}
	ceilingOnOrigin(t, repo, `{"drive": {"until_days": 31}, "drain": {"in_work_max": 2}}`)
	if v, _, err := resolveDriveDays(repo.Root(), roots); err != nil || v.Value != 31 {
		t.Fatalf("31, the most, is taken beside another namespace: %+v %v", v, err)
	}
}

// TestAWorkingTreeEditOfUntilDaysChangesNothing: a lane's agent can write the
// working tree, and a drive.until_days there, committed on the checkout's own
// branch or not, changes nothing; nor does a value the resolver would refuse.
func TestAWorkingTreeEditOfUntilDaysChangesNothing(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	roots := layered.Roots{Repo: repo.Root(), Home: t.TempDir()}
	ceilingOnOrigin(t, repo, `{"drive": {"until_days": 3}}`)
	repo.Write(".abcd/config.json", `{"drive": {"until_days": 30}}`)
	repo.Commit("a lane's own figure")
	repo.Write(".abcd/config.json", `{"drive": {"until_days": 99}}`)
	v, _, err := resolveDriveDays(repo.Root(), roots)
	if err != nil || v.Value != 3 || v.Layer != "repo" {
		t.Fatalf("the working tree and the checkout's own branch change nothing: %+v %v", v, err)
	}
}

// TestUntilDaysOfZeroAndThirtyTwoRefuseNamingTheirOrigin: 0, 32, a string and
// a misspelt key in the repository's file at origin's default branch refuse,
// naming the file at the branch, and never fall back to seven.
func TestUntilDaysOfZeroAndThirtyTwoRefuseNamingTheirOrigin(t *testing.T) {
	for _, c := range []struct{ body, want string }{
		{`{"drive": {"until_days": 0}}`, "from 1 to 31"},
		{`{"drive": {"until_days": 32}}`, "from 1 to 31"},
		{`{"drive": {"until_days": "7"}}`, "drive.until_days is \"7\""},
		{`{"drive": {"until_day": 7}}`, "drive.until_day"},
	} {
		t.Run(c.body, func(t *testing.T) {
			repo, _ := reservingDrainRepo(t, "commands/drain.md")
			ceilingOnOrigin(t, repo, c.body)
			v, _, err := resolveDriveDays(repo.Root(), layered.Roots{Repo: repo.Root(), Home: t.TempDir()})
			rf := mustRefusal(t, err)
			if !strings.Contains(rf.Reason, ".abcd/config.json at origin/main") || !strings.Contains(rf.Reason, c.want) {
				t.Fatalf("the refusal names the file at origin's default branch and %q: %q", c.want, rf.Reason)
			}
			if !strings.Contains(rf.Remedy, "drive.until_days") || v != (PaceValue{}) {
				t.Fatalf("the remedy names the key and no figure is returned: %q %+v", rf.Remedy, v)
			}
		})
	}
}

// TestUntilDaysInTheMachinesFileIsRefused: the machine's file does not hold
// the default, and one there is refused naming the file, before origin is
// asked, never passed over.
func TestUntilDaysInTheMachinesFileIsRefused(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	home := t.TempDir()
	writeConfigAt(t, abcdhome.Path(home, "config.json"), `{"drive": {"until_days": 3}}`)
	_, _, err := resolveDriveDays(repo.Root(), layered.Roots{Repo: repo.Root(), Home: home})
	rf := mustRefusal(t, err)
	if !strings.Contains(rf.Reason, "config.json sets drive.until_days") || !strings.Contains(rf.Reason, "does not hold") {
		t.Fatalf("the machine's drive.until_days is refused naming the file: %q", rf.Reason)
	}
	writeConfigAt(t, abcdhome.Path(home, "config.json"), `{"drain": {"bundle_max": 2}}`)
	if v, _, err := resolveDriveDays(repo.Root(), layered.Roots{Repo: repo.Root(), Home: home}); err != nil || v.Value != 7 {
		t.Fatalf("a machine file with no drive key is no refusal: %+v %v", v, err)
	}
}

// TestACheckoutWithNoOriginReadsNoRepositoryDaysAndSaysSo: with no origin,
// the checkout's own committed and working-tree figures are not the
// repository's; the drive runs seven days and the note says why.
func TestACheckoutWithNoOriginReadsNoRepositoryDaysAndSaysSo(t *testing.T) {
	alone := gittest.NewRepo(t)
	alone.Write(".abcd/config.json", `{"drive": {"until_days": 2}}`)
	alone.Commit("config")
	v, note, err := resolveDriveDays(alone.Root(), layered.Roots{Repo: alone.Root(), Home: t.TempDir()})
	if err != nil || v != bundledDriveDays() || !strings.Contains(note, "no remote named origin") || !strings.Contains(note, "7 days") {
		t.Fatalf("no origin reads no repository figure, said: %+v %q %v", v, note, err)
	}
}
