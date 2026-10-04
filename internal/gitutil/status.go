package gitutil

import "strings"

// StatusEntry is one entry of git's NUL-separated status listing
// (`git status --porcelain=v1 -z`).
type StatusEntry struct {
	// XY is the entry's two status columns as git writes them: "??" an
	// untracked path, "!!" an ignored one, otherwise the index column and the
	// working-tree column.
	XY string
	// Path is the path as git reports it, relative to the repository's root
	// and verbatim (the -z form never quotes or escapes it). A path git
	// reports whole as a directory ends in "/": an ignored directory under
	// StatusOptions.Ignored, or an untracked nested repository.
	Path string
	// Orig is a rename's or a copy's source path, and empty for every other
	// entry.
	Orig string
}

// StatusOptions are the variants of the status listing Status runs.
type StatusOptions struct {
	// Ignored lists ignored paths too (--ignored=matching): a file that
	// matches an ignore pattern by itself, and a directory that matches one
	// as the directory alone, never what is inside it.
	Ignored bool
	// Pathspecs, when any, narrows the listing to them.
	Pathspecs []string
}

// Status is the working tree's status under root, as git lists it: `git
// --no-optional-locks status --porcelain=v1 -z --untracked-files=all`, run
// through the isolated environment and refused, never truncated, past
// maxBytes. It is the one reader of that listing; a caller that needs a
// variant adds it to StatusOptions.
//
// --untracked-files=all is fixed: under git's default an untracked directory
// collapses to one entry, and a file inside a new directory is never named.
// --no-optional-locks keeps the read from refreshing the index, which the
// isolated environment's GIT_OPTIONAL_LOCKS=0 already does; it is stated on
// the command so the read stays read-only whatever environment runs it.
func Status(root string, maxBytes int, opt StatusOptions) ([]StatusEntry, error) {
	args := []string{"--no-optional-locks", "status", "--porcelain=v1", "-z", "--untracked-files=all"}
	if opt.Ignored {
		args = append(args, "--ignored=matching")
	}
	if len(opt.Pathspecs) > 0 {
		args = append(append(args, "--"), opt.Pathspecs...)
	}
	out, err := RunCappedBytes(root, maxBytes, args...)
	if err != nil {
		return nil, err
	}
	return ParseStatus(out), nil
}

// ParseStatus parses git's NUL-separated status listing.
//
// The -z form is what makes it parseable: each entry is NUL-terminated and
// its path is never quoted or escaped, so a filename holding a space, a quote
// or a newline arrives verbatim and core.quotePath cannot change the format
// under the parser. A rename's or a copy's source follows its entry as a
// record of its own, and either column can declare one (`R ` a staged rename,
// ` R` a working-tree one); it is carried as Orig, never read as an entry.
func ParseStatus(out []byte) []StatusEntry {
	records := strings.Split(string(out), "\x00")
	var entries []StatusEntry
	for i := 0; i < len(records); i++ {
		rec := records[i]
		if len(rec) < 4 {
			continue
		}
		e := StatusEntry{XY: rec[:2], Path: rec[3:]}
		if x, y := e.XY[0], e.XY[1]; x == 'R' || x == 'C' || y == 'R' || y == 'C' {
			i++
			if i < len(records) {
				e.Orig = records[i]
			}
		}
		entries = append(entries, e)
	}
	return entries
}
