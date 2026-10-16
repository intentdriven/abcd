package claim

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// The status branches whose names are not read (C11): counted, never named,
// whatever their number or their length.

// pushNames makes each of names a status branch on origin at one commit.
func (w *world) pushNames(names []string) {
	w.t.Helper()
	tip := w.pushStatus("seedname", announcementMsg("merging", "#1", "seedname", t0, 30*time.Minute), "")
	var b strings.Builder
	for _, n := range names {
		fmt.Fprintf(&b, "create %s%s %s\n", StatusRefPrefix, n, tip)
	}
	fmt.Fprintf(&b, "delete %sseedname %s\n", StatusRefPrefix, tip)
	runGitStdin(w.t, w.bare, b.String(), "update-ref", "--stdin")
}

// TestAThousandNamesThatAreNotReadAreCountedInFull: a pusher who makes a
// thousand branches by names the reader does not read gets them counted, all
// of them, in the view and in the block's one line; and once origin no longer
// holds them, a listing that succeeds counts none.
func TestAThousandNamesThatAreNotReadAreCountedInFull(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	var names []string
	for i := 0; i < 1000; i++ {
		names = append(names, fmt.Sprintf("JUNK-%04d-%s", i, strings.Repeat("x", 80)))
	}
	w.pushNames(names)
	pat.status(nil)
	if v := pat.read(); v.UnreadNames != 1000 || v.UnreadNamesUnknown {
		t.Fatalf("unread names = %d (unknown %v); want 1000", v.UnreadNames, v.UnreadNamesUnknown)
	}
	if got := pat.notes(); !strings.Contains(got, "- 1000 status branches whose names are not read; `abcd status` says how to list them\n") {
		t.Fatalf("the block:\n%s\nwant the thousand counted", got)
	}

	var del strings.Builder
	for _, n := range names {
		fmt.Fprintf(&del, "delete %s%s\n", StatusRefPrefix, n)
	}
	runGitStdin(t, w.bare, del.String(), "update-ref", "--stdin")
	pat.status(nil)
	if v := pat.read(); v.UnreadNames != 0 || v.UnreadNamesUnknown {
		t.Fatalf("after origin dropped them, unread names = %d (unknown %v); want none", v.UnreadNames, v.UnreadNamesUnknown)
	}
}

// TestLongNamesThatShareABeginningAreCountedApart: three names that differ
// only past their first hundred characters are three branches, and a name
// this checkout also holds a copy of (a fetch of the whole namespace brought
// it) is still one.
func TestLongNamesThatShareABeginningAreCountedApart(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	long := "Shared-" + strings.Repeat("y", 100)
	w.pushNames([]string{long + "-1", long + "-2", long + "-3"})
	pat.status(nil)
	if v := pat.read(); v.UnreadNames != 3 {
		t.Fatalf("unread names = %d; want 3", v.UnreadNames)
	}
	// A plain fetch of the whole namespace, as `abcd announce` makes, brings
	// a copy of each: the count stays three.
	pat.git("fetch", "-q", "origin", "+"+StatusRefPrefix+"*:"+StatusTrackingPrefix+"*")
	if v := pat.read(); v.UnreadNames != 3 {
		t.Fatalf("with a copy of each held, unread names = %d; want 3", v.UnreadNames)
	}
}

// TestACountThatCannotBeReadIsUnknownNotZero: when what origin's last listing
// counted cannot be read back, the count is unknown, and the view and the
// block say so rather than count none.
func TestACountThatCannotBeReadIsUnknownNotZero(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	w.pushNames([]string{"Octo"})
	pat.status(nil)
	kept := filepath.Join(pat.dir, filepath.FromSlash(statusFile(skippedRel)))
	if err := os.WriteFile(kept, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	v := pat.read()
	if !v.UnreadNamesUnknown {
		t.Fatalf("unread names = %d, unknown %v; want unknown", v.UnreadNames, v.UnreadNamesUnknown)
	}
	if got := pat.notes(); !strings.Contains(got, "- "+UnreadNamesUnknownLine+"; `abcd status` says how to list them\n") {
		t.Fatalf("the block:\n%s\nwant the count said to be unknown", got)
	}
}

// TestADeletedBranchWhoseNameIsNotReadStopsCounting: a status branch whose
// name is not read, of which this checkout holds a copy (a fetch of the whole
// namespace, as `abcd announce` made before it listed the branches, brought
// it), is counted until origin deletes it; then the next fetch of either kind
// drops the copy, and the count with it, in the view and in the block.
func TestADeletedBranchWhoseNameIsNotReadStopsCounting(t *testing.T) {
	for _, next := range []string{"announce", "status"} {
		t.Run(next, func(t *testing.T) {
			w := newWorld(t)
			pat := w.clone("9+pat@users.noreply.github.com", "")
			w.pushNames([]string{"Octo"})
			pat.announceAs("merging", "#1")
			pat.fetchOnly()
			if refs := pat.git("for-each-ref", "--format=%(refname)", StatusTrackingPrefix); !strings.Contains(refs, "Octo") {
				t.Fatalf("the whole namespace's fetch brought no copy of Octo: %s", refs)
			}
			if v := pat.read(); v.UnreadNames != 1 {
				t.Fatalf("with a copy held, unread names = %d; want 1", v.UnreadNames)
			}
			runGit(t, w.bare, "update-ref", "-d", StatusRefPrefix+"Octo")
			if next == "announce" {
				pat.announceAs("merged", "#1")
			} else {
				pat.status(nil)
			}
			if refs := pat.git("for-each-ref", "--format=%(refname)", StatusTrackingPrefix); strings.Contains(refs, "Octo") {
				t.Fatalf("after origin deleted Octo, the next %s kept its copy: %s", next, refs)
			}
			if v := pat.read(); v.UnreadNames != 0 || v.UnreadNamesUnknown {
				t.Fatalf("after origin deleted Octo, unread names = %d (unknown %v); want none", v.UnreadNames, v.UnreadNamesUnknown)
			}
			if got := pat.notes(); strings.Contains(got, "whose name") {
				t.Fatalf("after origin deleted Octo, the block:\n%s\nwant no count of names not read", got)
			}
		})
	}
}

// fetchNamed fetches the named status branches into p, as `abcd status`'s
// fetch does, but keeps no count of the names it did not read: the fetch of
// a FetchStatus whose write of that count failed.
func (p *participant) fetchNamed(names ...string) {
	p.w.t.Helper()
	status := statusBranches
	status.only = names
	if _, err := fetch(p.dir, p.w.clock(), "nothing was fetched", reservations, status); err != nil {
		p.w.t.Fatalf("fetch: %v", err)
	}
}

// TestACountNotKeptAtTheLastFetchIsUnknown: when the count of names not read
// was not kept at the status branches' last fetch, missing or older than it,
// the reader says it is unknown, never zero and never the older count; a
// checkout that never fetched them, and one whose last fetch was of the whole
// namespace, keep a count, or need none, and say no such thing.
func TestACountNotKeptAtTheLastFetchIsUnknown(t *testing.T) {
	w := newWorld(t)
	w.pushStatus("maya", announcementMsg("merging", "#1", "maya", t0, 30*time.Minute), "")
	w.pushNames([]string{"Octo"})

	fresh := w.clone("9+pat@users.noreply.github.com", "")
	if v := fresh.read(); v.UnreadNamesUnknown {
		t.Fatal("a checkout that never fetched the status says the count is unknown")
	}
	fresh.fetchNamed("maya")
	if v := fresh.read(); !v.UnreadNamesUnknown {
		t.Fatalf("with no count kept at the last fetch, unread names = %d, unknown %v; want unknown", v.UnreadNames, v.UnreadNamesUnknown)
	}

	stale := w.clone("7+quinn@users.noreply.github.com", "")
	stale.status(nil)
	if v := stale.read(); v.UnreadNamesUnknown || v.UnreadNames != 1 {
		t.Fatalf("after a full status fetch: unread names = %d, unknown %v; want 1 counted", v.UnreadNames, v.UnreadNamesUnknown)
	}
	w.set(t0.Add(time.Minute))
	stale.fetchNamed("maya")
	if v := stale.read(); !v.UnreadNamesUnknown {
		t.Fatalf("with an older count kept: unread names = %d, unknown %v; want unknown", v.UnreadNames, v.UnreadNamesUnknown)
	}
	stale.status(nil)
	if v := stale.read(); v.UnreadNamesUnknown || v.UnreadNames != 1 {
		t.Fatalf("after the next status fetch: unread names = %d, unknown %v; want 1 counted", v.UnreadNames, v.UnreadNamesUnknown)
	}

	whole := w.clone("8+zed@users.noreply.github.com", "")
	whole.fetchOnly()
	if v := whole.read(); v.UnreadNamesUnknown || v.UnreadNames != 1 {
		t.Fatalf("after a fetch of the whole namespace: unread names = %d, unknown %v; want 1 counted", v.UnreadNames, v.UnreadNamesUnknown)
	}
}

// TestAGitTooOldToReadRefspecsOnStdinIsRefusedByItsVersion: `abcd status`
// names each status branch by its own refspec on `git fetch --stdin`, which
// git reads from 2.29; an older git runs no fetch, and the refusal names the
// version in fixed words, kept for every reader to say.
func TestAGitTooOldToReadRefspecsOnStdinIsRefusedByItsVersion(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	w.pushStatus("maya", announcementMsg("merging", "#1", "maya", t0, 30*time.Minute), "")
	for _, c := range []struct{ line, says string }{
		{"git version 2.28.1 (Apple Git-1)", "this git is 2.28"},
		{"not a version", "this git's version could not be read"},
	} {
		restore := gitutil.SwapGitVersionForTest(func() (string, error) { return c.line, nil })
		log := traced(t)
		_, err := FetchStatus(pat.dir, nil, pat.opts())
		restore()
		want := "`git fetch --stdin`, which the status fetch reads its branches on, needs git 2.29 or later, and " + c.says
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("FetchStatus under %q = %v; want the refusal %q", c.line, err, want)
		}
		if slices.Contains(log.subcommands(), "fetch") {
			t.Fatalf("under %q a fetch ran: %v", c.line, log.subcommands())
		}
		if u := pat.read().Unread; u == nil || !strings.Contains(u.Reason, want) {
			t.Fatalf("under %q the unread status = %+v; want the refusal kept", c.line, u)
		}
	}
}
