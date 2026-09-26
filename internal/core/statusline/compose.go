package statusline

// Compose is the one composition of the row, in core, for both front doors:
// the status verb the harness invokes and the bare `/abcd` board. It gathers
// the three inputs spc-70 names — the parsed harness payload, the repository,
// and the record — into an Input and hands it to Render, so the verb and the
// board cannot disagree about a single element.
//
// It reads, and that is the whole of what separates it from Render: the mode
// store, git's answer for the branch, and one directory listing for each of
// the two counts. It writes nothing, and it never touches the network. It also
// never asks whether the repository is MANAGED: that decision belongs to the
// caller, because the package that can answer it (ahoy) is a consumer of this
// one, and a composition that imported its own consumer would be a cycle.
//
// THE COUNTS ARE FOLDER COUNTS, NOT PARSES. The harness runs the status verb
// on every refresh, and a composition that loaded and parsed every open issue
// and every intent and spec — the board's readers — was measured at 0.96 s
// over 20,001 open records: a status line that costs that is a status line
// people switch off (the argument ahoy's Managed makes for its own read).
// Folder membership is the record's own status signal — an issue is open
// because it is in open/, an intent is unshipped because it is in drafts/ or
// planned/ — so a count of the files named for the family in each folder is
// the same number the board's readers give, except for a record those readers
// would skip as unreadable, which the folder still holds. One guarded readdir
// per folder, no parse, no subprocess (2026-09-15).

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Result is what Compose returns: the state the badge reports, the row, and
// the notes the composition had to make on the way.
//
// Notes are the out-of-band channel the rest of core uses (rules.Resolve,
// history.Resolve, LoadFrom here): a count the readers could not take is
// dropped from the row and NAMED, because a row that silently lost an element
// is exactly the plausible-wrong-answer the drop rule is meant to make
// visible. Core never prints; the front door puts them on stderr.
type Result struct {
	State State    `json:"state"`
	Row   Row      `json:"row"`
	Notes []string `json:"-"`
}

// RecordRelDir is the durable record's root, repo-relative. Its presence is
// what says a repository has a record to count: without it the two count
// elements are switched off rather than rendered as zeros nobody means.
const RecordRelDir = ".abcd/development"

// Compose renders the row for an already-resolved checkout root.
//
// The caller resolves the root (gitutil.CheckoutRoot) and decides whether the
// repository is managed before calling; Compose takes the root as given. The
// state is read from the mode store, and a store that cannot be read — a
// fourth word, a symlink, a device — is an ERROR rather than a quiet badge,
// for the reason mode.ReadAt gives: somebody wrote something there, and
// reporting "nobody is waiting" over it would hide the parked stop the badge
// exists to show.
//
// The branch comes from git: the symbolic ref's short name, or the short sha
// on a detached HEAD, or nothing (the element drops) when git will not answer.
// The counts are the record's folder counts (see the package comment), and are
// switched off in a repository that has no record. The settings are copied
// before any switch is thrown, so the caller's own Settings are never mutated.
func Compose(root string, p Payload, set Settings) (Result, error) {
	state, err := mode.ReadAt(root)
	if err != nil {
		return Result{}, err
	}
	in := Input{
		State:   state,
		Repo:    Repo{Name: filepath.Base(root), Branch: branch(root)},
		Payload: p,
	}

	set = clone(set)
	var notes []string
	if fsutil.IsRealDir(filepath.Join(root, filepath.FromSlash(RecordRelDir))) {
		if n, err := openIssues(root); err != nil {
			set.Elements[KeyIssues] = false
			notes = append(notes, dropped(KeyIssues, "issue ledger", root, err))
		} else {
			in.Counts.Issues = n
		}
		if n, err := unshippedIntents(root); err != nil {
			set.Elements[KeyIntents] = false
			notes = append(notes, dropped(KeyIntents, "intent store", root, err))
		} else {
			in.Counts.Intents = n
		}
	} else {
		set.Elements[KeyIssues] = false
		set.Elements[KeyIntents] = false
	}

	return Result{State: state, Row: Render(in, set), Notes: notes}, nil
}

// branch is the checkout's branch name, the short sha when HEAD is detached,
// or "" when git will not answer — in which case the element drops, the same
// answer an absent payload field gets.
func branch(root string) string {
	if name, err := gitutil.Run(root, "symbolic-ref", "--quiet", "--short", "HEAD"); err == nil && name != "" {
		return name
	}
	if sha, err := gitutil.Run(root, "rev-parse", "--short", "HEAD"); err == nil {
		return sha
	}
	return ""
}

// openIssues counts the ledger's open folder: the regular files named
// iss-*.md directly in open/. The folder is the ledger's own path constant and
// the open state's own directory name, the first of the status folders —
// nothing here restates where the ledger is. Both are read from the leaves
// core/capture derives them from (recordid, issueschema), not from capture
// itself: capture reads the site package's section walk, the site package
// reaches ahoy, and ahoy reads this package, so an import of capture here
// closes a cycle.
func openIssues(root string) (int, error) {
	return countRecords(filepath.Join(root, filepath.FromSlash(recordid.IssuesRelDir), issueschema.StatusDirs[0]), "iss-")
}

// unshippedIntents counts the intents not yet shipped: the itd-*.md files in
// the drafts bucket plus the planned one, the intent store's own two names.
func unshippedIntents(root string) (int, error) {
	n := 0
	for _, bucket := range []string{intent.BucketDrafts, intent.BucketPlanned} {
		c, err := countRecords(filepath.Join(root, filepath.FromSlash(intent.IntentsRelDir), bucket), "itd-")
		if err != nil {
			return 0, err
		}
		n += c
	}
	return n, nil
}

// countRecords is one guarded readdir: the number of regular files in dir
// whose name carries the family's prefix and the record suffix. A folder that
// does not exist holds no records and counts zero, which is a fact about the
// record rather than a failure. Anything else standing where the folder
// should be is an error the caller drops the count over: the directory is
// opened O_NOFOLLOW|O_DIRECTORY, so a symlinked folder (ELOOP) is refused
// rather than followed — a checkout can commit a link to anywhere, and a
// count taken through it would be a count of somewhere else — and a file
// (ENOTDIR) is refused rather than read. One open, no lstat-then-open window.
// A symlinked ENTRY is not a regular file and is simply not counted, the same
// answer the board's readers give it.
func countRecords(dir, prefix string) (int, error) {
	f, err := os.OpenFile(dir, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_DIRECTORY, 0)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return 0, nil
	case errors.Is(err, syscall.ELOOP):
		return 0, fmt.Errorf("%s is a symlink; the count reads only a real directory", dir)
	case errors.Is(err, syscall.ENOTDIR):
		return 0, fmt.Errorf("%s is not a directory", dir)
	case err != nil:
		return 0, err
	}
	defer f.Close()
	entries, err := f.ReadDir(-1)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if e.Type().IsRegular() && strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".md") {
			n++
		}
	}
	return n, nil
}

// dropped phrases the note for a count that could not be taken. The error
// carries the absolute path of the folder it refused; it is stripped to the
// repo-relative form and sanitised before it can reach a terminal, so no
// diagnostic carries a local path (iss-76, iss-81).
func dropped(k ElementKey, store, root string, err error) string {
	msg := err.Error()
	if root != "" {
		msg = strings.ReplaceAll(msg, root+string(os.PathSeparator), "")
		msg = strings.ReplaceAll(msg, root, ".")
	}
	if errors.Is(err, os.ErrPermission) {
		msg = "permission denied"
	}
	return fmt.Sprintf("statusline: DROPPED the %s count — the %s could not be read (%s)",
		k, store, termsafe.CleanProseLine(msg, 200))
}
