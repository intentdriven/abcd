package site

// `abcd site build` — the whole render, from repository to output tree.
//
// The order is: read the committed inputs (manifest, allowlist, baseline,
// package metadata), read the record ONCE through the lint engine's own scan,
// walk git history ONCE, then compose. Nothing here reaches the network, and
// nothing writes outside the output directory: every write goes through an
// os.Root opened at the destination, so a path arriving from a manifest cannot
// escape it even by construction.
//
// The build is transport-agnostic. It returns a Result describing what it wrote;
// the front door prints it.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// DefaultOutDir is where the build writes when the caller names no directory.
const DefaultOutDir = "site"

// The committed sources the build copies into the output tree verbatim, and the
// names they take there. Cloudflare reads the destinations; the sources are
// reviewable files with comments, which the destinations keep.
var copiedSources = []struct{ src, dst string }{
	{"site-src/redirects", "_redirects"},
	{"site-src/headers", "_headers"},
	{"site-src/site.css", "site.css"},
	{"site-src/site.js", "site.js"},
	// The chart's runtime, loaded by `/record/graph/` alone. It is a separate
	// file so the landing page never pays for it.
	{"site-src/record.js", "record.js"},
}

// outDirState is what the build found where it is about to write.
type outDirState int

const (
	outDirAbsent  outDirState = iota // nothing there yet
	outDirEmpty                      // there, holding nothing
	outDirOurs                       // there, holding a tree this build wrote
	outDirForeign                    // there, holding something else entirely
)

// inspectOutDir reports what is in the output directory, without changing it.
// outDir has passed resolveOutDir; rootCommit is the repository's identity the
// marker must name for the directory to read as ours. A marker that is present
// but says something else — another repository's, a symlink, an oversize or
// non-regular file — makes the directory foreign, not an error: the build
// cannot tell it apart from one it did not write, which is the refusal. A
// repository with NO identity (no root commit) can own nothing: its non-empty
// output is foreign before the marker is read, because a marker it could
// accept is one every other identity-less tree writes too.
func inspectOutDir(outDir, rootCommit string) (outDirState, error) {
	entries, err := os.ReadDir(outDir)
	switch {
	case os.IsNotExist(err):
		return outDirAbsent, nil
	case err != nil:
		return 0, err
	case len(entries) == 0:
		return outDirEmpty, nil
	case rootCommit == "":
		return outDirForeign, nil
	}
	data, err := fsutil.ReadGuarded(filepath.Join(outDir, siteMarkerName), maxSiteMarkerBytes)
	switch {
	case err == nil:
		if markerIdentifies(data, rootCommit) {
			return outDirOurs, nil
		}
		return outDirForeign, nil
	case os.IsNotExist(err), errors.Is(err, fsutil.ErrNotRegular), errors.Is(err, fsutil.ErrTooBig), errors.Is(err, syscall.ELOOP):
		return outDirForeign, nil
	default:
		return 0, err
	}
}

// errForeignOutDir is the refusal. It names the directory, says why it stopped,
// and says what would make it proceed — because the reader is looking at a build
// that did nothing, and the useful question is which case they are in.
//
// There are three, and the third is the one worth spelling out: a tree left by a
// build from before the marker existed is genuinely ours, and neither "use an
// empty directory" nor "use one an earlier build wrote" tells that reader
// anything they can act on. Emptying it themselves does.
func errForeignOutDir(outDir, rootCommit string) error {
	if rootCommit == "" {
		return fmt.Errorf("site: %s is not empty, and this repository has no root commit for a %s to name, so this build cannot tell the directory apart from one it did not write; refusing to remove it — point --out at an empty directory, or empty this one yourself if it is an old build's output",
			outDir, siteMarkerName)
	}
	return fmt.Errorf("site: %s is not empty and holds no %s naming this repository's build, so this build cannot tell it apart from a directory it did not write; refusing to remove it — point --out at an empty directory, or empty this one yourself if it is an old build's output",
		outDir, siteMarkerName)
}

// purgeOutDir clears a directory a previous build wrote, keeping the marker.
//
// The ENTRIES go, not the directory: a caller may be sitting in it, it may be a
// mount point, and re-creating it would change what it is. The directory has
// passed resolveOutDir and refuseTrackedOutDir — this is the mechanism, not the
// decision — and every removal goes through the os.Root openOutDir returned, so
// the path cannot be re-pointed between the decision and the removal.
//
// The marker STAYS, and that is the whole subtlety here. `os.ReadDir` returns
// names in order and `.abcd-site-build` sorts ahead of everything, so removing
// it like any other entry would remove it first — and a purge interrupted after
// that point (killed process, full disk, one unremovable file) leaves a
// non-empty directory carrying no marker, which every later build then refuses.
// The tool would jam on its own wreckage and need a human with `rm` to get it
// going again. Keeping the marker costs nothing: the build rewrites it with
// identical bytes, so it is present at every instant and correct at the end.
// (For a tree with no root commit that jam IS the design: it owns nothing, so
// its non-empty output is emptied by hand every time; the marker it keeps here
// is documentation, never a claim it could accept.)
//
// A symlink among the entries is removed as a LINK; `Root.RemoveAll` does not
// follow it, so nothing outside this directory is reachable from here.
func purgeOutDir(root *os.Root) error {
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() == siteMarkerName {
			continue
		}
		if err := root.RemoveAll(e.Name()); err != nil {
			return err
		}
	}
	return nil
}

// The installer: one committed template, served at one route. It is NOT in
// copiedSources because it is the one copied file the build adds a byte to —
// the stamp comment — and a reader is owed the difference being deliberate.
const (
	installTemplateRelPath = "site-src/install.sh.tmpl"
	installScriptName      = "install.sh"
)

// renderInstallScript renders the committed installer to the bytes served at
// /install.sh: the template, plus one comment naming the build that published
// it.
//
// The template is complete shell and the render is a copy — nothing is
// substituted, because a reader who verifies the script against the file in the
// repository must find them the same, and a template with holes in it makes that
// comparison a judgement call. The stamp is a COMMENT, and it goes after the
// shebang, which must stay on the first line for the kernel and for `sh` to read
// it.
//
// A build with no version and no commit adds nothing. "built from " naming
// nothing is worse than silence: it looks like a stamp that failed rather than a
// build that had nothing to stamp.
func renderInstallScript(tmpl []byte, stamp BuildStamp) []byte {
	var parts []string
	if v := stampWord(strings.TrimPrefix(stamp.Version, "v")); v != "" {
		parts = append(parts, "v"+v)
	}
	if c := stampWord(stamp.Commit); c != "" {
		parts = append(parts, c)
	}
	if len(parts) == 0 {
		return tmpl
	}
	comment := "# built from " + strings.Join(parts, " ") + "\n"

	body := string(tmpl)
	if !strings.HasPrefix(body, "#!") {
		return []byte(comment + body)
	}
	// After the shebang LINE, so a script that is one line long still gets a
	// stamp on a line of its own.
	cut := strings.Index(body, "\n")
	if cut < 0 {
		return []byte(body + "\n" + comment)
	}
	return []byte(body[:cut+1] + comment + body[cut+1:])
}

// stampWord reduces one stamp field to what may appear inside a shell comment.
//
// The fields are repository facts — a version parsed out of the changelog, a
// git object name — so this is not defence against an attacker. It is defence
// against the OUTPUT FORMAT: a comment is terminated by a newline, and this file
// is a script people pipe into a shell, so a stamp carrying one would not
// corrupt a document, it would add a COMMAND. Nothing downstream would notice,
// because the result is still a valid script.
//
// The rule is a whitelist rather than an escape, because there is no escaping
// inside a `#` comment: the only safe answer to a character that could end the
// line is to not write it. What survives is what a version or an object name is
// made of; a field left empty by it is treated as a field that said nothing.
func stampWord(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '-', r == '_', r == '+':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// pluginManifestRelPath carries the package's own metadata: its name, its forge
// URL, its licence and its author. The site reads them rather than restating
// them, so there is one place a rename has to happen.
const pluginManifestRelPath = ".claude-plugin/plugin.json"

const maxPluginManifestBytes = 256 * 1024

// RepoMeta is the package metadata the site links and credits from.
type RepoMeta struct {
	Name       string `json:"name"`
	Repository string `json:"repository"`
	License    string `json:"license"`
	Author     struct {
		Name string `json:"name"`
	} `json:"author"`
	// AuthorName flattens Author.Name for the footer.
	AuthorName string `json:"-"`
}

// LoadRepoMeta reads the package manifest. An absent one is a state: the pages
// that would carry a forge link render without it.
func LoadRepoMeta(repoRoot string) (RepoMeta, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return RepoMeta{}, err
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, pluginManifestRelPath, maxPluginManifestBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return RepoMeta{}, nil
		}
		return RepoMeta{}, err
	}
	var m RepoMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return RepoMeta{}, fmt.Errorf("site: %s: %w", pluginManifestRelPath, err)
	}
	// The repository address becomes an href on every emitted page (the forge
	// link and the release-download links derive from it). It is repository text
	// like any other — the same reason LoadBibliography screens its URLs — so an
	// executable scheme is refused here rather than escaped downstream: escaping
	// an attribute is no defence against a well-formed javascript: address.
	if m.Repository != "" {
		if scheme, bad := executableScheme(m.Repository); bad {
			return RepoMeta{}, fmt.Errorf("site: %s: repository addresses the %q scheme, which runs code in the reader's browser; the site links to it from every page", pluginManifestRelPath, scheme)
		}
	}
	m.AuthorName = m.Author.Name
	return m, nil
}

// Request is one build.
type Request struct {
	// RepoRoot is the repository to render.
	RepoRoot string
	// OutDir is where to write, absolute or relative to the caller's cwd.
	OutDir string
	// Stamp is the build metadata the footer and record.json carry. Every field
	// is injected so a test can pin the output byte for byte; empty fields are
	// filled from the repository (the version from the changelog, the commit
	// from git HEAD).
	Stamp BuildStamp
}

// Result describes what a build wrote.
type Result struct {
	OutDir string `json:"out_dir"`
	// Files are the output-relative paths written, sorted.
	Files []string `json:"files"`
	// Bytes is the total size written.
	Bytes int64 `json:"bytes"`
	// Pages is how many HTML pages the explorer rendered, beyond the landing
	// page: one per route and one per record.
	Pages int `json:"pages"`
	// Records, Links and Mentions summarise the exported graph.
	Records  int `json:"records"`
	Links    int `json:"links"`
	Mentions int `json:"mentions"`
	// Unresolved is the number of typed references the record cannot account
	// for; Baseline the committed ratchet's size.
	Unresolved int `json:"unresolved"`
	Baseline   int `json:"baseline"`
	// Overlaps is the packing's sanity count over BOTH arrangements. It is zero
	// or the chart is wrong, and it is reported rather than assumed.
	Overlaps int `json:"overlaps"`
	// Version and Commit are what the footer says the site was built from.
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// ErrNoManifest is returned when the repository declares no composition.
var ErrNoManifest = errors.New("site: this repo declares no site composition (" + ManifestRelPath + " is absent)")

// Build renders the site into req.OutDir.
func Build(req Request) (Result, error) {
	repoRoot := req.RepoRoot
	outDir, err := resolveOutDir(repoRoot, req.OutDir)
	if err != nil {
		return Result{}, err
	}

	// The two version instructions are opposites — "stamp exactly this" and
	// "stamp no version at all" — so a build given both would have to silently
	// discard one and publish its output under the other's name.
	if req.Stamp.Preview && req.Stamp.Version != "" {
		return Result{}, errors.New("site: a preview build carries no version, so --preview and --version cannot be given together")
	}

	// Refuse EARLY. The render below is minutes of work, and a build that waits
	// until the writes to discover it may not touch the directory has spent them
	// for nothing. The PURGE waits, though: it happens at the first write, so a
	// failure anywhere in between leaves the previous output standing rather than
	// clearing a good tree in exchange for one that never arrived.
	rootCommit := gitutil.RootCommit(repoRoot)
	outState, err := inspectOutDir(outDir, rootCommit)
	if err != nil {
		return Result{}, err
	}
	if outState == outDirForeign {
		return Result{}, errForeignOutDir(outDir, rootCommit)
	}
	if outState == outDirOurs {
		if err := refuseTrackedOutDir(outDir); err != nil {
			return Result{}, err
		}
	}

	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, ErrNoManifest
		}
		return Result{}, err
	}
	ui, err := LoadUI(repoRoot, manifest.UIStrings)
	if err != nil {
		return Result{}, err
	}
	repo, err := LoadRepoMeta(repoRoot)
	if err != nil {
		return Result{}, err
	}

	lintRoot, err := os.OpenRoot(repoRoot)
	if err != nil {
		return Result{}, err
	}
	lintCfg, err := lint.LoadConfigInRoot(lintRoot, ".abcd/record-lint.json")
	lintRoot.Close()
	if err != nil && !os.IsNotExist(err) {
		return Result{}, err
	}
	graph, err := lint.LoadRecordGraph(lintCfg, repoRoot)
	if err != nil {
		return Result{}, err
	}
	// The principle store carries no frontmatter, so the lint scan cannot see
	// it. It joins the graph here, and a repository that keeps none simply has
	// none — the pages that would list them are omitted.
	principles, err := LoadPrinciples(repoRoot, PrinciplesDir(lintCfg))
	if err != nil {
		return Result{}, err
	}
	hist, err := LoadHistory(repoRoot)
	if err != nil {
		return Result{}, err
	}

	stamp := req.Stamp
	if stamp.Commit == "" {
		stamp.Commit = HeadCommit(repoRoot)
	}
	releases, _, err := changelog.DatedReleases(repoRoot)
	if err != nil {
		return Result{}, err
	}
	version, releaseDate := "", ""
	if len(releases) > 0 {
		version, releaseDate = releases[0].Version, releases[0].Date
	}
	// A preview takes no version — not the pinned one, and not the changelog
	// fallback. The fallback is the whole defect: a build of main is ahead of the
	// newest release, so stamping it with that release's number publishes a
	// provenance a reader could go and check against a different tree.
	if !stamp.Preview && stamp.Version == "" {
		stamp.Version = version
	}
	if stamp.GeneratedAt == "" && len(releases) > 0 {
		// With no injected stamp the build still refuses to read the clock: the
		// newest release's own date is a fact about the tree, and a build of one
		// tree must produce one set of bytes.
		stamp.GeneratedAt = releases[0].Date
	}

	// The READ containment root, opened once at the repository for the whole
	// build. Every repo-relative source read below resolves through it, so a
	// committed directory symlink planted as an ANCESTOR of a source path cannot
	// walk the read outside the repository — the counterpart of the destination
	// os.Root the writes already go through (gh #487).
	srcRoot, err := os.OpenRoot(repoRoot)
	if err != nil {
		return Result{}, err
	}
	defer srcRoot.Close()

	c := &composer{
		repoRoot: repoRoot, root: srcRoot, manifest: manifest, ui: ui, repo: repo,
		assets: newAssetPipe(repoRoot, srcRoot), graph: graph, history: hist, stamp: stamp,
		beta: isPreOne(version), version: version, releaseDate: releaseDate,
	}
	html, err := c.ComposeLanding()
	if err != nil {
		return Result{}, err
	}

	export, err := BuildRecordExport(repoRoot, manifest.Checks.UnresolvedReferenceBaseline, graph, principles, hist, stamp, manifest.Record)
	if err != nil {
		return Result{}, err
	}
	recordJSON, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return Result{}, err
	}
	recordJSON = append(recordJSON, '\n')

	// The record explorer. Its optional inputs — a bibliography, a principle
	// store, a changelog — each omit the page they feed rather than failing.
	bib, err := LoadBibliography(repoRoot, CSLRelPath, AcknowledgementsRelPath)
	if err != nil {
		return Result{}, err
	}
	recordRoot := ""
	if len(lintCfg.Roots) > 0 {
		recordRoot = lintCfg.Roots[0]
	}
	ex, err := newExplorer(c, export, bib, recordRoot)
	if err != nil {
		return Result{}, err
	}
	pages, err := ex.Pages()
	if err != nil {
		return Result{}, err
	}

	res := Result{
		OutDir:     outDir,
		Records:    len(export.Nodes),
		Links:      len(export.Edges),
		Mentions:   len(export.Mentions),
		Unresolved: len(export.Health.Unresolved),
		Baseline:   export.Health.BaselineCount,
		Overlaps:   export.Layout.Overlaps,
		Version:    stamp.Version,
		Commit:     stamp.Commit,
	}

	// The output tree is a RENDER of this commit, not an accumulation of every
	// build that ever ran here. A file left by an earlier one is stale the moment
	// this build does not rewrite it, and it is stale INVISIBLY — it is served,
	// and it looks exactly like a file that built.
	//
	// Every rule runs again here, at the instant of the first destructive step
	// — the path, the marker, the ownership — and the purge goes through the
	// handle regateOutDir opened over the real directory, never by path.
	root, err := regateOutDir(repoRoot, outDir, rootCommit)
	if err != nil {
		return Result{}, err
	}
	defer root.Close()

	write := func(rel string, data []byte) error {
		if !fsutil.ValidRelPath(rel) {
			return fmt.Errorf("site: refusing to write %q: not a relative path inside the output directory", rel)
		}
		if err := fsutil.WriteFileAtomicInRoot(root, rel, data, 0o644); err != nil {
			return err
		}
		res.Files = append(res.Files, rel)
		res.Bytes += int64(len(data))
		return nil
	}

	// FIRST, before anything else in the tree: a build killed halfway must still
	// leave a directory the next one recognises as its own. With the marker
	// written last, the wreckage of a failed build would carry none, and every
	// later build would refuse it — the tool jammed on its own debris, freed only
	// by hand. A tree with no root commit is freed by hand regardless: it has
	// no identity for the marker to name, and the next build refuses its
	// non-empty output by design.
	if err := write(siteMarkerName, siteMarker(rootCommit)); err != nil {
		return Result{}, err
	}
	if err := write("index.html", []byte(html)); err != nil {
		return Result{}, err
	}
	if err := write("record.json", recordJSON); err != nil {
		return Result{}, err
	}
	// Written in a fixed order, so two builds of one tree write the same files
	// in the same sequence and the reported list is a function of the record.
	routes := make([]string, 0, len(pages))
	for route := range pages {
		routes = append(routes, route)
	}
	sort.Strings(routes)
	for _, route := range routes {
		if err := write(route, []byte(pages[route])); err != nil {
			return Result{}, err
		}
	}
	res.Pages = len(pages)
	for _, cp := range copiedSources {
		data, err := fsutil.ReadGuardedInRoot(srcRoot, cp.src, maxAssetBytes)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Result{}, err
		}
		if err := write(cp.dst, data); err != nil {
			return Result{}, err
		}
	}
	// The installer. A repository that commits no template serves no /install.sh
	// — the same graceful absence every other copied source has.
	switch tmpl, err := fsutil.ReadGuardedInRoot(srcRoot, installTemplateRelPath, maxAssetBytes); {
	case err == nil:
		if err := write(installScriptName, renderInstallScript(tmpl, stamp)); err != nil {
			return Result{}, err
		}
	case os.IsNotExist(err):
	default:
		return Result{}, err
	}
	for _, pair := range c.assets.Copies() {
		data, err := fsutil.ReadGuardedInRoot(srcRoot, pair[0], maxAssetBytes)
		if err != nil {
			return Result{}, err
		}
		if err := write(path.Clean(pair[1]), data); err != nil {
			return Result{}, err
		}
	}
	sort.Strings(res.Files)
	return res, nil
}

// isPreOne reports whether a version's major component is 0 — the rule the Beta
// badge renders on, so no copy changes at v1.
func isPreOne(version string) bool {
	if version == "" {
		return false
	}
	major, _, _ := strings.Cut(strings.TrimPrefix(version, "v"), ".")
	return major == "0"
}

// Status is the read-only board the bare verb renders: what the repository has
// declared, and what the last build left behind.
type Status struct {
	Manifest  bool   `json:"manifest"`
	UIStrings bool   `json:"ui_strings"`
	UIPath    string `json:"ui_strings_path,omitempty"`
	Baseline  bool   `json:"baseline"`
	BaselineN int    `json:"baseline_entries"`
	// BaselinePath is the ratchet the board actually read: the manifest's
	// declaration where it makes one, the default otherwise.
	BaselinePath string `json:"baseline_path"`
	OutDir       string `json:"out_dir"`
	OutExists    bool   `json:"out_exists"`
	OutFiles     int    `json:"out_files"`
	// OutRefused is why the board did not look inside the output directory:
	// the same gate the build applies, reported rather than raised, because a
	// board that exits non-zero answers the question by refusing to answer it.
	OutRefused string `json:"out_refused,omitempty"`
	Chapters   int    `json:"chapters"`
	IssueLedge bool   `json:"issue_ledger"`
	Version    string `json:"version"`
	Commit     string `json:"commit"`
}

// Describe reads the site's declared state without writing anything.
func Describe(repoRoot, outDir string) (Status, error) {
	if outDir == "" {
		outDir = DefaultOutDir
	}
	st := Status{OutDir: outDir, Commit: HeadCommit(repoRoot), BaselinePath: BaselineRelPath}
	baselineRel := ""
	m, err := LoadManifest(repoRoot)
	switch {
	case err == nil:
		st.Manifest = true
		st.Chapters = len(m.Home.Chapters)
		st.IssueLedge = m.Record.IssueLedger
		st.UIPath = m.UIStrings
		if root, rerr := os.OpenRoot(repoRoot); rerr == nil {
			if _, serr := root.Stat(m.UIStrings); serr == nil {
				st.UIStrings = true
			}
			root.Close()
		}
		if baselineRel = m.Checks.UnresolvedReferenceBaseline; baselineRel != "" {
			st.BaselinePath = baselineRel
		}
	case os.IsNotExist(err):
	default:
		return Status{}, err
	}
	// The board reports the path it ACTUALLY read, so a reader can tell the
	// declared ratchet from the default one without opening the manifest.
	//
	// A baseline that is declared but missing, or present but unreadable, is
	// REPORTED here rather than raised. This verb is the read-only board, and a
	// broken baseline is precisely the kind of thing somebody runs it to
	// discover — a board that exits non-zero instead of saying "absent" answers
	// the question by refusing to answer it. `build` still refuses, because a
	// health count measured against nothing would be published as if it meant
	// something.
	b, ok, err := LoadBaseline(repoRoot, baselineRel)
	if err != nil {
		ok, b = false, Baseline{}
	}
	st.Baseline, st.BaselineN = ok, len(b.UnresolvedReferences)

	releases, _, err := changelog.DatedReleases(repoRoot)
	if err != nil {
		return Status{}, err
	}
	if len(releases) > 0 {
		st.Version = releases[0].Version
	}

	abs := outDir
	if !filepath.IsAbs(abs) {
		abs = joinRepo(repoRoot, outDir)
	}
	gated, gerr := resolveOutDir(repoRoot, abs)
	if gerr != nil {
		// Redacted at the source, so the text board and --json agree on this
		// field: OutRefused never carries an absolute developer-identity path
		// (iss-81; OutDir itself is captured as iss-2608291957114882).
		st.OutRefused = fsutil.RedactHome(fsutil.RedactRoot(gerr.Error(), repoRoot, "."))
		return st, nil
	}
	entries, err := os.ReadDir(gated)
	if err == nil {
		st.OutExists = true
		st.OutFiles = len(entries)
	}
	return st, nil
}
