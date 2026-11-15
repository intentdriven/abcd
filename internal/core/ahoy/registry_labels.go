package ahoy

// RegistryLabel is one repository's labels in ~/.abcd.noindex/history/index.json:
// the name it is shown under and the path the registry last recorded for its
// checkout. Both are mutable labels on an entry keyed by the root commit; the
// path may name a directory that no longer exists.
type RegistryLabel struct {
	Name string
	Path string
}

// RegistryLabels maps each root commit registered in
// ~/.abcd.noindex/history/index.json to its labels, read through
// loadHistoryIndex, the one loader that scrubs the registry on the way in and
// refuses a symlinked ~/.abcd.noindex or history directory. It is the worktree
// store's read of the registry (spc-2609301811532881): `abcd ahoy worktree
// list --all` shows a lane under its repository's name where its root commit
// is registered (itd-2609091014076309's criterion 7).
//
// An absent registry is an empty map and no error. A registry that cannot be
// read or parsed is an empty map and the error, which a caller that only
// labels rows drops, so each lane stays under its root commit. An entry with
// no root commit or no name is left out; where two entries share a root
// commit, the first with a name wins, as indexEntry finds the first.
func RegistryLabels() (map[string]RegistryLabel, error) {
	out := map[string]RegistryLabel{}
	idx, err := loadHistoryIndex()
	if err != nil || idx == nil {
		return out, err
	}
	for _, r := range idx.Repos {
		if r.RootCommit == "" || r.Name == "" {
			continue
		}
		if _, seen := out[r.RootCommit]; seen {
			continue
		}
		out[r.RootCommit] = RegistryLabel{Name: r.Name, Path: r.Path}
	}
	return out, nil
}
