package worktree

// start.go is where a worktree the store added began (iss-2610231011465325):
// Add records the worktree's branch and the commit that branch started at, one
// small file a worktree, and the merged judgement reads the branch's own
// commits from that start rather than from wherever its reflog begins, since a
// reflog can carry another branch's history.
//
// The records live in the lane, in a directory whose dot-name ValidName never
// gives a worktree, so no worktree can stand there and the lane's own proof
// (the caller's alone, every level a real directory) covers them; list and
// prune skip the directory as the store's own. A later Add at the same name
// replaces the record; prune leaves it, since it deletes nothing git did not
// delete, and a stale record is read only for a worktree on the branch it
// names.

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// startsDir is the lane's directory of start records.
const startsDir = ".starts"

// startCap bounds a start record: a schema version, a ref and an object name.
const startCap = 4 << 10

// startSchema is the start record's schema version.
const startSchema = 1

// startPerm is a start record's mode: the caller's alone.
const startPerm = 0o600

// startRecord is where the branch of one worktree the store added began.
type startRecord struct {
	SchemaVersion int `json:"schema_version"`
	// Branch is the full ref the worktree was added on, refs/heads/<name>.
	Branch string `json:"branch"`
	// Start is the commit the branch stood at when the store added the
	// worktree: a new branch's base, or an existing branch's tip.
	Start string `json:"start"`
}

// writeStart records rec for the worktree name in l, replacing any earlier
// record at that name. The directory is made through the store's level maker,
// and the file written atomically inside it, owner-only.
func writeStart(l Lane, name string, rec startRecord) error {
	if _, err := levels(l.Home, l.Rel+"/"+startsDir, "worktree store", true); err != nil {
		return err
	}
	root, err := fsutil.OpenRealDir(filepath.Join(l.Dir, startsDir))
	if err != nil {
		return err
	}
	defer root.Close()
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomicInRoot(root, name, append(data, '\n'), startPerm)
}

// readStart is the start record for the worktree name in l. A record that is
// absent, is not a regular file in a directory the store would make, or does
// not parse to a branch and a full object name is no record, and the caller
// judges as if the store had not added the worktree.
func readStart(l Lane, name string) (startRecord, bool) {
	if !ValidName(name) {
		return startRecord{}, false
	}
	if ok, err := levels(l.Home, l.Rel+"/"+startsDir, "worktree store", false); !ok || err != nil {
		return startRecord{}, false
	}
	root, err := fsutil.OpenRealDir(filepath.Join(l.Dir, startsDir))
	if err != nil {
		return startRecord{}, false
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, name, startCap)
	if err != nil {
		return startRecord{}, false
	}
	var rec startRecord
	if json.Unmarshal(data, &rec) != nil || rec.SchemaVersion != startSchema ||
		!strings.HasPrefix(rec.Branch, "refs/heads/") || !gitutil.IsFullSHA(rec.Start) {
		return startRecord{}, false
	}
	return rec, true
}

// recordStart records where the worktree name's branch, ref, starts: its tip
// now, just after git checked it out.
func recordStart(repoRoot string, l Lane, name, ref string) error {
	tip, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(tip) {
		return fmt.Errorf("reading the tip of %s: %v", strings.TrimPrefix(ref, "refs/heads/"), err)
	}
	return writeStart(l, name, startRecord{SchemaVersion: startSchema, Branch: ref, Start: tip})
}
