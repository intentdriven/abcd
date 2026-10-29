package interview

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestLatestRefusesARecordACommitCarries: a repository's interview records
// are its local tier's, so one git tracks came with a commit, not from this
// checkout's interview, and is refused rather than read as an answer given.
func TestLatestRefusesARecordACommitCarries(t *testing.T) {
	repo := repoWithLocalTier(t)
	p := Place{Repo: repo}
	if _, err := Write(p, answersRecord("setup", map[string]string{"visibility": "private"}), time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-f", "--", RecordsRel}} {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if got, err := Latest(p, "setup"); err == nil || !strings.Contains(err.Error(), "git tracks") {
		t.Fatalf("Latest over a tracked record = %v, %v; want it refused as tracked", got, err)
	}
}

// TestLatestReadsTheHomeInsideARepository: the home's records are not a
// checkout's local tier, so a repository around the home (a dotfiles one that
// tracks them, say) refuses nothing there.
func TestLatestReadsTheHomeInsideARepository(t *testing.T) {
	home := t.TempDir()
	rec := answersRecord("setup", map[string]string{"statusline.offered": "no"})
	rec.Target = TargetMachine
	if _, err := Write(Place{Home: home}, rec, time.Now()); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-f", "--", "."}} {
		cmd := exec.Command("git", append([]string{"-C", home}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if got, err := Latest(Place{Home: home}, "setup"); err != nil || got["statusline.offered"] != "no" {
		t.Fatalf("Latest from a home a repository tracks = %v, %v; want the home's answer", got, err)
	}
}
