package guard

// checklistpath.go is the file tools' refusal on a working copy's checklist
// (spc-2610230805429429 step 5). The checklist and its last good copy are the
// `abcd checklist` verbs' to write: each write validates the whole list and
// scans every field for secrets first. A Write or Edit there skips both, so the
// guard hook denies the host's own file tools the two files and names the verbs.
//
// It is a mistake filter only, and no claim rests on it: a shell line still
// reaches the file, and the reader is the enforcement. A file written past the
// verbs is judged like any other: the reader sets aside one git tracks, one
// another account owns or anyone else can read, one in a tier that is a link,
// and one that fails the schema or the secret scan, and falls back to the last
// good copy (checklist.Read). It cannot tell a file that passes those from one
// the verbs wrote.

import (
	"path"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/checklist"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// checklistStoreSuffixes are the two files only the verbs write, as the tail of
// a cleaned, folded, slash-separated path.
var checklistStoreSuffixes = []string{
	"/" + fsutil.FoldPath(checklist.FileRelPath, true),
	"/" + fsutil.FoldPath(checklist.LastGoodRelPath, true),
}

// ChecklistStorePath reports whether p names a working copy's checklist or its
// last good copy in the local tier, wherever the working copy stands. Case is
// folded, since the volume may fold it, and the path is cleaned, so a ".." or
// "." spelling of the same file is caught.
func ChecklistStorePath(p string) bool {
	if p == "" {
		return false
	}
	p = "/" + path.Clean(filepath.ToSlash(fsutil.FoldPath(p, true)))
	for _, s := range checklistStoreSuffixes {
		if len(p) >= len(s) && p[len(p)-len(s):] == s {
			return true
		}
	}
	return false
}

// ChecklistStoreRefusal is what a file tool refused ChecklistStorePath is told.
func ChecklistStoreRefusal() string {
	return "checklist.json and checklist.last-good.json are written only through the checklist verbs, " +
		"which check the whole list against its schema and scan every field for a secret before they write. " +
		"Run instead: `abcd checklist add \"<title>\"`, or `abcd checklist set <id> --state <state>`."
}
