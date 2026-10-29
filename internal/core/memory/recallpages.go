package memory

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// recallpages.go — the read the prompt router's memory recall makes of the
// pages (spc-2610111627213138 step 3): each page's filename, its index line's
// summary, its body and its recall: list, from the workspace store and, read
// only, from the machine's ~/.abcd.noindex/memory/. Nothing here writes.

// RecallPage is one memory page as the recall reads it.
type RecallPage struct {
	// Filename is the page's <type>_<domain>_<slug>.md name in its store.
	Filename string
	// Summary is the page's index.md summary, derived from the page as
	// RenderIndex derives it, so the recall reads no index.md.
	Summary string
	// Body is the page's text after its frontmatter, less the line Summary
	// comes from, trimmed: the title is not counted, or sent, twice.
	Body string
	// Recall is the page's recall: frontmatter list, empty when it has none.
	Recall []string
	// IngestedAt is the latest ingested_at the page's source block records
	// (one source's, or the latest of several), the zero time when it records
	// none. It is the page's own record of when it was written, which travels
	// with it; the file's modification time is whenever a checkout wrote it.
	IngestedAt time.Time
}

// RecallRefusal names a scope or a page the read refused, and why. File is
// the store-relative or tilde form, never an expanded home; Filename is the
// refused page's own name, "" when the whole scope was refused.
type RecallRefusal struct {
	File     string
	Filename string
	Err      error
}

// machineStoreDisplay is the machine page scope as a message names it.
var machineStoreDisplay = abcdhome.Display("memory") + "/"

// workspaceStoreDisplay is the workspace page scope as a message names it.
const workspaceStoreDisplay = ".abcd/memory/"

// ReadRecallPages reads the workspace store under repoRoot through the store
// handle: a symlinked or swapped store is refused whole, each page is read
// through the handle's guarded read. An absent store yields nothing. The
// pages read are those index.md is rendered from (the store's top-level page
// files, storeHandle.pageNames) whose names are <type>_<domain>_<slug>.md;
// index.md itself is never read.
func ReadRecallPages(repoRoot string) ([]RecallPage, []RecallRefusal) {
	store, err := openStore(repoRoot)
	if err != nil {
		return nil, []RecallRefusal{{File: workspaceStoreDisplay, Err: err}}
	}
	defer store.Close()
	return recallPagesFrom(store, workspaceStoreDisplay, store.read)
}

// ReadMachineRecallPages reads the machine scope, ~/.abcd.noindex/memory/
// under home, read only, with the guards the machine memory.json is read
// with: the directory is opened through fsutil.OpenHomeDeclarationScope, so a
// symlinked ~/.abcd.noindex or memory/ is refused, and so is one another
// account can change; each page is read through the declaration's guarded
// read relative to that descriptor, a regular file this account owns that no
// one else can write. An absent scope, no absolute home, or a home or
// ~/.abcd.noindex this account cannot search yields nothing, whatever the
// directories above it are.
func ReadMachineRecallPages(home string) ([]RecallPage, []RecallRefusal) {
	if home == "" || !filepath.IsAbs(home) {
		return nil, nil
	}
	// Absence is decided before any directory is judged, as
	// ReadHomeDeclaration decides it, so an exposed ~/.abcd.noindex that
	// holds no page scope costs nothing and says nothing.
	if _, err := os.Lstat(abcdhome.Path(home, "memory")); err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) || errors.Is(err, fs.ErrPermission) {
			return nil, nil
		}
		return nil, []RecallRefusal{{File: machineStoreDisplay, Err: recallReadError(err)}}
	}
	root, err := fsutil.OpenHomeDeclarationScope(home, abcdhome.Rel("memory"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) || errors.Is(err, fs.ErrPermission) {
			return nil, nil
		}
		return nil, []RecallRefusal{{File: machineStoreDisplay, Err: err}}
	}
	store := &storeHandle{dir: abcdhome.Path(home, "memory"), root: root}
	defer store.Close()
	read := func(name string, limit int64) ([]byte, error) {
		data, _, err := fsutil.ReadHomeDeclarationDenyingIn(root, home, abcdhome.Rel("memory", name), limit, 0)
		return data, err
	}
	return recallPagesFrom(store, machineStoreDisplay, read)
}

// The recall runs on every prompt, over a store of any size,
// so what one prompt reads of a scope is bounded: no page larger than a
// recall could send (a block over the injection budget is set aside whole),
// and no more pages, or bytes, than the bounds below. The bounds keep the
// hook's worst case within its budget; real stores sit far below them.
const (
	// maxRecallPageBytes is twice the injection budget (64 KiB): a page past
	// it can never come back, so its body is never read for ranking.
	maxRecallPageBytes = 128 << 10
	// maxRecallPagesPerScope and maxRecallBytesPerScope bound one scope's
	// pages per prompt, read in name order.
	maxRecallPagesPerScope = 256
	maxRecallBytesPerScope = 1 << 20
)

// recallPagesFrom reads, through read, the pages the store lists whose name
// is <type>_<domain>_<slug>.md, in name order and within the bounds above; a
// page read refuses is named, never followed, and pages past a bound are
// named once.
func recallPagesFrom(store *storeHandle, shown string, read func(string, int64) ([]byte, error)) ([]RecallPage, []RecallRefusal) {
	var pages []RecallPage
	var refused []RecallRefusal
	var bytesRead, tried int
	for _, name := range store.pageNames() {
		if _, _, _, ok := ParsePageFilename(name); !ok {
			continue
		}
		// Every page tried counts, a refused one included, so a store of
		// pages the recall refuses cannot make it open every one of them.
		if tried >= maxRecallPagesPerScope || bytesRead >= maxRecallBytesPerScope {
			refused = append(refused, RecallRefusal{File: shown, Err: fmt.Errorf("it holds more pages than one prompt's recall reads (%d pages or %d bytes); from %s on, its pages are not recalled",
				maxRecallPagesPerScope, maxRecallBytesPerScope, name)})
			break
		}
		tried++
		raw, err := read(name, maxRecallPageBytes)
		if err != nil {
			refused = append(refused, RecallRefusal{File: shown + name, Filename: name, Err: recallReadError(err)})
			continue
		}
		if bytesRead+len(raw) > maxRecallBytesPerScope {
			refused = append(refused, RecallRefusal{File: shown, Err: fmt.Errorf("it holds more pages than one prompt's recall reads (%d pages or %d bytes); from %s on, its pages are not recalled",
				maxRecallPagesPerScope, maxRecallBytesPerScope, name)})
			break
		}
		bytesRead += len(raw)
		p := parsePage(string(raw))
		info := pageInfoOf(name, p)
		pages = append(pages, RecallPage{Filename: name, Summary: info.Summary, Body: bodyAfterSummary(p.body), Recall: recallList(p.fm),
			IngestedAt: latestIngest(p.source())})
	}
	return pages, refused
}

// bodyAfterSummary is body less its first non-blank line, the one
// pageInfoOf takes the summary from, trimmed.
func bodyAfterSummary(body string) string {
	rest := strings.TrimLeft(body, "\r\n\t ")
	if i := strings.IndexByte(rest, '\n'); i >= 0 {
		return strings.TrimSpace(rest[i+1:])
	}
	return ""
}

// recallList is a page's recall: list, its non-blank string entries trimmed.
func recallList(fm map[string]any) []string {
	raw, ok := fm["recall"].([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, v := range raw {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

// latestIngest is the latest ingested_at date a source block records, in
// its single-source or multi-source shape; the zero time when none parses.
func latestIngest(source map[string]any) time.Time {
	entries := []any{source}
	if list, ok := source["sources"].([]any); ok {
		entries = append(entries, list...)
	}
	var latest time.Time
	for _, e := range entries {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		s, ok := m["ingested_at"].(string)
		if !ok {
			continue
		}
		if at, err := time.Parse(time.DateOnly, s); err == nil && at.After(latest) {
			latest = at
		}
	}
	return latest
}

// recallReadError words a refused page read without the expanded path.
func recallReadError(err error) error {
	switch {
	case errors.Is(err, fsutil.ErrTooBig):
		return fmt.Errorf("it is larger than the %d bytes a recall reads, so it is never recalled", maxRecallPageBytes)
	case errors.Is(err, fsutil.ErrNotRegular):
		return errors.New("it is not a regular file")
	case errors.Is(err, fsutil.ErrDeclarationWritable):
		return errors.New("it is writable by others, so its words are not necessarily yours (chmod go-w it)")
	case errors.Is(err, fsutil.ErrDeclarationForeignOwner):
		return errors.New("it is not owned by this session's uid")
	}
	var pe *os.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}
