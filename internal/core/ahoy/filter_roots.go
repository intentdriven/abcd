package ahoy

import "github.com/intentdriven/abcd/internal/gitutil"

// FilterRootsIgnoredGapID is the note for a ~/.abcd.noindex/filter-roots file
// abcd ignored (iss-2610091920437492).
const FilterRootsIgnoredGapID = "filter_roots.ignored"

// detectFilterRoots reports a ~/.abcd.noindex/filter-roots file that abcd
// ignores because it fails its ownership, mode or symlink checks. Ignoring it
// switches the content filters of every checkout it lists back off, and the
// reads that consult it (gitutil.Status) have no output channel to say so, so
// the board does: a machine-scope fact, reported from any folder. It is a
// note, never a repair: abcd does not rewrite a declaration the person keeps.
func detectFilterRoots() []Gap {
	note := gitutil.FilterRootsIgnored()
	if note == "" {
		return nil
	}
	return []Gap{{
		ID: FilterRootsIgnoredGapID, Category: UserState, Scope: "machine",
		Title:  gitutil.FilterRootsDisplay + " ignored",
		Detail: note + ", so abcd's everyday reads run there with the repository's content filters off.",
		FixHint: "Make " + gitutil.FilterRootsDisplay + " a regular file you own that no one else can write, " +
			"not a symbolic link and not behind a symbolically linked folder (chmod 600 " + gitutil.FilterRootsDisplay +
			"), or remove it.",
	}}
}
