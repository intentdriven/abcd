package memory

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// writer.go — the atomic single-writer for .abcd/memory/ (ADR-13 §1–3). EVERY
// memory-store mutation goes through WritePages: advisory flock, six-step
// durable write per file, idempotent sibling reconciliation, registry-driven
// orphan prune. No journal, no crash recovery — per-file atomicity plus
// self-correcting reconciliation before every mutating write.

// PageWrite is one page to materialise: <type>_<domain>_<slug>.md, full
// frontmatter (must carry a schema-valid source: block), and the markdown body.
type PageWrite struct {
	Filename    string
	Frontmatter map[string]any
	Body        string
}

// WriteReport records what one WritePages call did (all lists sorted).
type WriteReport struct {
	CreatedSiblings []string `json:"created_siblings"`
	Backfilled      []string `json:"backfilled"`
	Pruned          []string `json:"pruned"`
	Reconciled      []string `json:"reconciled"`
	WrotePages      []string `json:"wrote_pages"`
}

type renderedWrite struct {
	write PageWrite
	text  string
}

// WithStoreLock holds the exclusive non-blocking advisory lock on
// .abcd/memory/.lock for fn's extent. The closure form keeps the locked/unlocked
// split structural — flock does not nest.
func WithStoreLock(repoRoot string, fn func() error) error {
	memDir, err := validatedMemoryDir(repoRoot)
	if err != nil {
		return err
	}
	path := filepath.Join(memDir, ".lock")

	if fi, err := os.Lstat(path); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
			return &UnsafeStorePathError{Msg: "memory store lock path is a symlink or non-regular file: " + path}
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return &UnsafeStorePathError{Msg: "unsafe memory store lock open for " + path + ": " + err.Error()}
	}
	defer syscall.Close(fd)

	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		return err
	}
	if !lockModeIsRegular(uint32(st.Mode)) {
		return &UnsafeStorePathError{Msg: "memory store lock fd is not a regular file: " + path}
	}
	if st.Nlink < 1 {
		return &UnsafeStorePathError{Msg: "memory store lock fd has zero links: " + path}
	}

	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return &StoreLockHeldError{Path: path}
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)

	return fn()
}

// lockModeIsRegular reports whether a stat mode names a regular file. The
// file-type field is an enumeration under S_IFMT, not a set of flags: a socket
// and a symlink both carry the S_IFREG bit, so testing that bit alone admitted
// them (iss-2608261133210491).
func lockModeIsRegular(mode uint32) bool {
	return mode&syscall.S_IFMT == syscall.S_IFREG
}

// memoryDir is the ONE walk that resolves <repoRoot>/.abcd/memory for every
// entry point, read or write. Each owned segment is lstat-refused as a symlink
// or non-directory, so a committed `.abcd/memory` DIRECTORY symlink (git mode
// 120000 pointing at an out-of-repo tree) can neither be walked, read, nor
// written into (GHSA-72rp-qxm2-r8vq). The leaf-guarded reads (fsutil.ReadGuarded,
// O_NOFOLLOW) only bind the LEAF; they do NOT contain a symlinked ANCESTOR
// directory, which is exactly the shape this guards. Callers resolve the store
// through this once, up front, and refuse on error before any
// WalkDir/ReadDir/ReadFile/MkdirAll touches the path.
//
// create says what a missing segment means: the write side materialises it,
// the read/lint side must not (a read or a health-check never creates the
// store) and reports present=false instead. The returned path is ALWAYS the
// canonical Dir(repoRoot), present or not, so a caller never re-derives it.
// One walker rather than a read copy and a write copy, because this is the
// guard: two copies means a hardening fix to one leaves the other open.
func memoryDir(repoRoot string, create bool) (string, bool, error) {
	current := repoRoot
	for _, segment := range []string{".abcd", "memory"} {
		current = filepath.Join(current, segment)
		fi, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				if !create {
					return Dir(repoRoot), false, nil
				}
				if err := os.Mkdir(current, 0o755); err != nil {
					return Dir(repoRoot), false, err
				}
				continue
			}
			return Dir(repoRoot), false, err
		}
		if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
			return Dir(repoRoot), false, &UnsafeStorePathError{Msg: "memory store segment is a symlink or non-directory: " + current}
		}
	}
	return Dir(repoRoot), true, nil
}

// validatedMemoryDir is the write side of memoryDir: it returns the store,
// creating a missing segment.
func validatedMemoryDir(repoRoot string) (string, error) {
	mem, _, err := memoryDir(repoRoot, true)
	return mem, err
}

// safeMemoryDir is the read/lint side of memoryDir: it never creates, and
// reports present=false when the store (or an ancestor) is simply absent — the
// path it returns is still the canonical one, so a caller's downstream joins
// stay absolute and fail cleanly as absent.
func safeMemoryDir(repoRoot string) (string, bool, error) {
	return memoryDir(repoRoot, false)
}

// RegistryMerge recomputes the COMPLETE new registry from the registry as
// freshly read under the store lock. WritePages calls it with the on-disk
// registry loaded INSIDE the lock, so the merged result is never derived from a
// stale pre-lock snapshot. This closes the load-merge-write lost-update: two
// concurrent ingests that each loaded the registry before either wrote would, if
// each wrote its own wholesale pre-computed registry, have the last writer
// silently clobber the other's entry (and orphan its pages). Re-running the
// merge against the locked read makes each write additive. A nil RegistryMerge
// leaves the registry untouched (a heal-only pass).
type RegistryMerge func(current map[string]any) (map[string]any, error)

// WritePages is the single mutating entry point for .abcd/memory/. It acquires
// the advisory store lock, runs the full locked sequence (skeleton ensure ->
// legacy backfill -> orphan prune -> pre-reconcile -> page writes -> registry
// load+merge+write when given -> post-reconcile -> log append), and returns a
// report. An empty writes slice is a valid heal-only pass. merge may be nil (no
// registry mutation); when non-nil it is invoked with the registry read fresh
// under the lock and must return the COMPLETE new mapping. A zero now is sampled
// as time.Now().UTC() AFTER the lock is held.
func WritePages(repoRoot string, writes []PageWrite, merge RegistryMerge, now time.Time) (WriteReport, error) {
	// Redact every page BODY here, in the one primitive every verb writes
	// through, so no body reaches the committed store unscanned whichever verb
	// built it (GHSA-j5f5-phgm-9m73). A redactor that stood at one call site
	// left the next verb's bodies — a host-delegated distiller's file-back
	// page — landing raw. index.md and log.md are derived from these bodies
	// (reconcile reads the written files; the log event is derived from the
	// PageWrite body), so redacting here covers those derived surfaces too.
	//
	// Every string leaf of the FRONTMATTER goes through the same detector
	// (GHSA-x46m-mw9h-5jwj, iss-2608291941064448): a host-supplied citation
	// title, recall entry, contradicts target, sources[] licence or weighting
	// note was rendered verbatim by renderWrites, and contradictions.md is
	// derived from it. So do the REGISTRY leaves the merge introduces, judged
	// against the registry as read under the lock (writePagesLocked) — which
	// is why the redactor is built whenever there is a merge, not only when
	// there are pages: the registry-only fast path used to build none and
	// wrote its fill-if-empty origin raw. It fails closed on a degraded
	// scanner; a heal-only pass (no pages, no merge) writes no acquired text
	// and needs no scanner, so the check is skipped only then.
	var redactor *storeRedactor
	if len(writes) > 0 || merge != nil {
		r, err := newStoreRedactor(repoRoot)
		if err != nil {
			return WriteReport{}, err
		}
		redactor = r
	}
	if len(writes) > 0 {
		redacted := make([]PageWrite, 0, len(writes))
		for _, w := range writes {
			// The FILENAME first, and refused rather than rewritten
			// (iss-2609020321100138). It is judged HERE, before renderWrites
			// and before the store lock, because that placement covers by
			// construction every place the slug lands: the page file and the
			// log event are derived from this same rendered write, index.md is
			// reconciled from the files on disk, and the registry back-link is
			// merged inside writePagesLocked — none of which is reached once
			// this returns an error.
			if err := redactor.judgeFilename(w.Filename); err != nil {
				return WriteReport{}, err
			}
			body, _, rerr := redactor.redactText(w.Body, w.Filename)
			if rerr != nil {
				return WriteReport{}, rerr
			}
			w.Body = body
			// The caller's map is never mutated; a nil frontmatter is left for
			// renderWrites to refuse with its contract error.
			if w.Frontmatter != nil {
				fm := deepCopyMap(w.Frontmatter)
				if err := redactor.redactLeaves(nil, fm, w.Filename); err != nil {
					return WriteReport{}, err
				}
				w.Frontmatter = fm
			}
			redacted = append(redacted, w)
		}
		writes = redacted
	}
	rendered, err := renderWrites(writes)
	if err != nil {
		return WriteReport{}, err
	}
	var report WriteReport
	lockErr := WithStoreLock(repoRoot, func() error {
		r, err := writePagesLocked(repoRoot, rendered, merge, redactor, now)
		if err != nil {
			return err
		}
		report = r
		return nil
	})
	if lockErr != nil {
		return WriteReport{}, lockErr
	}
	return report, nil
}

func writePagesLocked(repoRoot string, rendered []renderedWrite, merge RegistryMerge, redactor *storeRedactor, now time.Time) (WriteReport, error) {
	mem := Dir(repoRoot)
	if now.IsZero() {
		now = time.Now().UTC()
	}
	stamp := now.Format("2006-01-02 15:04")

	created, err := ensureSkeleton(mem)
	if err != nil {
		return WriteReport{}, err
	}
	backfilled, err := backfillLegacy(mem)
	if err != nil {
		return WriteReport{}, err
	}
	pruned, err := pruneOrphans(repoRoot, mem)
	if err != nil {
		return WriteReport{}, err
	}
	healed, err := reconcile(mem)
	if err != nil {
		return WriteReport{}, err
	}

	var wrote []string
	for _, r := range rendered {
		if err := writeStringAtomic(filepath.Join(mem, r.write.Filename), r.text); err != nil {
			return WriteReport{}, err
		}
		wrote = append(wrote, r.write.Filename)
	}

	if merge != nil {
		// Load the registry fresh UNDER the lock, then apply the caller's merge
		// to that authoritative read — never to a snapshot taken before the lock
		// was held (the lost-update fix).
		current, err := LoadRegistry(SourcesIndexPath(repoRoot))
		if err != nil {
			return WriteReport{}, err
		}
		// The baseline for "what did this write introduce" is a copy taken
		// BEFORE the merge runs: MergeIngest copies its input, but a merge is
		// free to mutate the map it was given and hand it back, and comparing
		// the result against itself would judge nothing.
		baseline := deepCopyMap(current)
		registry, err := merge(current)
		if err != nil {
			return WriteReport{}, err
		}
		// Every leaf the merge introduced — a fresh entry's origin, licence and
		// citation, a fill-if-empty origin, a backlinked or file-back consumer's
		// citation — is judged here; a leaf the registry already held is not
		// (GHSA-x46m-mw9h-5jwj). In place, so the merged map the caller may
		// still hold is the written one.
		if redactor == nil {
			return WriteReport{}, newWriterContractError("registry merge without a store redactor")
		}
		if err := redactor.redactRegistryLeaves(baseline, registry, filepath.Base(SourcesIndexPath(repoRoot))); err != nil {
			return WriteReport{}, err
		}
		if err := writeStringAtomic(SourcesIndexPath(repoRoot), SerializeRegistry(registry)); err != nil {
			return WriteReport{}, err
		}
	}

	post, err := reconcile(mem)
	if err != nil {
		return WriteReport{}, err
	}

	if len(rendered) > 0 {
		var events []string
		for _, r := range rendered {
			events = append(events, logEvent(r.write, stamp))
		}
		if err := appendLog(mem, events); err != nil {
			return WriteReport{}, err
		}
	}

	reconciled := sortedUnion(healed, post)
	sort.Strings(created)
	sort.Strings(backfilled)
	sort.Strings(pruned)
	sort.Strings(wrote)
	return WriteReport{
		CreatedSiblings: created,
		Backfilled:      backfilled,
		Pruned:          pruned,
		Reconciled:      reconciled,
		WrotePages:      wrote,
	}, nil
}

// ---------------------------------------------------------------------------
// Validation / rendering (no I/O — runs before the lock)
// ---------------------------------------------------------------------------

func renderWrites(writes []PageWrite) ([]renderedWrite, error) {
	var rendered []renderedWrite
	seen := map[string]bool{}
	for _, w := range writes {
		if err := validatePageFilename(w.Filename); err != nil {
			return nil, err
		}
		if seen[w.Filename] {
			return nil, newWriterContractError("duplicate page filename in one batch: %s", w.Filename)
		}
		seen[w.Filename] = true
		if w.Frontmatter == nil {
			return nil, newWriterContractError("%s: frontmatter must be a mapping", w.Filename)
		}
		source, ok := w.Frontmatter["source"]
		if !ok {
			return nil, newWriterContractError("%s: page frontmatter must carry a typed source: block", w.Filename)
		}
		if err := validateSourceBlock(source); err != nil {
			return nil, err
		}
		region, err := dumpFrontmatter(w.Frontmatter)
		if err != nil {
			return nil, newWriterContractError("%s: %v", w.Filename, err)
		}
		body := w.Body
		if !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		rendered = append(rendered, renderedWrite{write: w, text: joinFileFrontmatter(region, "\n"+body)})
	}
	return rendered, nil
}

func validatePageFilename(filename string) error {
	if !IsMemoryPageName(filename) {
		return newWriterContractError("not a writable memory page filename: %q (siblings, dotfiles and paths are refused)", filename)
	}
	if _, _, _, ok := ParsePageFilename(filename); !ok {
		return newWriterContractError("page filename must follow <type>_<domain>_<slug>.md: %q", filename)
	}
	return nil
}

func logEvent(w PageWrite, stamp string) string {
	source, _ := w.Frontmatter["source"].(map[string]any)
	classes := SourceClasses(source)
	label := "(unclassified)"
	if len(classes) > 0 {
		label = strings.Join(classes, "+")
	}
	slug := w.Filename
	if _, _, s, ok := ParsePageFilename(w.Filename); ok {
		slug = s
	}
	summary := pageInfoFrom(w.Filename, w.Body).Summary
	return renderLogEvent(stamp, label, slug, summary)
}

// ---------------------------------------------------------------------------
// Six-step durable write + tri-state read
// ---------------------------------------------------------------------------

// writeStringAtomic durably writes string content through the canonical
// fsutil primitive at the store's fixed 0644 mode — a thin string adapter, not
// a second implementation (iss-32: one-canonical-primitive). fsutil handles the
// temp-file, fsync, rename, and parent-dir fsync, matching the durability this
// store previously carried in its own copy.
func writeStringAtomic(path, content string) error {
	return fsutil.WriteFileAtomic(path, []byte(content), 0o644)
}

// triStateRead: (content, present, error). Absent -> ("", false, nil); a
// present-but-unreadable file -> WriterContractError (never overwrite what we
// cannot read back); bytes -> (text, true, nil).
func triStateRead(path string) (string, bool, error) {
	// Guarded: a symlinked or oversize page on the write path is
	// present-but-unreadable, never followed.
	raw, err := fsutil.ReadGuarded(path, maxMemoryPageBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, newWriterContractError("cannot read %s (%v); refusing to overwrite a file that cannot be read back", path, err)
	}
	return string(raw), true, nil
}

func pageFiles(mem string) ([]string, error) {
	entries, err := os.ReadDir(mem)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.Type().IsRegular() && IsMemoryPageName(e.Name()) {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

// ---------------------------------------------------------------------------
// Locked sequence steps
// ---------------------------------------------------------------------------

func ensureSkeleton(mem string) ([]string, error) {
	skeletons := []struct {
		name   string
		render func() string
	}{
		{"README.md", skeletonReadme},
		{"index.md", skeletonIndex},
		{"log.md", skeletonLog},
		{"contradictions.md", skeletonContradictions},
	}
	var created []string
	for _, s := range skeletons {
		path := filepath.Join(mem, s.name)
		_, present, err := triStateRead(path)
		if err != nil {
			return nil, err
		}
		if !present {
			if err := writeStringAtomic(path, s.render()); err != nil {
				return nil, err
			}
			created = append(created, s.name)
		}
	}
	return created, nil
}

func backfillLegacy(mem string) ([]string, error) {
	files, err := pageFiles(mem)
	if err != nil {
		return nil, err
	}
	var backfilled []string
	for _, name := range files {
		path := filepath.Join(mem, name)
		text, present, err := triStateRead(path)
		if err != nil {
			return nil, err
		}
		if !present {
			continue
		}
		// Ask the parser, not the leading bytes. A page led by an HTML comment or
		// opening on an indented delimiter has frontmatter as far as every reader
		// in this package is concerned; wrapping it as the body of a synthetic
		// block would demote its declared source: provenance into prose and
		// replace it with a fabricated session_memory class.
		if textOpensFrontmatter(text) {
			region, body, err := splitFileFrontmatter(text)
			if err != nil {
				continue
			}
			fm, err := parseFrontmatter("---\n" + region + "---\n")
			if err != nil {
				continue
			}
			if _, ok := fm["source"]; ok {
				continue
			}
			// Rebuilding the file from (region, body) cannot carry a tolerated
			// preamble across, so a page that has one is left alone rather than
			// backfilled lossily.
			if !strings.HasPrefix(text, "---") {
				continue
			}
			newRegion := region + "source:\n  class: " + backfillSourceClass + "\n"
			if err := writeStringAtomic(path, joinFileFrontmatter(newRegion, body)); err != nil {
				return nil, err
			}
		} else {
			newRegion := "source:\n  class: " + backfillSourceClass + "\n"
			if err := writeStringAtomic(path, joinFileFrontmatter(newRegion, "\n"+text)); err != nil {
				return nil, err
			}
		}
		backfilled = append(backfilled, name)
	}
	return backfilled, nil
}

func pruneOrphans(repoRoot, mem string) ([]string, error) {
	registry, err := LoadRegistry(SourcesIndexPath(repoRoot))
	if err != nil {
		return nil, err
	}
	if len(registry) == 0 {
		return nil, nil
	}
	files, err := pageFiles(mem)
	if err != nil {
		return nil, err
	}
	var pruned []string
	for _, name := range files {
		path := filepath.Join(mem, name)
		text, present, err := triStateRead(path)
		if err != nil {
			return nil, err
		}
		if !present {
			continue
		}
		source := pageSourceBlock(text)
		hashes := SourceHashes(source)
		var known []string
		for _, h := range hashes {
			if _, ok := registry[h]; ok {
				known = append(known, h)
			}
		}
		if len(known) == 0 {
			continue
		}
		live := false
		for _, h := range known {
			entry, _ := registry[h].(map[string]any)
			consumers, _ := entry["consumers"].(map[string]any)
			memConsumer, _ := consumers["memory"].(map[string]any)
			pages := anyToStrings(memConsumer["pages"])
			if contains(pages, name) {
				live = true
				break
			}
		}
		if !live {
			if err := os.Remove(path); err != nil {
				return nil, err
			}
			pruned = append(pruned, name)
		}
	}
	return pruned, nil
}

func pageSourceBlock(text string) map[string]any {
	return parsePage(text).source()
}

func reconcile(mem string) ([]string, error) {
	files, err := pageFiles(mem)
	if err != nil {
		return nil, err
	}
	var infos []PageInfo
	for _, name := range files {
		text, present, err := triStateRead(filepath.Join(mem, name))
		if err != nil {
			return nil, err
		}
		if present {
			infos = append(infos, pageInfoFrom(name, text))
		}
	}
	desired := []struct {
		name    string
		content string
	}{
		{"index.md", RenderIndex(infos)},
		{"contradictions.md", RenderContradictions(infos)},
	}
	var rewritten []string
	for _, d := range desired {
		path := filepath.Join(mem, d.name)
		current, present, err := triStateRead(path)
		if err != nil {
			return nil, err
		}
		if !present || sha256Hex(current) != sha256Hex(d.content) {
			if err := writeStringAtomic(path, d.content); err != nil {
				return nil, err
			}
			rewritten = append(rewritten, d.name)
		}
	}
	return rewritten, nil
}

func appendLog(mem string, events []string) error {
	path := filepath.Join(mem, "log.md")
	current, present, err := triStateRead(path)
	if err != nil {
		return err
	}
	base := skeletonLog()
	if present {
		base = current
	}
	return writeStringAtomic(path, base+strings.Join(events, ""))
}

// ---------------------------------------------------------------------------
// small helpers
// ---------------------------------------------------------------------------

func sortedUnion(a, b []string) []string {
	set := map[string]bool{}
	for _, x := range a {
		set[x] = true
	}
	for _, x := range b {
		set[x] = true
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
