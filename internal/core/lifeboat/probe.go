package lifeboat

import (
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// SchemaVersion is the coverage-report schema version. It is stamped into every
// report and checked by the aggregate, so a future breaking change to the shape
// is detectable rather than silently misread.
const SchemaVersion = 2

// maxProbeReadBytes caps any single file the probe reads. A coverage probe reads
// prose (READMEs, ADRs, decision logs), never data blobs, so a file larger than
// this is not section evidence and is skipped. The cap also bounds memory on a
// hostile or accidental giant file.
const maxProbeReadBytes = 4 << 20 // 4 MiB

// maxGitOutputBytes caps how much stdout the probe buffers from any one git
// command. Generous enough for a large legitimate history, bounded so a hostile
// repo cannot exhaust memory through a read-only command.
const maxGitOutputBytes = 16 << 20 // 16 MiB

// ignoredListCapBytes caps the one `ls-files` listing that decides which paths
// git ignores. It is a var, not a const, so a test can force the overflow path
// on a small tree — the truncation hazard is untestable at 16 MiB, and an
// untested refusal path is the one that rots.
var ignoredListCapBytes = maxGitOutputBytes

// maxDirEntries caps how many entries a single directory read materialises, so a
// directory with millions of files cannot exhaust memory when the probe indexes
// it. It is the one canonical per-directory bound, shared by ListDir and by
// WalkFiles' per-directory read (readDirBounded) — not a second constant.
const maxDirEntries = 50000

// maxWalkFiles caps how many regular files WalkFiles returns from one walk, and
// equally how many directories it descends into, mirroring maxDirEntries: a
// whole-tree walk of a vast monorepo must terminate and stay bounded in memory.
// Both halves are load-bearing — a tree of directories holding no regular file
// never reaches a file cap. Reaching either is reported, never silent, so an
// adapter can say in its evidence that it saw only part of the tree.
const maxWalkFiles = maxDirEntries

// maxWalkDepth caps how many levels below its start WalkFiles descends. The walk
// holds a sub-root per directory (os.Root.OpenRoot), so each descent opens one
// component in O(1) and a chain costs O(depth) rather than the square of its
// depth — but an unbounded chain is still an unbounded recursion and a
// pathological cost, so the cap prunes it. Real trees are shallow — the deepest
// path in this repository is six levels — so the cap prunes only the
// pathological ones, and says so when it does.
const maxWalkDepth = 32

// walkSkipDirs are the directory names WalkFiles never descends into, matched by
// name at any depth: VCS internals, dependency trees, language caches, and
// build/distribution output across the common ecosystems. None of them holds a
// team's own material, and together they are the dominant cost of an unfiltered
// walk — and, walked as if they were source, the origin of a vendored TODO cited
// as this project's own open question. Covered: VCS (.git); Node (node_modules);
// Go/generic (vendor, generated); Python (.venv, venv, .tox, __pycache__); Rust
// and generic build output (target, build, dist); CocoaPods (Pods).
var walkSkipDirs = []string{
	".git", ".tox", ".venv", "Pods", "__pycache__", "build", "dist",
	"generated", "node_modules", "target", "vendor", "venv",
}

// Confidence qualifies a non-blank status: how sure the adapter is that the
// evidence it cites actually grounds the section. It is meaningless for a blank.
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

// Evidence is what one Source reports for its section against one repository.
// The orchestrator stamps the Tier and Section from the Source itself, so an
// adapter cannot misreport which tier or section it speaks for.
//
// Contract by status:
//   - grounded / partial: Sources must be non-empty — every claim cites a file
//     or a git ref. Confidence should be set.
//   - blank: Searched should say what was looked for and Question should name
//     the thing a human must answer. A blank is a first-class result.
type Evidence struct {
	Status     Status
	Confidence Confidence
	Sources    []string // evidence cited (repo-relative paths, git refs)
	Searched   []string // what was looked for (esp. on a blank)
	Question   string   // the human question (esp. on a blank)
}

// blank is the conventional empty result for a Source that found nothing.
func blank(searched []string, question string) Evidence {
	return Evidence{Status: StatusBlank, Searched: searched, Question: question}
}

// Source is one tiered adapter: it reads a single brief section at a single
// tier and reports what it found. Probe must be side-effect-free and must never
// write to the source repository — a probe is read-only by construction.
//
// (M3 adds a Plan method to this interface so that pack is "plan plus a write"
// over the same adapters; probe needs only Probe.)
type Source interface {
	Section() Section
	Tier() Tier
	Probe(*SourceContext) Evidence
}

// allSources is the registry: every tier's adapters, concatenated. The three
// tier constructors live in sources_git.go, sources_conventions.go, and
// sources_native.go so each can be developed independently.
func allSources() []Source {
	var s []Source
	s = append(s, gitSources()...)
	s = append(s, conventionSources()...)
	s = append(s, nativeSources()...)
	return s
}

// SourceContext is the read-only material every Source probes. It is built once
// per repository and shared across all adapters, so git history is queried and
// files are read through a single contained, cached surface. Every read is
// contained to the repository root via os.Root (no symlinked component can
// redirect a read outside the repo) and bounded in size, so probing a hostile
// or archived tree cannot escape it, hang on a FIFO, or exhaust memory.
type SourceContext struct {
	RepoRoot string

	root    *os.Root // containment scope for every file read; nil if unopenable
	isGit   bool
	rootSHA string

	// includeIgnored widens the walk to files git ignores. It is OFF by default,
	// and moving off that default is a caller's explicit choice, never inferred
	// (iss-2608241828356533).
	//
	// The default is what a user already believes: a file they told git to ignore
	// is out of scope. A packed lifeboat cites evidence by path:line, so a scan
	// that read ignored files could carry a repository's scratch, logs and local
	// notes into an artefact meant to be shared — this repository's own gitignored
	// local tier is not in walkSkipDirs, so it was in scope.
	//
	// The opt-in exists because disembark is offered over DEAD and ARCHIVED
	// repositories, where uncommitted residue is often the most valuable thing
	// left. Refusing to read it would lose the case the verb exists for. So the
	// wide scan stays reachable, and asking for it is the disclosure.
	includeIgnored bool
	ignoredOnce    sync.Once
	notIgnored     map[string]struct{} // nil when unknown or not applicable

	// listCap bounds a single directory listing (ListDir / listDirNoted). It is a
	// field defaulting to maxDirEntries — not the const directly — so a test can
	// force the cap-truncation path on a small tree, the way ignoredListCapBytes is
	// a var for the same reason (iss-2608270908348796).
	listCap int

	mu       sync.Mutex
	gitCache map[string]gitResult
}

type gitResult struct {
	out string
	err error
}

// newSourceContext opens repoRoot for contained reads and records whether it is
// a git repository. It never writes.
func newSourceContext(repoRoot string) (*SourceContext, error) {
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}
	c := &SourceContext{RepoRoot: abs, gitCache: map[string]gitResult{}, listCap: maxDirEntries}
	// os.OpenRoot refuses any later path component that escapes the root,
	// symlinked intermediates included — the same containment the privacy audit
	// adopted. A root that cannot be opened leaves reads returning "absent".
	if root, err := os.OpenRoot(abs); err == nil {
		c.root = root
	}
	if gitutil.InRepo(abs) {
		c.isGit = true
		c.rootSHA = gitutil.RootCommit(abs)
	}
	return c, nil
}

// Close releases the containment handle.
func (c *SourceContext) Close() {
	if c.root != nil {
		_ = c.root.Close()
	}
}

// IsGit reports whether the source is a git working tree.
func (c *SourceContext) IsGit() bool { return c.isGit }

// RootSHA is the canonical root-commit SHA, or "" outside a git repo.
func (c *SourceContext) RootSHA() string { return c.rootSHA }

// Git runs a read-only git subcommand under the repo, isolated from the
// developer's git config, and caches the result so repeated identical queries
// across adapters cost one exec. Outside a git repo it returns an error.
func (c *SourceContext) Git(args ...string) (string, error) {
	key := strings.Join(args, "\x00")
	c.mu.Lock()
	if r, ok := c.gitCache[key]; ok {
		c.mu.Unlock()
		return r.out, r.err
	}
	c.mu.Unlock()

	// Cap git stdout: an untrusted repo can make a read-only command emit
	// arbitrarily much, and the probe must not let that grow memory without
	// bound.
	out, err := gitutil.RunLimited(c.RepoRoot, maxGitOutputBytes, args...)

	c.mu.Lock()
	c.gitCache[key] = gitResult{out: out, err: err}
	c.mu.Unlock()
	return out, err
}

// GitLines runs a git subcommand and splits stdout into non-empty lines.
func (c *SourceContext) GitLines(args ...string) []string {
	out, err := c.Git(args...)
	if err != nil || out == "" {
		return nil
	}
	return splitLines(out)
}

// CommitCount is the number of commits reachable from HEAD, or 0 outside a repo.
func (c *SourceContext) CommitCount() int {
	out, err := c.Git("rev-list", "--count", "HEAD")
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(out))
	return n
}

// ReadFile reads a repo-relative file through the containment root, bounded and
// non-blocking. It returns (data, true) only for a regular file within the cap;
// a missing file, a directory, a FIFO/device, an escaping path, or an oversized
// file returns (nil, false) — never an error and never a blocked read.
func (c *SourceContext) ReadFile(rel string) ([]byte, bool) {
	if c.root == nil {
		return nil, false
	}
	f, err := c.root.OpenFile(filepath.FromSlash(rel), os.O_RDONLY|nonBlock, 0)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, false
	}
	if info.Size() > maxProbeReadBytes {
		return nil, false
	}
	// Read cap+1, not exactly the cap: a file that grew past the cap between the
	// fstat and the read (a size TOCTOU over an untrusted target repo) is refused
	// rather than silently truncated to a prefix the convention scanners would
	// then judge as the whole file — matching fsutil.ReadGuarded's discipline.
	data, err := io.ReadAll(io.LimitReader(f, maxProbeReadBytes+1))
	if err != nil || int64(len(data)) > maxProbeReadBytes {
		return nil, false
	}
	return data, true
}

// Exists reports whether a repo-relative path exists (of any kind) within the
// containment root.
func (c *SourceContext) Exists(rel string) bool {
	if c.root == nil {
		return false
	}
	_, err := c.root.Stat(filepath.FromSlash(rel))
	return err == nil
}

// IsDir reports whether a repo-relative path is a directory within the root.
func (c *SourceContext) IsDir(rel string) bool {
	if c.root == nil {
		return false
	}
	info, err := c.root.Stat(filepath.FromSlash(rel))
	return err == nil && info.IsDir()
}

// FindFirst returns the first candidate that exists (case-sensitive, as given),
// or "" if none do. Adapters pass the conventional spellings they care about
// (e.g. "README.md", "README", "readme.md").
func (c *SourceContext) FindFirst(candidates ...string) string {
	for _, cand := range candidates {
		if c.Exists(cand) {
			return cand
		}
	}
	return ""
}

// ListDir returns the names (not paths) of entries directly under a repo-relative
// directory, sorted. It never recurses and never escapes the root. The open is
// non-blocking (matching ReadFile): a probed tree is untrusted, so a FIFO planted
// where a directory is expected must return promptly instead of blocking open().
func (c *SourceContext) ListDir(rel string) []string {
	names, _ := c.listDirNoted(rel)
	return names
}

// listDirNoted is ListDir plus a truncation flag: it reports whether the directory
// held more entries than the per-directory cap, so a caller scanning a record home
// can SAY its scan saw only a prefix rather than dropping the tail in silence. It
// reads one past the cap so "more remain" is detectable in a single bounded call,
// exactly as readDirBounded does for the recursive walk.
func (c *SourceContext) listDirNoted(rel string) (names []string, truncated bool) {
	if c.root == nil {
		return nil, false
	}
	f, err := c.root.OpenFile(filepath.FromSlash(rel), os.O_RDONLY|nonBlock, 0)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	// Bounded: a directory with millions of entries cannot balloon memory here.
	entries, err := f.ReadDir(c.listCap + 1)
	if err != nil && len(entries) == 0 {
		return nil, false
	}
	if len(entries) > c.listCap {
		truncated = true
		entries = entries[:c.listCap]
	}
	names = make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, truncated
}

// WalkFiles returns the repo-relative POSIX paths of every regular file beneath
// a repo-relative directory, sorted, and reports whether the walk stopped at any
// of its bounds — the file cap, the directory cap, the depth cap, or a single
// directory exceeding the per-directory read bound. It is the recursive
// counterpart of ListDir, for adapters whose evidence is the shape of the tree
// rather than a known filename. Content is still read through ReadFile, so the
// walk adds no second read path.
//
// It is contained by the same os.Root as every other read, descending through a
// sub-root per directory so each child opens in O(1) while the containment
// property holds; it skips the walkSkipDirs trees, and skips non-regular files
// (FIFOs, devices, sockets) so no path it yields can block on open. Symlinks —
// file or directory — are
// SKIPPED and the walk continues. This deliberately differs from embark's
// walkLifeboatFiles, where a symlink is a trust violation in a packed lifeboat
// and therefore fatal: a probe reads an arbitrary foreign tree in which a
// symlink is ordinary, so refusing to follow one is enough, and erroring on one
// would blank a whole section over a repository's normal furniture.
func (c *SourceContext) WalkFiles(rel string) (paths []string, truncated bool) {
	return c.walkFilesLimited(rel, maxWalkFiles)
}

// walkFilesLimited is WalkFiles with the whole-walk file-and-directory cap
// injected, so the truncation branches are exercisable by a test at an
// affordable scale. The shipped cap stays a const: adapters run concurrently,
// and a mutable package-level cap would be shared state between them.
func (c *SourceContext) walkFilesLimited(rel string, limit int) (paths []string, truncated bool) {
	return c.walkFilesBounded(rel, limit, maxDirEntries)
}

// walkFilesBounded is WalkFiles with both bounds injected — the whole-walk cap
// (limit: regular files and directories) and the per-directory read bound
// (perDir) — so each is exercisable by a test at an affordable scale.
//
// It holds a sub-root per directory: each child directory is opened with
// os.Root.OpenRoot from its parent's already-open handle, so a descent opens one
// component relative to that directory (O(1)) rather than re-resolving the whole
// path from the containment root on every open (O(depth)). A chain of
// directories therefore costs O(entries), not O(entries × depth). Each directory
// is read with a bounded ReadDir(perDir) — the same guard ListDir uses — so a
// single directory of millions of entries cannot balloon memory before the file
// cap applies.
//
// The os.Root containment guarantee the FS() walk had survives unchanged:
// OpenRoot refuses any component that escapes the root, and a symlinked
// directory is detected from its ReadDir type and skipped before it is ever
// opened, so no symlink is ever followed out of the tree.
func (c *SourceContext) walkFilesBounded(rel string, limit, perDir int) (paths []string, truncated bool) {
	if c.root == nil {
		return nil, false
	}
	start := path.Clean(filepath.ToSlash(rel))
	if !fs.ValidPath(start) {
		return nil, false
	}
	startRoot := c.root
	if start != "." {
		r, err := c.root.OpenRoot(filepath.FromSlash(start))
		if err != nil {
			return nil, false
		}
		defer r.Close()
		startRoot = r
	}

	dirs := 1 // the start directory itself counts against the directory cap
	var walk func(dirRoot *os.Root, prefix string, depth int) (stop bool)
	walk = func(dirRoot *os.Root, prefix string, depth int) bool {
		entries, more := readDirBounded(dirRoot, perDir)
		if more {
			// The directory held more entries than the per-directory bound: only
			// the bound was materialised, exactly as ListDir bounds one listing.
			truncated = true
		}
		for _, e := range entries {
			name := e.Name()
			child := name
			if prefix != "." {
				child = prefix + "/" + name
			}
			if e.Type()&fs.ModeSymlink != 0 {
				// Never followed — skipped alone, and the walk continues.
				continue
			}
			if e.IsDir() {
				if isSkipDir(name) {
					continue
				}
				// Directories are capped alongside files: a tree of directories
				// holding nothing regular yields no path, so a file cap alone never
				// fires and the walk would run to exhaustion over a foreign tree.
				if dirs >= limit {
					truncated = true
					return true
				}
				dirs++
				if depth+1 >= maxWalkDepth {
					// Prune the chain, not the tree: the directory is counted but
					// not descended into, and the truncation is reported either way.
					truncated = true
					continue
				}
				sub, err := dirRoot.OpenRoot(name)
				if err != nil {
					// Unreadable (or vanished) in a foreign tree: skip it and report
					// only what could be read.
					continue
				}
				stop := walk(sub, child, depth+1)
				sub.Close()
				if stop {
					return true
				}
				continue
			}
			if !e.Type().IsRegular() {
				continue
			}
			if c.pathIsIgnored(child) {
				continue
			}
			if len(paths) >= limit {
				truncated = true
				return true
			}
			paths = append(paths, child)
		}
		return false
	}
	walk(startRoot, start, 0)
	sort.Strings(paths)
	return paths, truncated
}

// pathIsIgnored reports whether git ignores rel. It answers false whenever it
// cannot know — a non-git tree, or a git call that fails — because the walk must
// not silently narrow on a repository it could not interrogate: losing evidence
// quietly is the failure this whole adapter family exists to avoid.
//
// The not-ignored set is computed ONCE per context, from a single
// `ls-files --cached --others --exclude-standard`. That is git's own answer to
// "everything tracked, plus everything untracked that is not ignored", so the
// complement is exactly the ignored set — no .gitignore parsing of our own, and
// no per-file `check-ignore`, which would be one process per file over a foreign
// tree.
func (c *SourceContext) pathIsIgnored(rel string) bool {
	if c.includeIgnored || !c.isGit {
		return false
	}
	c.ignoredOnce.Do(func() {
		// RunCapped, not the cached Git/GitLines path. Those use RunLimited, which
		// TRUNCATES silently when git's output exceeds the cap — and a truncated
		// listing here is not a smaller answer, it is an inverted one: every file
		// past the cut is absent from the set and therefore reads as ignored, so a
		// large repository would silently drop the tail of its own tree from the
		// scan. RunCapped exists for exactly this shape of caller, and its error
		// leaves the set nil, which means "unknown" and narrows nothing.
		out, err := gitutil.RunCapped(c.RepoRoot, ignoredListCapBytes,
			"ls-files", "--cached", "--others", "--exclude-standard", "-z")
		if err != nil {
			return // unknown: git could not answer, so narrow nothing
		}
		// An empty listing is a definite answer, not an unknown: git is saying
		// nothing is tracked and nothing untracked is un-ignored, so every path in
		// this tree is ignored. That must leave an EMPTY (non-nil) set, which
		// narrows every path — not the nil set below, which widens. A repository
		// whose files are all ignored is exactly the leak the default scan refuses,
		// so it must NOT fall through to the "unknown" widening above.
		set := make(map[string]struct{}, 256)
		for _, f := range strings.Split(out, "\x00") {
			if f != "" {
				set[f] = struct{}{}
			}
		}
		c.notIgnored = set
	})
	if c.notIgnored == nil {
		return false
	}
	_, ok := c.notIgnored[rel]
	return !ok
}

// IgnoredAreIncluded reports whether this walk reads files git ignores. Adapters
// use it to SAY which scan ran, so a reader of a packed lifeboat can tell whether
// an absent citation means "nothing there" or "not looked at".
func (c *SourceContext) IgnoredAreIncluded() bool { return c.includeIgnored }

// readDirBounded reads at most bound entries from the directory dirRoot points
// at, sorted by name for a deterministic walk, and reports whether the directory
// held more than bound. It materialises at most bound+1 entries, so a directory
// of millions cannot balloon memory here — the shared per-directory guard ListDir
// applies with ReadDir(maxDirEntries).
func readDirBounded(dirRoot *os.Root, bound int) (entries []fs.DirEntry, more bool) {
	f, err := dirRoot.Open(".")
	if err != nil {
		return nil, false
	}
	defer f.Close()
	// Read one past the bound so "more remain" is detectable in a single call
	// without ever materialising the whole listing.
	entries, _ = f.ReadDir(bound + 1)
	if len(entries) > bound {
		more = true
		entries = entries[:bound]
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, more
}

// isSkipDir reports whether a directory of this name is one WalkFiles never
// descends into — matched by name at any depth.
func isSkipDir(name string) bool {
	for _, s := range walkSkipDirs {
		if name == s {
			return true
		}
	}
	return false
}

// ProbeOption adjusts one probe run. Options are variadic so the default stays
// the safe one and every widening is written at the call site.
type ProbeOption func(*SourceContext)

// IncludeIgnored widens the walk to files git ignores. Off by default: a file a
// user told git to ignore is out of scope until they say otherwise, and a packed
// lifeboat cites evidence by path:line. Pass it for the salvage case — a dead or
// archived repository whose uncommitted residue is the point.
func IncludeIgnored() ProbeOption { return func(c *SourceContext) { c.includeIgnored = true } }

// Probe runs every registered adapter over one repository and reduces the
// results to a Coverage report. Adapters run concurrently; the reduction is
// deterministic. It never writes to the source repository.
//
// By default it honours .gitignore. See IncludeIgnored.
func Probe(repoRoot string, opts ...ProbeOption) (Coverage, error) {
	ctx, err := newSourceContext(repoRoot)
	if err != nil {
		return Coverage{}, err
	}
	for _, o := range opts {
		o(ctx)
	}
	defer ctx.Close()

	present := tiersPresent(ctx)
	presentSet := map[Tier]bool{}
	for _, t := range present {
		presentSet[t] = true
	}

	// Run adapters concurrently. An adapter whose tier is not present in this
	// repo is skipped rather than run-and-blanked, so its tier-specific
	// "searched"/"question" never colours a repo that tier is absent from.
	type result struct {
		section Section
		tier    Tier
		ev      Evidence
	}
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []result
	)
	for _, s := range allSources() {
		if !presentSet[s.Tier()] {
			continue
		}
		wg.Add(1)
		go func(s Source) {
			defer wg.Done()
			ev := s.Probe(ctx)
			mu.Lock()
			results = append(results, result{section: s.Section(), tier: s.Tier(), ev: ev})
			mu.Unlock()
		}(s)
	}
	wg.Wait()

	// Index the best evidence per section: highest status wins; on a tie the
	// richer tier wins (a grounded-at-conventions beats grounded-at-git).
	best := map[Section]result{}
	for _, r := range results {
		if r.ev.Status == StatusBlank {
			continue // a blank never displaces a real result or another blank
		}
		cur, ok := best[r.section]
		if !ok || beats(r, cur) {
			best[r.section] = r
		}
	}
	// For a blank fallback, keep the richest tier's blank so its searched/
	// question is the most informative available.
	blankFallback := map[Section]result{}
	for _, r := range results {
		if r.ev.Status != StatusBlank {
			continue
		}
		cur, ok := blankFallback[r.section]
		if !ok || tierRank(r.tier) > tierRank(cur.tier) {
			blankFallback[r.section] = r
		}
	}

	// Assemble one row per brief section, in the mapping's canonical order, so
	// the report is stable and every section always appears.
	sections := make([]SectionCoverage, 0, len(Table))
	var sum Summary
	for _, m := range Table {
		sc := SectionCoverage{Name: m.Section, Status: StatusBlank}
		if r, ok := best[m.Section]; ok {
			sc = SectionCoverage{
				Name:       m.Section,
				Status:     r.ev.Status,
				Confidence: r.ev.Confidence,
				Tier:       r.tier,
				Evidence:   dedupeSorted(r.ev.Sources),
				Searched:   dedupeSorted(r.ev.Searched),
				Question:   r.ev.Question,
			}
		} else if r, ok := blankFallback[m.Section]; ok {
			sc.Searched = dedupeSorted(r.ev.Searched)
			sc.Question = r.ev.Question
		} else {
			// No adapter spoke for this section at all: derive an honest blank
			// from the hypothesis row so the report still names what a lifeboat
			// would look for and the question a human must answer.
			sc.Searched = splitReads(m.Reads)
			sc.Question = "Nothing probed grounds " + string(m.Section) + "; a human must supply it."
		}
		// Every section carries its kind (adr-36); a blank starts life open, so
		// the round-trip can track whether a human later answers or defers it.
		sc.Kind = m.Section.Kind()
		switch sc.Status {
		case StatusGrounded:
			sum.Grounded++
		case StatusPartial:
			sum.Partial++
		default:
			sc.Resolution = ResolutionOpen
			sum.Blank++
		}
		sections = append(sections, sc)
	}

	return Coverage{
		SchemaVersion: SchemaVersion,
		Repo: RepoInfo{
			Name:    filepath.Base(ctx.RepoRoot),
			RootSHA: ctx.RootSHA(),
			Commits: ctx.CommitCount(),
		},
		TiersPresent:    present,
		Sections:        sections,
		Summary:         sum,
		IncludedIgnored: ctx.IgnoredAreIncluded(),
	}, nil
}

// beats reports whether candidate a is a better result than incumbent b:
// higher status, or equal status at a richer tier.
func beats(a, b struct {
	section Section
	tier    Tier
	ev      Evidence
}) bool {
	if a.ev.Status.rank() != b.ev.Status.rank() {
		return a.ev.Status.rank() > b.ev.Status.rank()
	}
	return tierRank(a.tier) > tierRank(b.tier)
}

// tiersPresent reports which tiers a repository actually has, poorest first.
func tiersPresent(c *SourceContext) []Tier {
	var present []Tier
	if c.IsGit() {
		present = append(present, TierGit)
	}
	if hasConventions(c) {
		present = append(present, TierConventions)
	}
	if hasNative(c) {
		present = append(present, TierNative)
	}
	return present
}

// hasConventions is true when any file or directory the Tier-1 convention
// adapters actually read exists — the union of their evidence sets, not just the
// headline docs. The tier gate skips every adapter of an absent tier
// (probe.go:314), so narrowing this below what the adapters consult produces
// false blanks: a repo carrying only build manifests, CI workflows, ADR dirs,
// a glossary, or an issues file would have its whole Tier-1 set skipped even
// though those adapters would find grounding evidence. Composed from the
// adapters' own name lists in sources_conventions.go so the two cannot drift.
func hasConventions(c *SourceContext) bool {
	candidates := []string{
		"docs", "LICENSE", "LICENSE.md", "CONTRIBUTING.md", "CONTRIBUTING",
		"ISSUES.md", "ISSUES",
		// Directory evidence the adapters treat as grounding.
		".github/workflows", "issues", ".github/ISSUE_TEMPLATE",
	}
	candidates = append(candidates, convReadmeNames...)      // convReadme
	candidates = append(candidates, convChangelogNames...)   // convWhatWorkedSource
	candidates = append(candidates, convGlossaryDocNames...) // convGlossarySource
	candidates = append(candidates, convPlatformFiles...)    // convPlatformSource (Dockerfile, Makefile, go.mod, package.json)
	candidates = append(candidates, convADRDirs...)          // convADRsSource
	candidates = append(candidates, convNamingDocNames...)   // convNamingSource
	// convInternalsSource: an architecture document, an architecture tree, or the
	// package layout on its own is grounding evidence for internals.
	candidates = append(candidates, convArchitectureDocNames...)
	candidates = append(candidates, convArchitectureDirs...)
	candidates = append(candidates, convLayoutRoots...)
	for _, ml := range convManifestLocks { // convDependenciesSource
		candidates = append(candidates, ml.manifest)
	}
	return c.FindFirst(candidates...) != ""
}

// hasNative is true when the repo carries an abcd record.
func hasNative(c *SourceContext) bool {
	return c.IsDir(".abcd/development") || c.IsDir(".abcd/work")
}

// tierRank orders tiers by richness for tie-breaking.
func tierRank(t Tier) int {
	switch t {
	case TierGit:
		return 0
	case TierConventions:
		return 1
	case TierNative:
		return 2
	}
	return -1
}

// splitLines returns the non-empty, whitespace-trimmed lines of s.
func splitLines(s string) []string {
	raw := strings.Split(s, "\n")
	out := make([]string, 0, len(raw))
	for _, l := range raw {
		if t := strings.TrimSpace(l); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// splitReads turns a mapping row's free-text "Reads" into discrete searched
// entries, so a blank derived from the hypothesis still lists concrete targets.
func splitReads(reads string) []string {
	parts := strings.FieldsFunc(reads, func(r rune) bool { return r == ',' || r == ';' })
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// dedupeSorted returns the unique, sorted, non-empty members of in, or nil.
func dedupeSorted(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}
