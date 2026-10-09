package gitutil

import (
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// FilterRootsRelPath is the home-scoped declaration that switches the
// repository's content filters back on for Status in the checkouts it lists:
// ~/.abcd.noindex/filter-roots, one absolute checkout path per line, "#"
// starting a comment. It lives in the person's home, never in the repository,
// because a repository could otherwise switch its own filters on.
var FilterRootsRelPath = abcdhome.Rel("filter-roots")

// maxFilterRootsBytes caps the declaration read; a hand-kept list of
// checkouts is a few hundred bytes.
const maxFilterRootsBytes = 64 << 10

// FiltersSwitchedOn reports whether the person has listed root in
// ~/.abcd.noindex/filter-roots, switching the repository's content filters
// back on for Status there. The declaration is read through
// fsutil.HomeDeclarationNames, the reader every "declare this checkout"
// opt-in shares, so it is honoured only while it is a regular file this
// account owns that no one else can write, reached through no symlinked
// folder; ignored is the one-line note saying why a present declaration was
// not honoured, naming the file in tilde form, and is empty when there is none
// or it was read. Status has no output channel and drops it; FilterRootsIgnored
// is the reading a front door reports (iss-2610091920437492).
func FiltersSwitchedOn(root string) (on bool, ignored string) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return false, ""
	}
	on, why := fsutil.HomeDeclarationNames(home, FilterRootsRelPath, maxFilterRootsBytes, root, fsutil.CaseFoldingFS())
	if why != "" {
		return false, "IGNORED " + FilterRootsDisplay + " — " + why + "; content filters stay off in every checkout it lists"
	}
	return on, ""
}

// FilterRootsDisplay is the filter-roots declaration's path in tilde form, so a
// message naming it carries no home path.
var FilterRootsDisplay = abcdhome.Display("filter-roots")

// FilterRootsIgnored is the note FiltersSwitchedOn gives when a present
// ~/.abcd.noindex/filter-roots file fails its ownership, mode or symlink
// checks, naming the file and the check it failed; empty when the file is
// absent or was read. Which checkout is asked about does not change it: the
// checks are the file's, not an entry's.
func FilterRootsIgnored() string {
	_, note := FiltersSwitchedOn("")
	return note
}

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
//
// The repository's content filters are off (FilterOverrides): over a file
// whose saved stat no longer matches, git status re-reads it through
// filter.<name>.clean, a program the repository names, and these are everyday
// reads (iss-2610090821548169). The person switches them back on for a
// checkout by listing it in ~/.abcd.noindex/filter-roots (FiltersSwitchedOn).
// With them off, a file a filter would have rewritten can read as modified,
// and a filter the repository marks required fails the read.
//
// --ignore-submodules=dirty keeps git from starting a status inside each
// checked-out submodule, which reads the submodule's own config and so runs a
// clean filter FilterOverrides cannot see (iss-2610091935327982). The flag,
// not diff.ignoreSubmodules: a repository's submodule.<name>.ignore=none beats
// that config and loses to the flag. A moved submodule pointer is still
// listed; uncommitted content inside a submodule is not.
func Status(root string, maxBytes int, opt StatusOptions) ([]StatusEntry, error) {
	var args []string
	if on, _ := FiltersSwitchedOn(root); !on {
		filters, err := FilterOverrides(root)
		if err != nil {
			return nil, err
		}
		args = filters
	}
	args = append(args, "--no-optional-locks", "status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignore-submodules=dirty")
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
