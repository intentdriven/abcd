package claim

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// writeCommit writes one reservation commit with `git commit-tree`: the empty
// tree, the parent when there is one, and the message on standard input. It
// touches no HEAD, index or working tree, so a detached HEAD, a dirty tree or
// a rebase in progress makes no difference. Author and committer are the
// person's git identity, dated by the act's clock. The message is measured
// once more here, so no caller can write one over the cap.
func writeCommit(p *prepared, m message, tree, parent string) (string, error) {
	text := m.text()
	if err := checkMessage(p.root, text); err != nil {
		return "", err
	}
	date := "@" + strconv.FormatInt(p.now.Unix(), 10) + " +0000"
	env := []string{
		"GIT_AUTHOR_NAME=" + p.who.Name, "GIT_AUTHOR_EMAIL=" + p.who.Email, "GIT_AUTHOR_DATE=" + date,
		"GIT_COMMITTER_NAME=" + p.who.Name, "GIT_COMMITTER_EMAIL=" + p.who.Email, "GIT_COMMITTER_DATE=" + date,
	}
	args := []string{"commit-tree", tree}
	if parent != "" {
		args = append(args, "-p", parent)
	}
	args = append(args, "-F", "-")
	out, _, err := personalGit(p.root, text, env, args...)
	if err != nil {
		return "", refuse("git could not write the reservation commit: %v; %s", err, nothingPushed)
	}
	sha := strings.TrimSpace(out)
	if !gitutil.IsFullSHA(sha) {
		return "", refuse("git named the reservation commit %s, which is not an object name; %s", echo(sha), nothingPushed)
	}
	return sha, nil
}

// readTip reads the commit ref names in this checkout's object store as the
// tip of record's reservation branch. ok is false when the ref does not exist.
// The object is read through a bounded read; one past the bound is unreadable,
// never truncated into something that parses.
func readTip(root, record, ref, tree string, fetchedAt time.Time) (t tip, ok bool, err error) {
	out, err := localGit(root, 64<<10, "for-each-ref", "--format=%(refname) %(objectname) %(objecttype)", ref)
	if err != nil {
		return tip{}, false, fmt.Errorf("reading %s: %w", ref, err)
	}
	// for-each-ref matches a pattern by leading path components, so only the
	// line naming the ref exactly counts.
	var name, typ string
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) == 3 && f[0] == ref {
			name, typ = f[1], f[2]
		}
	}
	if name == "" {
		return tip{}, false, nil
	}
	if !gitutil.IsFullSHA(name) {
		return tip{}, false, fmt.Errorf("reading %s: git named its object %s", ref, echo(name))
	}
	if typ != "commit" {
		return dated(root, tip{Commit: name, Unreadable: reasonNoObject}, fetchedAt), true, nil
	}
	obj, err := localGit(root, maxObjectBytes, "cat-file", "commit", name)
	if err != nil {
		if strings.Contains(err.Error(), "exceeded") {
			t := tip{Commit: name, Unreadable: reasonObjectTooBig}
			// The object is too big to parse, but git reads its date.
			if out, derr := localGit(root, 256, "show", "-s", "--format=%ct", name); derr == nil {
				if sec, perr := strconv.ParseInt(strings.TrimSpace(out), 10, 64); perr == nil {
					t.WrittenAt = time.Unix(sec, 0).UTC()
				}
			}
			return dated(root, t, fetchedAt), true, nil
		}
		return tip{}, false, fmt.Errorf("reading %s: %w", ref, err)
	}
	return dated(root, parseTip(record, name, []byte(obj), tree), fetchedAt), true, nil
}

// undatedRelDir keeps, per tip, when this copy first fetched an unreadable
// tip whose commit date cannot be read.
const undatedRelDir = stateRelDir + "/undated"

// dated gives an unreadable tip whose commit date could not be read the time
// this copy first fetched it, so its hold is the default lease from then
// rather than from the year 1. The first time is kept in the local tier, so a
// later fetch does not restart the hold; if it cannot be kept, the fetch at
// hand is the time.
func dated(root string, t tip, fetchedAt time.Time) tip {
	if t.Unreadable == "" || !t.WrittenAt.IsZero() {
		return t
	}
	t.WrittenAt, t.DatedFromFetch = firstFetched(root, t.Commit, fetchedAt), true
	return t
}

// firstFetched is when this copy first fetched commit, recorded now as at when
// nothing records it yet.
func firstFetched(root, commit string, at time.Time) time.Time {
	at = at.UTC().Truncate(time.Second)
	if !gitutil.IsFullSHA(commit) {
		return at
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return at
	}
	defer r.Close()
	for _, d := range []string{stateRelDir, undatedRelDir} {
		if err := r.Mkdir(d, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return at
		}
	}
	rel := undatedRelDir + "/" + commit
	if data, err := fsutil.ReadGuardedInRoot(r, rel, 64); err == nil {
		if seen, perr := time.Parse(time.RFC3339, strings.TrimSpace(string(data))); perr == nil {
			return seen.UTC()
		}
	}
	_ = fsutil.WriteFileAtomicInRoot(r, rel, []byte(stamp(at)+"\n"), stateMode)
	return at
}
