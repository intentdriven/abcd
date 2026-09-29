package gitutil

import (
	"errors"
	"fmt"
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
// directory above it (an ignored directory drops everything beneath it). A
// directory is asked as `dir/`, the form archive itself asks: only a path that
// ends in a slash matches the directory-only pattern (`dir/ export-ignore`), a
// bare pattern (`dir export-ignore`) matches it too, and where the two forms
// disagree (`dir export-ignore` then `dir/ -export-ignore`) the answer for
// `dir/` is the one archive acts on. Asking `dir` as well would drop a
// directory archive keeps, and the scan would miss files the tag ships.
// Submodules are skipped: archive carries no submodule content.
//
// Attributes are read from rev's own .gitattributes, the files git archive
// reads, so neither a working-tree edit nor a staged one changes the answer
// (iss-2609260933592838). Where the index holds exactly rev's .gitattributes
// they are asked of the index (--cached), which every git answers; where they
// differ they are asked of rev itself (--source, git 2.40 or later), and a git
// that cannot is refused by name rather than answered from the index. An error
// is a tree git could not list — no commit at rev, not a repository, git
// absent — and is never reported as an empty tree.
func ArchiveTree(root, rev string) ([]ArchiveEntry, error) {
	out, err := isolatedGit(root, "ls-tree", "-r", "-z", "--full-tree", "--end-of-options", rev).Output()
	if err != nil {
		return nil, withStderr(err)
	}
	var entries []ArchiveEntry
	treeAttrs := map[string]string{}
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
		if path.Base(p) == ".gitattributes" {
			treeAttrs[p] = fields[0] + " " + fields[2]
		}
	}
	if len(entries) == 0 {
		return nil, nil
	}

	// Each file is asked as itself; each directory above one is asked as `dir/`.
	candidates := map[string]struct{}{}
	for _, e := range entries {
		candidates[e.Path] = struct{}{}
		for p := path.Dir(e.Path); p != "." && p != "/" && p != ""; p = path.Dir(p) {
			candidates[p+"/"] = struct{}{}
		}
	}
	list := make([]string, 0, len(candidates))
	for p := range candidates {
		list = append(list, p)
	}
	source, err := attrSource(root, rev, treeAttrs)
	if err != nil {
		return nil, err
	}
	cmd := isolatedGit(root, "check-attr", source, "-z", "--stdin", "export-ignore")
	cmd.Stdin = strings.NewReader(strings.Join(list, "\x00") + "\x00")
	attrs, err := cmd.Output()
	if err != nil {
		if source != "--cached" {
			return nil, fmt.Errorf("the index's .gitattributes differ from %s's, and reading %s's own needs "+
				"check-attr --source (git 2.40 or later): commit or unstage the .gitattributes change (%w)",
				rev, rev, withStderr(err))
		}
		return nil, withStderr(err)
	}
	ignored := map[string]struct{}{}
	fields := strings.Split(string(attrs), "\x00")
	// -z emits three fields per record: path, attribute, value.
	for i := 0; i+2 < len(fields); i += 3 {
		if fields[i+2] == "set" {
			ignored[strings.TrimSuffix(fields[i], "/")] = struct{}{}
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

// attrSource is the check-attr option that reads rev's .gitattributes: --cached
// when the index holds exactly the .gitattributes files rev does (same paths,
// modes and blobs, none conflicted), which any git answers, and --source=<rev>
// otherwise. treeAttrs maps each .gitattributes path in rev to "mode sha".
func attrSource(root, rev string, treeAttrs map[string]string) (string, error) {
	out, err := isolatedGit(root, "ls-files", "--stage", "-z", "--", ":(glob)**/.gitattributes").Output()
	if err != nil {
		return "", withStderr(err)
	}
	same := true
	n := 0
	for _, rec := range strings.Split(string(out), "\x00") {
		if rec == "" {
			continue
		}
		meta, p, ok := strings.Cut(rec, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 {
			return "", errors.New("git ls-files returned a record it does not document: " + rec)
		}
		n++
		if fields[2] != "0" || treeAttrs[p] != fields[0]+" "+fields[1] {
			same = false
		}
	}
	if same && n == len(treeAttrs) {
		return "--cached", nil
	}
	return "--source=" + rev, nil
}
