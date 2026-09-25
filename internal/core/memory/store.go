package memory

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// storeHandle is an open memory store: the one way a verb reads
// .abcd/memory/ (iss-2608291814572914).
//
// Containment used to be a per-verb pre-check — each entry point called
// safeMemoryDir and then read by PATH — so a verb that forgot the check
// (fileBack did) read through a symlinked store, and even a verb that
// remembered it read by a path that a swap after the check could redirect.
// The handle holds the store directory open as an os.Root, reached through an
// os.Root on the repository, after the same segment walk refuses a symlinked
// or non-directory `.abcd` or `memory`. Every read then resolves inside that
// descriptor: a name cannot climb out of it, a symlink leaf is refused by
// fsutil.ReadGuardedInRoot, and replacing the directory after it was opened
// changes nothing the handle reads. A verb that has a handle cannot read the
// store any other way, which is what makes the containment structural rather
// than remembered.
//
// An absent store is a handle with no root: every read reports not-exist and
// every listing is empty, so callers need no separate branch for it.
type storeHandle struct {
	dir  string   // the canonical store path, for messages and for the writer
	root *os.Root // nil when the store is absent
}

// openStore opens the repository's memory store for reading. It never creates
// anything; a symlinked or non-directory store segment is refused with an
// *UnsafeStorePathError, exactly as the per-verb check refused it.
func openStore(repoRoot string) (*storeHandle, error) {
	mem, present, err := safeMemoryDir(repoRoot)
	if err != nil {
		return nil, err
	}
	h := &storeHandle{dir: mem}
	if !present {
		return h, nil
	}
	repo, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, err
	}
	defer repo.Close()
	// Opened THROUGH the repository root, so the store cannot resolve outside
	// the repository even if a segment was swapped after the walk above.
	root, err := repo.OpenRoot(filepath.Join(".abcd", "memory"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return h, nil
		}
		return nil, &UnsafeStorePathError{Msg: "memory store could not be opened inside the repository: " + mem}
	}
	// The directory opened must be the one the walk vetted: a symlink
	// swapped in between the two resolves to a different directory.
	opened, err := root.Stat(".")
	if err != nil {
		root.Close()
		return nil, err
	}
	vetted, err := os.Lstat(mem)
	if err != nil || vetted.Mode()&os.ModeSymlink != 0 || !os.SameFile(opened, vetted) {
		root.Close()
		return nil, &UnsafeStorePathError{Msg: "memory store changed while it was being opened: " + mem}
	}
	h.root = root
	if storeOpened != nil {
		storeOpened()
	}
	return h, nil
}

// storeOpened, when set, runs as openStore hands back a present store. It is a
// test seam and nothing else: it lets a test swap the store directory in the
// window after the handle was vetted, which is exactly where a read by path
// would follow the swap and a read through the handle would not. Nil outside
// tests.
var storeOpened func()

// Close releases the store descriptor. It is safe on an absent store.
func (h *storeHandle) Close() {
	if h != nil && h.root != nil {
		h.root.Close()
	}
}

// present reports whether the store directory exists.
func (h *storeHandle) present() bool { return h.root != nil }

// path is the absolute path of a store-relative name, for messages and
// findings only; nothing reads by it.
func (h *storeHandle) path(rel string) string {
	return filepath.Join(h.dir, filepath.FromSlash(rel))
}

// read reads one store file by its store-relative, slash-separated name:
// contained in the store, a symlink leaf refused, size-capped.
func (h *storeHandle) read(rel string, limit int64) ([]byte, error) {
	if h.root == nil {
		return nil, &fs.PathError{Op: "open", Path: h.path(rel), Err: fs.ErrNotExist}
	}
	return fsutil.ReadGuardedInRoot(h.root, filepath.FromSlash(rel), limit)
}

// readText is read for the small text files a status renders, reporting only
// whether the read succeeded.
func (h *storeHandle) readText(rel string) (string, bool) {
	raw, err := h.read(rel, maxMemoryPageBytes)
	if err != nil {
		return "", false
	}
	return string(raw), true
}

// pageNames lists the typed-page filenames at the store's top level, sorted.
// Only regular files are listed; a symlink entry is not a page.
func (h *storeHandle) pageNames() []string {
	if h.root == nil {
		return nil
	}
	entries, err := fs.ReadDir(h.root.FS(), ".")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.Type().IsRegular() && IsMemoryPageName(e.Name()) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// typedPages crawls the store for every typed memory page — a page-shaped
// .md file outside sources/ whose frontmatter carries a source block — and
// returns each once, read once, sorted by its store-relative name. The walk
// never descends a symlinked directory (fs.WalkDir does not follow them) and
// every read goes through the handle.
func (h *storeHandle) typedPages() []crawledPage {
	if h.root == nil {
		return nil
	}
	var pages []crawledPage
	_ = fs.WalkDir(h.root.FS(), ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() || !strings.HasSuffix(rel, ".md") {
			return nil
		}
		raw, err := h.read(rel, maxMemoryPageBytes)
		if err != nil {
			return nil
		}
		if isTypedMemoryPage(rel, string(raw)) {
			pages = append(pages, crawledPage{rel: rel, text: string(raw)})
		}
		return nil
	})
	sort.Slice(pages, func(i, j int) bool { return pages[i].rel < pages[j].rel })
	return pages
}

// isTypedMemoryPage is the typed-page gate over a page already read: a
// page-shaped filename that is not a sibling file, not under sources/, whose
// frontmatter parses and carries a source block.
func isTypedMemoryPage(rel, text string) bool {
	base := path.Base(rel)
	if siblingFiles[base] {
		return false
	}
	if _, _, _, ok := ParsePageFilename(base); !ok {
		return false
	}
	segs := strings.Split(rel, "/")
	for i := 0; i < len(segs)-1; i++ {
		if segs[i] == "sources" {
			return false
		}
	}
	fm, err := parseFrontmatter(text)
	if err != nil {
		return false
	}
	_, ok := fm["source"]
	return ok
}

// registry loads the sources index through the handle, with LoadRegistry's
// meaning: absent is an empty registry, and an unreadable or malformed one is
// a *RegistryFormatError.
func (h *storeHandle) registry() (map[string]any, error) {
	const name = ".sources_index.json"
	raw, err := h.read(name, maxRegistryBytes)
	return decodeRegistry(raw, err, h.path(name))
}
