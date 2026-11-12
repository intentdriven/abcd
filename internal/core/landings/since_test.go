package landings

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// dated makes one commit on main committed at date (git's date form).
func dated(t *testing.T, r *gittest.Repo, msg, date string) string {
	t.Helper()
	r.Write("dated/"+date, msg)
	r.Git("add", "-A")
	cmd := exec.Command("git", "-C", r.Root(), "-c", "user.email=d@example.invalid", "-c", "user.name=Dated", "-c", "commit.gpgsign=false",
		"commit", "-q", "-m", msg)
	cmd.Env = append(r.Env(), "GIT_COMMITTER_DATE="+date, "GIT_AUTHOR_DATE="+date)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	return r.Git("rev-parse", "HEAD")
}

// sinceFx is a default branch of four commits a day apart, the second tagged
// v0.1.0 and the third v0.2.0.
func sinceFx(t *testing.T) (*gittest.Repo, []string) {
	r := gittest.NewRepo(t)
	var c []string
	for _, d := range []string{"2026-10-01T12:00:00Z", "2026-10-02T12:00:00Z", "2026-10-03T12:00:00Z", "2026-10-04T12:00:00Z"} {
		c = append(c, dated(t, r, "chore: "+d, d))
	}
	r.Git("tag", "v0.1.0", c[1])
	r.Git("tag", "-a", "-m", "release", "v0.2.0", c[2])
	r.Git("tag", "other", c[0])
	return r, c
}

// TestSinceTag: a tag opens the range after the commit it names.
func TestSinceTag(t *testing.T) {
	r, c := sinceFx(t)
	got, err := Resolve(t.Context(), r.Root(), "v0.1.0")
	if err != nil || got != (Range{From: c[1], To: c[3], Tip: "main", Since: "v0.1.0"}) {
		t.Fatalf("resolve(v0.1.0) = %+v, %v", got, err)
	}
	if got, err := Resolve(t.Context(), r.Root(), "v0.2.0"); err != nil || got.From != c[2] {
		t.Fatalf("an annotated tag resolves to %+v, %v", got, err)
	}
}

// TestSinceDate: a date opens the range at the first first-parent commit at
// or after it.
func TestSinceDate(t *testing.T) {
	r, c := sinceFx(t)
	for since, from := range map[string]string{
		"2026-10-03": c[1],
		"2026-10-02": c[0],
		"2026-09-01": "",
		"2026-10-05": c[3],
	} {
		got, err := Resolve(t.Context(), r.Root(), since)
		if err != nil || got.From != from || got.To != c[3] || got.Since != since {
			t.Fatalf("resolve(%s) = %+v, %v; want from %q", since, got, err, from)
		}
	}
	got, _ := Resolve(t.Context(), r.Root(), "2026-10-03")
	l, err := List(t.Context(), r.Root(), got, Options{Forge: (&fakeForge{t: t}).call})
	if err != nil || len(l.Rows) != 2 || l.Rows[1].Commit != c[2] || l.Since != "2026-10-03" || l.Tip != "main" {
		t.Fatalf("the listing from 2026-10-03: %+v, %v", l, err)
	}
	if _, err := Resolve(t.Context(), r.Root(), "2026-13-40"); err == nil {
		t.Fatal("a date that is no day resolved")
	}
}

// TestDefaultRangeIsTheLastReleaseTag: with no since, the range opens after
// the last release tag reachable from the tip.
func TestDefaultRangeIsTheLastReleaseTag(t *testing.T) {
	r, c := sinceFx(t)
	got, err := Resolve(t.Context(), r.Root(), "")
	if err != nil || got != (Range{From: c[2], To: c[3], Tip: "main", Since: "v0.2.0"}) {
		t.Fatalf("resolve() = %+v, %v", got, err)
	}
}

// TestNoTagAndNoSinceRefuses: with no release tag and no since, there is no
// range, and the refusal names the flag.
func TestNoTagAndNoSinceRefuses(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("chore: one")
	r.Git("tag", "other")
	_, err := Resolve(t.Context(), r.Root(), "")
	if !errors.Is(err, ErrNoReleaseTag) || !strings.Contains(err.Error(), "--since") {
		t.Fatalf("resolve() = %v", err)
	}
}

// TestNoDefaultBranchRefuses: a repository with no commit has no range.
func TestNoDefaultBranchRefuses(t *testing.T) {
	r := gittest.NewRepo(t)
	if _, err := Resolve(t.Context(), r.Root(), ""); !errors.Is(err, ErrNoDefaultBranch) {
		t.Fatalf("resolve() = %v", err)
	}
}

// TestSinceRefusesWhatIsNeither: a since that is neither a tag the repository
// holds nor a date is refused, masked, and never read as a flag.
func TestSinceRefusesWhatIsNeither(t *testing.T) {
	r, _ := sinceFx(t)
	for _, since := range []string{"v9.9.9", "--all", "main~1", "x\x1b[2J", "refs/heads/main"} {
		_, err := Resolve(t.Context(), r.Root(), since)
		if err == nil || strings.ContainsRune(err.Error(), 0x1b) {
			t.Fatalf("resolve(%q) = %v", since, err)
		}
	}
}

// TestABackDatedCommitDoesNotCutADateRangeShort: the range opens at the
// oldest first-parent commit dated on or after the day, so a commit dated
// before it nearer the tip is inside the range, and the range's note says
// the line's dates are not in order.
func TestABackDatedCommitDoesNotCutADateRangeShort(t *testing.T) {
	r, c := sinceFx(t)
	back := dated(t, r, "chore: back-dated", "2026-09-15T12:00:00Z")
	after := dated(t, r, "chore: after", "2026-10-06T12:00:00Z")
	got, err := Resolve(t.Context(), r.Root(), "2026-10-03")
	if err != nil || got.From != c[1] || got.To != after || !strings.Contains(got.Note, "1 of the range's first-parent commits are dated before 2026-10-03") {
		t.Fatalf("resolve(2026-10-03) = %+v, %v", got, err)
	}
	l, err := List(t.Context(), r.Root(), got, Options{Forge: (&fakeForge{t: t}).call})
	if err != nil || len(l.Rows) != 4 || l.Rows[1].Commit != back || l.SinceNote == "" {
		t.Fatalf("the listing from 2026-10-03: %+v, %v", l, err)
	}
	if got, _ := Resolve(t.Context(), r.Root(), "2026-10-02"); got.From != c[0] || got.Note == "" {
		t.Fatalf("resolve(2026-10-02) = %+v", got)
	}
	inOrder, _ := sinceFx(t)
	if got, _ := Resolve(t.Context(), inOrder.Root(), "2026-10-03"); got.Note != "" {
		t.Fatalf("a line in order has a note: %q", got.Note)
	}
}

// TestAnEndedCtxIsReportedAsThat: a ctx that has ended is resolve's error,
// not a branch or a tag the repository lacks.
func TestAnEndedCtxIsReportedAsThat(t *testing.T) {
	r, _ := sinceFx(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	for _, since := range []string{"", "v0.1.0", "2026-10-03"} {
		if _, err := Resolve(ctx, r.Root(), since); !errors.Is(err, context.Canceled) {
			t.Fatalf("resolve(%q) with an ended ctx = %v", since, err)
		}
	}
}

// TestATagOffTheFirstParentLineIsSaid: a tag on a branch merged into the
// default branch opens a range of what the tip reaches that the tag does not,
// and the range's note says so.
func TestATagOffTheFirstParentLineIsSaid(t *testing.T) {
	r, c := sinceFx(t)
	r.Git("checkout", "-q", "-b", "release", c[1])
	tagged := dated(t, r, "chore: release fix", "2026-10-05T12:00:00Z")
	r.Git("tag", "v0.1.1")
	r.Git("checkout", "-q", "main")
	r.Git("merge", "-q", "--no-ff", "-m", "Merge pull request #5 from org/release", "release")
	got, err := Resolve(t.Context(), r.Root(), "v0.1.1")
	if err != nil || got.From != tagged || !strings.Contains(got.Note, "not on main's first-parent line") {
		t.Fatalf("resolve(v0.1.1) = %+v, %v", got, err)
	}
	l, err := List(t.Context(), r.Root(), got, Options{Forge: (&fakeForge{t: t}).call})
	if err != nil || len(l.Rows) != 3 || !strings.Contains(l.SinceNote, "not on main's first-parent line") {
		t.Fatalf("the listing from v0.1.1: %+v, %v", l, err)
	}
	if got, err := Resolve(t.Context(), r.Root(), "v0.1.0"); err != nil || got.Note != "" {
		t.Fatalf("a tag on the line has a note: %+v, %v", got, err)
	}
}

// failingGit puts a git on PATH for the rest of t that fails, as a read git
// could not make, every call whose arguments hold one of words as a word, and
// runs the real git for every other.
func failingGit(t *testing.T, words ...string) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	script := "#!/bin/sh\nfor a in \"$@\"; do\n  case \"$a\" in\n"
	for _, w := range words {
		script += "    " + w + ") echo 'fatal: a read that failed' >&2; exit 128;;\n"
	}
	script += "  esac\ndone\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestAFailedReadOfTheTagsIsNotNoReleaseTag: describe fails both for no tag
// and for a read that failed, so only a listing of the release tags that
// comes back empty is "no release tag"; a failed describe beside a tag the
// tip reaches, or a failed listing, is a failed read of the tags.
func TestAFailedReadOfTheTagsIsNotNoReleaseTag(t *testing.T) {
	r, _ := sinceFx(t)
	empty := gittest.NewRepo(t)
	empty.Commit("chore: one")
	for _, tc := range []struct {
		name  string
		root  string
		words []string
		noTag bool
	}{
		{"describe fails beside a tag", r.Root(), []string{"describe"}, false},
		{"describe and the listing fail", r.Root(), []string{"describe", "--merged"}, false},
		{"the listing fails with no tag", empty.Root(), []string{"--merged"}, false},
		{"no tag, both read", empty.Root(), nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			failingGit(t, tc.words...)
			_, err := Resolve(t.Context(), tc.root, "")
			if err == nil || errors.Is(err, ErrNoReleaseTag) != tc.noTag {
				t.Fatalf("resolve() = %v; want no release tag %v", err, tc.noTag)
			}
		})
	}
}

// TestARenamedReleaseTagNamesTheFlag: git describes an annotated tag by the
// name stored inside it, so a release tag whose ref was renamed is described
// under a name no ref holds. That is a range the person must name, as with no
// release tag, and the refusal says so; a failed read of the tag list stays a
// failed read (TestAFailedReadOfTheTagsIsNotNoReleaseTag).
func TestARenamedReleaseTagNamesTheFlag(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("chore: one")
	r.Git("tag", "-a", "v1.0.0", "-m", "release")
	r.Git("update-ref", "refs/tags/v2.0.0", "refs/tags/v1.0.0")
	r.Git("update-ref", "-d", "refs/tags/v1.0.0")
	r.Commit("chore: two")
	_, err := Resolve(t.Context(), r.Root(), "")
	if !errors.Is(err, ErrReleaseTagUnnamed) || !strings.Contains(err.Error(), "--since") || !strings.Contains(err.Error(), "v1.0.0") {
		t.Fatalf("resolve() over a renamed release tag = %v", err)
	}
	// The name describe printed, and any like it, is no tag: rev-parse would
	// read its -g and hex as the commit they name.
	for _, since := range []string{"v1.0.0-1-g" + r.Git("rev-parse", "HEAD"), "v9-1-g" + r.Git("rev-parse", "HEAD")[:12]} {
		if _, err := Resolve(t.Context(), r.Root(), since); !errors.Is(err, ErrBadSince) {
			t.Fatalf("resolve(%q) = %v, want a refused since", since, err)
		}
	}
	if got, err := Resolve(t.Context(), r.Root(), "v2.0.0"); err != nil || got.From != r.Git("rev-parse", "HEAD~1") {
		t.Fatalf("resolve(v2.0.0) = %+v, %v", got, err)
	}
}
