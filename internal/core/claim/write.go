package claim

import (
	"fmt"
	"strconv"
	"strings"

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
func readTip(root, record, ref, tree string) (t tip, ok bool, err error) {
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
		return tip{Commit: name, Unreadable: reasonNoObject}, true, nil
	}
	obj, err := localGit(root, maxObjectBytes, "cat-file", "commit", name)
	if err != nil {
		if strings.Contains(err.Error(), "exceeded") {
			return tip{Commit: name, Unreadable: reasonObjectTooBig}, true, nil
		}
		return tip{}, false, fmt.Errorf("reading %s: %w", ref, err)
	}
	return parseTip(record, name, []byte(obj), tree), true, nil
}
