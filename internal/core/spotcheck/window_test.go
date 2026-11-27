package spotcheck

import (
	"os/exec"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// commitAt commits once on r at the committer time at and answers the commit.
func commitAt(t *testing.T, r *gittest.Repo, at time.Time) string {
	t.Helper()
	when := at.Format(time.RFC3339)
	cmd := exec.Command("git", "-C", r.Root(), "-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture",
		"-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "a landing")
	cmd.Env = append(gittest.Env(t), "GIT_COMMITTER_DATE="+when, "GIT_AUTHOR_DATE="+when)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	return r.Git("rev-parse", "HEAD")
}

// TestWindowByDays: a window of seven days stays open while every commit of
// its line was committed within them, and once one was committed after, it
// closes at the seven days' end, at the last commit before it; a later commit
// moves neither.
func TestWindowByDays(t *testing.T) {
	r, _ := originWithRemote(t)
	c := Config{WindowDays: 7, PerGroup: 10}
	from := r.Git("rev-parse", "HEAD")
	end := w0.Add(7 * 24 * time.Hour)
	commitAt(t, r, w0.Add(time.Hour))
	in := commitAt(t, r, end)
	if got, err := windowClose(t.Context(), r.Root(), c, from, w0, in); err != nil || got.by != "" {
		t.Fatalf("a window whose every commit is within its days closed: %+v, %v", got, err)
	}
	commitAt(t, r, end.Add(time.Second))
	tip := commitAt(t, r, end.Add(time.Hour))
	got, err := windowClose(t.Context(), r.Root(), c, from, w0, tip)
	if err != nil || got.by != ClosedByDays || got.commit != in || !got.at.Equal(end) {
		t.Fatalf("windowClose = %+v, %v; want days at %s, ending at %s", got, err, end, in[:12])
	}
}

// TestWindowByChanges: a window of ten changes closes at its tenth commit, at
// that commit's time, unless the days closed it first on the line.
func TestWindowByChanges(t *testing.T) {
	r, _ := originWithRemote(t)
	c := Config{WindowDays: 7, WindowChanges: 10, PerGroup: 10}
	from := r.Git("rev-parse", "HEAD")
	var tenth string
	for i := range 11 {
		sha := commitAt(t, r, w0.Add(time.Duration(i+1)*time.Hour))
		if i == 8 {
			if got, err := windowClose(t.Context(), r.Root(), c, from, w0, sha); err != nil || got.by != "" {
				t.Fatalf("nine changes closed a window of ten: %+v, %v", got, err)
			}
		}
		if i == 9 {
			tenth = sha
		}
	}
	got, err := windowClose(t.Context(), r.Root(), c, from, w0, r.Git("rev-parse", "HEAD"))
	if err != nil || got.by != ClosedByChanges || got.commit != tenth || !got.at.Equal(w0.Add(10*time.Hour)) {
		t.Fatalf("windowClose = %+v, %v; want changes at the tenth commit", got, err)
	}
	r2, _ := originWithRemote(t)
	from2 := r2.Git("rev-parse", "HEAD")
	first := commitAt(t, r2, w0.Add(time.Hour))
	commitAt(t, r2, w0.Add(8*24*time.Hour))
	for i := range 10 {
		commitAt(t, r2, w0.Add(9*24*time.Hour+time.Duration(i)*time.Second))
	}
	got, err = windowClose(t.Context(), r2.Root(), c, from2, w0, r2.Git("rev-parse", "HEAD"))
	if err != nil || got.by != ClosedByDays || got.commit != first {
		t.Fatalf("the days came first on the line, yet the close is %+v, %v", got, err)
	}
}
