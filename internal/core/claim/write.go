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

// writeCommit writes one reservation act's commit: writeEmptyCommit, as the
// person the act prepared, dated by its clock.
func writeCommit(p *prepared, m commitMessage, tree, parent string) (string, error) {
	return writeEmptyCommit(p.root, p.who, p.now, m, tree, parent)
}

// writeEmptyCommit writes one commit of m's family with `git commit-tree`: the
// empty tree, the parent when there is one, and the message on standard input.
// It touches no HEAD, index or working tree, so a detached HEAD, a dirty tree
// or a rebase in progress makes no difference. Author and committer are who,
// dated at now. The message is measured and judged once more here
// (checkText), so no caller can write one over the cap or past the outbound
// policy.
func writeEmptyCommit(root string, who identity, now time.Time, m commitMessage, tree, parent string) (string, error) {
	f := m.messageFamily()
	text := m.text()
	if err := checkText(root, f, text); err != nil {
		return "", err
	}
	date := "@" + strconv.FormatInt(now.Unix(), 10) + " +0000"
	env := []string{
		"GIT_AUTHOR_NAME=" + who.Name, "GIT_AUTHOR_EMAIL=" + who.Email, "GIT_AUTHOR_DATE=" + date,
		"GIT_COMMITTER_NAME=" + who.Name, "GIT_COMMITTER_EMAIL=" + who.Email, "GIT_COMMITTER_DATE=" + date,
	}
	args := []string{"commit-tree", tree}
	if parent != "" {
		args = append(args, "-p", parent)
	}
	args = append(args, "-F", "-")
	out, _, err := personalGit(root, text, env, args...)
	if err != nil {
		return "", refuse("git could not write the %s commit: %v; %s", f.name, err, nothingPushed)
	}
	sha := strings.TrimSpace(out)
	if !gitutil.IsFullSHA(sha) {
		return "", refuse("git named the %s commit %s, which is not an object name; %s", f.name, echo(sha), nothingPushed)
	}
	return sha, nil
}

// readTip reads the commit ref names in this checkout's object store as the
// tip of record's reservation branch. ok is false when the ref does not exist.
func readTip(root, record, ref, tree string, fetchedAt time.Time, keep bool) (t tip, ok bool, err error) {
	name, typ, ok, err := resolveRef(root, ref)
	if err != nil || !ok {
		return tip{}, false, err
	}
	if typ != "commit" {
		return tip{head: dated(root, head{Commit: name, Unreadable: reasonNoObject}, fetchedAt, keep)}, true, nil
	}
	h, m, err := readCommit(root, name, tree, fetchedAt, keep, reservationParser(record))
	if err != nil {
		return tip{}, false, fmt.Errorf("reading %s: %w", ref, err)
	}
	return tip{head: h, message: m}, true, nil
}

// resolveRef is the object ref names in this checkout and its type; ok is
// false when the ref does not exist.
func resolveRef(root, ref string) (name, typ string, ok bool, err error) {
	out, err := localGit(root, 64<<10, "for-each-ref", "--format=%(refname) %(objectname) %(objecttype)", ref)
	if err != nil {
		return "", "", false, fmt.Errorf("reading %s: %w", ref, err)
	}
	// for-each-ref matches a pattern by leading path components, so only the
	// line naming the ref exactly counts.
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) == 3 && f[0] == ref {
			name, typ = f[1], f[2]
		}
	}
	if name == "" {
		return "", "", false, nil
	}
	if !gitutil.IsFullSHA(name) {
		return "", "", false, fmt.Errorf("reading %s: git named its object %s", ref, echo(name))
	}
	return name, typ, true, nil
}

// readCommit reads the commit object name in this checkout's object store as
// a commit of the family parse reads, through parseTip. The object is read
// through a bounded read; one past the bound is unreadable, never truncated
// into something that parses. An unreadable commit whose date cannot be read
// is dated from when this copy first fetched it (dated). The error is git's,
// for its caller to say what it was reading.
func readCommit[M any](root, name, tree string, fetchedAt time.Time, keep bool, parse func(string) (M, string)) (head, M, error) {
	var zero M
	obj, err := localGit(root, maxObjectBytes, "cat-file", "commit", name)
	if err != nil {
		if strings.Contains(err.Error(), "exceeded") {
			h := head{Commit: name, Unreadable: reasonObjectTooBig}
			// The object is too big to parse, but git reads its date.
			if out, derr := localGit(root, 256, "show", "-s", "--format=%ct", name); derr == nil {
				if sec, perr := strconv.ParseInt(strings.TrimSpace(out), 10, 64); perr == nil {
					h.WrittenAt = time.Unix(sec, 0).UTC()
				}
			}
			return dated(root, h, fetchedAt, keep), zero, nil
		}
		return head{}, zero, err
	}
	h, m := parseTip(name, []byte(obj), tree, parse)
	return dated(root, h, fetchedAt, keep), m, nil
}

// undatedRelDir keeps, per tip, when this copy first fetched an unreadable
// tip whose commit date cannot be read.
const undatedRelDir = stateRelDir + "/undated"

// dated gives an unreadable commit whose date could not be read the time
// this copy first fetched it, so its hold is the default lease from then
// rather than from the year 1. The first time is kept in the local tier, so a
// later fetch does not restart the hold; if it cannot be kept, the fetch at
// hand is the time. keep is false for a reader that writes nothing (the view),
// which reads a kept time and records none.
func dated(root string, h head, fetchedAt time.Time, keep bool) head {
	if h.Unreadable == "" || !h.WrittenAt.IsZero() {
		return h
	}
	h.WrittenAt, h.DatedFromFetch = firstFetched(root, h.Commit, fetchedAt, keep), true
	return h
}

// firstFetched is when this copy first fetched commit, recorded as at when
// nothing records it yet and keep allows it.
func firstFetched(root, commit string, at time.Time, keep bool) time.Time {
	at = at.UTC().Truncate(time.Second)
	if !gitutil.IsFullSHA(commit) {
		return at
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return at
	}
	defer r.Close()
	rel := undatedRelDir + "/" + commit
	if data, err := fsutil.ReadGuardedInRoot(r, rel, 64); err == nil {
		if seen, perr := time.Parse(time.RFC3339, strings.TrimSpace(string(data))); perr == nil {
			return seen.UTC()
		}
	}
	if !keep {
		return at
	}
	for _, d := range []string{stateRelDir, undatedRelDir} {
		if err := r.Mkdir(d, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return at
		}
	}
	_ = fsutil.WriteFileAtomicInRoot(r, rel, []byte(stamp(at)+"\n"), stateMode)
	return at
}
