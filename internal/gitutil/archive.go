package gitutil

import (
	"errors"
	"path"
	"strings"
)

// ArchiveEntry is one file `git archive` would write for a revision: its
// repo-relative path and its git mode (100644, 100755 or 120000 for a link).
type ArchiveEntry struct {
	Path string
	Mode string
}

// ArchiveTree lists the files an archive of rev would carry — the release tag's
// view of the tree — without running `git archive` itself. `git archive` applies
// the repository's filter drivers, and a checkout's own .git/config is fully
// trusted by git, so the listing is built from the two commands that run no
// configured program: `ls-tree` for the committed files, and `check-attr` for the
// export-ignore attribute archive honours, asked of each file and of every
// directory above it (an ignored directory drops everything beneath it).
// Submodules are skipped: archive carries no submodule content.
//
// Attributes are read from the index (--cached), the committed view, so an
// uncommitted .gitattributes edit does not change the answer. An error is a tree
// git could not list — no commit at rev, not a repository, git absent — and is
// never reported as an empty tree.
func ArchiveTree(root, rev string) ([]ArchiveEntry, error) {
	out, err := isolatedGit(root, "ls-tree", "-r", "-z", "--full-tree", rev).Output()
	if err != nil {
		return nil, withStderr(err)
	}
	var entries []ArchiveEntry
	for _, rec := range strings.Split(string(out), "\x00") {
		if rec == "" {
			continue
		}
		meta, p, ok := strings.Cut(rec, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 {
			return nil, errors.New("git ls-tree returned a record it does not document: " + rec)
		}
		if fields[1] != "blob" {
			continue // a submodule (commit) carries no content into an archive
		}
		entries = append(entries, ArchiveEntry{Path: p, Mode: fields[0]})
	}
	if len(entries) == 0 {
		return nil, nil
	}

	candidates := map[string]struct{}{}
	for _, e := range entries {
		for p := e.Path; p != "." && p != "/" && p != ""; p = path.Dir(p) {
			candidates[p] = struct{}{}
		}
	}
	list := make([]string, 0, len(candidates))
	for p := range candidates {
		list = append(list, p)
	}
	cmd := isolatedGit(root, "check-attr", "--cached", "-z", "--stdin", "export-ignore")
	cmd.Stdin = strings.NewReader(strings.Join(list, "\x00") + "\x00")
	attrs, err := cmd.Output()
	if err != nil {
		return nil, withStderr(err)
	}
	ignored := map[string]struct{}{}
	fields := strings.Split(string(attrs), "\x00")
	// -z emits three fields per record: path, attribute, value.
	for i := 0; i+2 < len(fields); i += 3 {
		if fields[i+2] == "set" {
			ignored[fields[i]] = struct{}{}
		}
	}
	kept := entries[:0]
	for _, e := range entries {
		drop := false
		for p := e.Path; p != "." && p != "/" && p != ""; p = path.Dir(p) {
			if _, ok := ignored[p]; ok {
				drop = true
				break
			}
		}
		if !drop {
			kept = append(kept, e)
		}
	}
	return kept, nil
}
