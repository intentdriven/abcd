package ahoy

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"strings"

	"github.com/intentdriven/abcd/internal/core"
)

// pluginVersion is the version ahoy stamps into config.json["meta"] and
// compares against on upgrade detection. It tracks the build-stamped core.
func pluginVersion() string { return core.Version }

// ---------------------------------------------------------------------------
// git identity helpers
// ---------------------------------------------------------------------------

// originURL returns the trimmed origin remote URL with any credential
// scrubbed out of its userinfo (see scrubRemoteUserinfo), or "" on any failure.
// This is the only reader of the remote URL, so the scrub here is what keeps
// the registry files and the JSON surfaces credential-free.
func originURL(cwd string) string {
	out, err := runGit(cwd, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return scrubRemoteUserinfo(strings.TrimSpace(out))
}

func runGit(cwd string, args ...string) (string, error) {
	full := append([]string{"-C", cwd}, args...)
	cmd := exec.Command("git", full...)
	// Isolate: an inherited GIT_DIR/GIT_WORK_TREE overrides `-C cwd` and answers
	// for a DIFFERENT repository, so the root-commit SHA and origin URL this feeds
	// into RepoIdentity — which keys the cross-repo ~/.abcd/history registry and
	// drives install/refounding decisions — would be silently registered against
	// the wrong repo. rev-list/remote do not need global config, so full isolation
	// is safe here.
	cmd.Env = gitutil.IsolatedEnv()
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// deriveIdentity builds the deterministic RepoIdentity for cwd.
func deriveIdentity(cwd string) RepoIdentity {
	return RepoIdentity{
		Name:    filepath.Base(cwd),
		Github:  originURL(cwd),
		RootSHA: gitutil.RootCommit(cwd),
	}
}

// ---------------------------------------------------------------------------
// plugin root resolution
// ---------------------------------------------------------------------------

// osExecutable is os.Executable, overridable so a test can drive the
// executable-ancestor fallback without re-execing the test binary.
var osExecutable = os.Executable

// resolvePluginRoot resolves the plugin root via ABCD_PLUGIN_ROOT ->
// CLAUDE_PLUGIN_ROOT -> executable-ancestor fallback, validating the expected
// layout for each candidate. Returns ("", false) when every candidate fails.
func resolvePluginRoot() (string, bool) {
	var candidates []string
	if v := os.Getenv("ABCD_PLUGIN_ROOT"); v != "" {
		candidates = append(candidates, v)
	}
	if v := os.Getenv("CLAUDE_PLUGIN_ROOT"); v != "" {
		candidates = append(candidates, v)
	}
	if exe, err := osExecutable(); err == nil {
		// Walk up looking for a directory with the plugin layout. Canonicalise
		// first: os.Executable reports the path abcd was INVOKED through, and
		// `ahoy install` pins a PATH symlink (/usr/local/bin/abcd ->
		// <plugin-root>/abcd), so an unresolved path walks the symlink's
		// ancestors and never reaches the plugin root (iss-170). resolvePath
		// falls back to an absolutised form, then the original path, on error.
		dir := filepath.Dir(resolvePath(exe))
		for i := 0; i < 6 && dir != "/" && dir != "."; i++ {
			candidates = append(candidates, dir)
			dir = filepath.Dir(dir)
		}
	}
	// The owned-copy PATH entry (spc-35) is a regular file, so it has no symlink
	// target inside the plugin root for the ancestor walk to follow home — the
	// route the old pinned symlink doubled as. The home-scoped provenance record
	// carries the plugin root the copy was provisioned from, so it stands in as
	// the last candidate, AFTER the env-var and executable-ancestor routes: a
	// live CLAUDE_PLUGIN_ROOT is always preferred, and the recorded root — a
	// commit-stamped cache dir the harness replaces on update, which the hook-side
	// bootstrap re-stamps each time it provisions — is the terminal's fallback
	// (iss-2608210934566230). pluginRootValid still gates it, so a stale recorded
	// root that has been garbage-collected is skipped, not trusted.
	if rec, ok := readPathEntry(); ok && rec.pluginRoot != "" {
		candidates = append(candidates, rec.pluginRoot)
	}
	for _, c := range candidates {
		// Absolutise before validating OR returning. A relative candidate — the
		// shape a `cd abcd-cli && ABCD_PLUGIN_ROOT=.` user produces, and the env
		// var is first in the ladder — otherwise means one file to os.Stat (the
		// anti-dangling guard, resolving against the process CWD) and a different
		// file to os.Symlink (the kernel, resolving the stored string against the
		// LINK's directory). The guard then validates one path and links another,
		// writing a self-referential or dangling PATH entry it later reclassifies
		// as foreign so neither doctor nor uninstall can repair it (gh-334). The
		// sibling routes (executable-ancestor, --bin-dir, banlist scaffold) are
		// already absolutised; this closes the one gap. The exe-ancestor
		// candidates are already absolute, so filepath.Abs is a no-op for them.
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if pluginRootValid(abs) {
			return abs, true
		}
	}
	return "", false
}

// ResolvePluginRoot is the exported face of resolvePluginRoot for the front
// doors: the same ladder (ABCD_PLUGIN_ROOT -> CLAUDE_PLUGIN_ROOT ->
// executable-ancestor -> recorded path-entry root), the same layout check, so
// a surface never grows a second notion of where the plugin lives.
func ResolvePluginRoot() (string, bool) { return resolvePluginRoot() }

// pluginRootValid sanity-checks a candidate by verifying the expected plugin
// layout (a hooks/ directory).
func pluginRootValid(candidate string) bool {
	return isDir(filepath.Join(candidate, "hooks"))
}

// pluginBinaryPath is the binary a legacy owned PATH symlink points at (and the
// source the owned copy is provisioned from).
func pluginBinaryPath(pluginRoot string) string {
	return filepath.Join(pluginRoot, "abcd")
}

// binName is the name abcd occupies on PATH.
const binName = "abcd"

// userBinDir is the single-user install directory, ~/.local/bin — the location
// uv/pipx/rustup-class tools use, writable without privilege escalation. Empty
// when the home directory cannot be resolved.
func userBinDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "bin")
}

// binTarget is the DEFAULT PATH entry: ~/.local/bin/abcd (iss-171). abcd never
// escalates privileges, so a system-wide directory is reachable only through an
// explicit --bin-dir. Overridable for tests via ABCD_BIN_TARGET; empty when the
// home directory cannot be resolved, which every caller reads as "no target to
// act on" rather than falling back to a privileged path.
func binTarget() string {
	if v := os.Getenv("ABCD_BIN_TARGET"); v != "" {
		return v
	}
	dir := userBinDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, binName)
}

// pathDirs returns the PATH entries in order, dropping empties.
func pathDirs() []string {
	var dirs []string
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		if d != "" {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// dirOnPath reports whether dir is one of the PATH entries, comparing canonical
// forms so /home/x/.local/bin and a symlinked route to it are the same entry.
func dirOnPath(dir string) bool {
	if dir == "" {
		return false
	}
	want := resolvePath(dir)
	for _, d := range pathDirs() {
		if resolvePath(d) == want {
			return true
		}
	}
	return false
}

// pathEntry is one `abcd` found on PATH, classified.
type pathEntry struct {
	path     string        // the entry as it sits on PATH (unresolved)
	kind     binTargetKind // dev-shim / owned / foreign
	dangling bool          // ours, but the binary it points at is gone
}

// owned reports whether the entry is one abcd installed (the owned copy, a
// legacy pinned symlink, or the dev shim) rather than a foreign occupant.
func (e pathEntry) owned() bool {
	return e.kind == binTargetOwnedSymlink || e.kind == binTargetDevShim || e.kind == binTargetOwnedCopy
}

// scanPathEntries walks PATH in order and classifies every `abcd` it finds. It
// is the fix-the-detector half of iss-171: abcd is installed if abcd is on PATH,
// wherever it sits, rather than at one blessed location. Symlinks are resolved
// through the same seam as iss-170 (resolvePath -> filepath.EvalSymlinks), so a
// relative or indirect link classifies the same as a direct one.
func scanPathEntries(pluginRoot string) []pathEntry {
	var entries []pathEntry
	seen := map[string]bool{}
	for _, dir := range pathDirs() {
		candidate := filepath.Join(dir, binName)
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		if _, err := os.Lstat(candidate); err != nil {
			continue
		}
		e := pathEntry{path: candidate, kind: classifyBinTarget(candidate, pluginRoot)}
		// Stat FOLLOWS the link: a dangling entry is one whose target is gone. It
		// is computed for EVERY entry, ours or not — a foreign dangling `abcd`
		// still occupies the name and still shadows the entries behind it, and a
		// scan that only looked at our own would report it as nothing at all. A
		// stat error other than not-exist is not proof of a dangling link, so it
		// reads as healthy rather than manufacturing a gap.
		if present, err := fsutil.Exists(candidate); err == nil && !present {
			e.dangling = true
		}
		entries = append(entries, e)
	}
	return entries
}

// ownedPathEntry returns the first healthy abcd-owned entry on PATH. That entry
// IS the install: `ahoy install` adopts it in place rather than planting a second
// one, and detection reports it rather than a false symlink.missing.
func ownedPathEntry(pluginRoot string) (pathEntry, bool) {
	for _, e := range scanPathEntries(pluginRoot) {
		if e.owned() && !e.dangling {
			return e, true
		}
	}
	return pathEntry{}, false
}

// danglingPathEntry returns the first abcd-owned entry on PATH whose target has
// gone — a link that shadows whatever else on PATH would have answered.
func danglingPathEntry(pluginRoot string) (pathEntry, bool) {
	for _, e := range scanPathEntries(pluginRoot) {
		if e.owned() && e.dangling {
			return e, true
		}
	}
	return pathEntry{}, false
}

// effectiveBinTarget is the PATH entry every verb acts on: an existing owned
// install (adopted where it stands), else the default target.
func effectiveBinTarget(pluginRoot string) string {
	// An explicitly set ABCD_BIN_TARGET is an instruction, not a hint, and it
	// wins over adoption (iss-2608261447262355). Adoption is right when the
	// variable is unset — an install that finds an abcd it already owns should
	// update THAT one rather than leave a second copy elsewhere, which is how
	// `ahoy install` stays idempotent across the shapes a real machine presents.
	// But a caller that names a target has asked for a specific file, and the
	// only callers that name one are tests asking for a sandbox. Preferring an
	// owned entry over their ask is how `go test ./...` came to adopt a real
	// PATH entry and rewrite it into a temp dir: the machines that dogfood the
	// installer are exactly the machines with an owned entry to find.
	//
	// Inert in production, where ABCD_BIN_TARGET is unset and this falls through
	// to the adoption path unchanged.
	if os.Getenv("ABCD_BIN_TARGET") != "" {
		return binTarget()
	}
	if e, ok := ownedPathEntry(pluginRoot); ok {
		return e.path
	}
	return binTarget()
}

// sameEntry reports whether two PATH entries name the same file. The DIRECTORY
// is canonicalised and the leaf name compared verbatim: resolving the leaf would
// follow the symlink and compare an entry against the binary it points at, which
// is a different question (and would make an owned entry equal to every other
// link into the same plugin root).
func sameEntry(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if filepath.Base(a) != filepath.Base(b) {
		return false
	}
	return resolvePath(filepath.Dir(a)) == resolvePath(filepath.Dir(b))
}

// shadowingEntry returns the first `abcd` on PATH that precedes target — the
// binary that actually answers when the user types `abcd`, whatever abcd itself
// installed. Writing a correct entry BEHIND an existing one leaves the machine
// running the old binary while every report says the install is clean, which is
// exactly the state a copied pre-iss-171 install (a regular file at
// /usr/local/bin/abcd, foreign because it is not a symlink of ours) produces.
func shadowingEntry(pluginRoot, target string) (pathEntry, bool) {
	if target == "" {
		return pathEntry{}, false
	}
	for _, e := range scanPathEntries(pluginRoot) {
		if sameEntry(e.path, target) {
			return pathEntry{}, false // target is reached first: nothing shadows it
		}
		return e, true
	}
	return pathEntry{}, false
}

// describeEntry names what occupies a PATH entry, in the words a human needs to
// decide what to do about it. It never guesses: an entry abcd does not own is
// described by its shape, not by an assumption about its provenance.
func describeEntry(e pathEntry) string {
	switch {
	case e.kind == binTargetDevShim:
		return "abcd's own dev-mode shim"
	case e.kind == binTargetOwnedCopy:
		return "abcd's own installed copy"
	case e.kind == binTargetOwnedSymlink && e.dangling:
		return "an abcd symlink whose target is gone"
	case e.kind == binTargetOwnedSymlink:
		return "an abcd symlink"
	case e.dangling:
		return "a symlink whose target is gone"
	}
	if fi, err := os.Lstat(e.path); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if dest, rerr := os.Readlink(e.path); rerr == nil {
			return "a symlink to " + displayPath(dest) + ", which abcd does not own"
		}
		return "a symlink abcd does not own"
	}
	return "a file abcd does not own"
}

// shadowMessage is the single wording for a shadowed install, shared by the
// detection gap and the install-time note so the two can never drift.
func shadowMessage(e pathEntry, target string) string {
	return displayPath(e.path) + " (" + describeEntry(e) + ") comes before " +
		displayPath(target) + " on PATH, so it is what runs when you type `abcd`. " +
		"Remove or rename it, or install ahead of it with `abcd ahoy install --bin-dir <dir>`. " +
		"abcd never clobbers a binary it does not own."
}

// pathReachMessage is the single wording for an install directory that is not on
// PATH, shared by the detection gap and the install-time note.
func pathReachMessage(dir string) string {
	return displayPath(dir) + " is not on PATH, so `abcd` cannot be run by name. " +
		"Add it to your shell profile: " + exportPathLine(dir)
}

// dirWritable reports whether a file can actually be created in dir. It probes
// rather than reading the mode bits, so an ACL or a read-only mount answers
// honestly. A directory abcd cannot write to is refused, never escalated.
func dirWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".abcd-write-probe-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

// exportPathLine renders the one-line PATH fix for dir, in the form a user can
// paste. abcd prints it and never patches a shell profile (script-first).
func exportPathLine(dir string) string {
	d := displayPath(dir)
	if strings.HasPrefix(d, "~/") {
		d = "$HOME/" + strings.TrimPrefix(d, "~/")
	}
	return `export PATH="` + d + `:$PATH"`
}

// ---------------------------------------------------------------------------
// dev-mode shim (abcd ahoy install --dev)
// ---------------------------------------------------------------------------

// devShimMarker is the identifying first-comment line of the dev shim. Detection
// recognises the shim by this marker (never mistaking it for a foreign occupant),
// and uninstall removes only a file that carries it.
const devShimMarker = "# abcd-dev-shim"

// shSingleQuote wraps s for safe interpolation inside single quotes in a POSIX
// shell, so a source-repo path containing a quote cannot break out of the string.
func shSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// renderDevShim returns the POSIX-sh track-latest shim body. On every call it
// rebuilds abcd from sourceRepo's tip into freshBin and execs it; a failed build
// fails loudly and NEVER execs a stale binary (loud-staging). The absolute paths
// live only inside this installed file (outside any repo), never in a committed
// artefact.
func renderDevShim(sourceRepo, freshBin string) string {
	return "#!/bin/sh\n" +
		devShimMarker + " (managed by `abcd ahoy install --dev`)\n" +
		"# Rebuilds abcd from the source tip on every call, then execs the fresh\n" +
		"# binary. A broken build fails loudly and never execs a stale binary.\n" +
		"# Do not edit; run `abcd ahoy install` to pin a built binary instead.\n" +
		"ABCD_DEV_REPO=" + shSingleQuote(sourceRepo) + "\n" +
		"ABCD_DEV_BIN=" + shSingleQuote(freshBin) + "\n" +
		"if ! go build -C \"$ABCD_DEV_REPO\" -o \"$ABCD_DEV_BIN\" ./cmd/abcd; then\n" +
		"\tprintf 'abcd dev shim: build failed in %s — refusing to run a stale binary\\n' \"$ABCD_DEV_REPO\" >&2\n" +
		"\texit 1\n" +
		"fi\n" +
		"exec \"$ABCD_DEV_BIN\" \"$@\"\n"
}

// devShimPrefix is the exact opening of a shim renderDevShim produced: the
// shebang followed immediately by the marker line. Recognition requires this
// prefix — not a substring match anywhere in the file — so a foreign file that
// merely mentions the marker in its body is never mistaken for ours.
var devShimPrefix = []byte("#!/bin/sh\n" + devShimMarker)

// isDevShimFile reports whether path is a regular file we wrote as the dev shim.
// Uses the same guarded read as the rest of ahoy, so a symlink leaf or an
// oversized/irregular file reads as "not our shim" rather than being followed.
func isDevShimFile(path string) bool {
	data, err := fsutil.ReadGuarded(path, maxAhoyFileBytes)
	if err != nil {
		return false
	}
	return bytes.HasPrefix(data, devShimPrefix)
}

// binTargetKind classifies what currently occupies the PATH target.
type binTargetKind int

const (
	binTargetAbsent       binTargetKind = iota // nothing there
	binTargetOwnedSymlink                      // our LEGACY symlink -> the pinned binary (pre-spc-35)
	binTargetDevShim                           // our dev-mode rebuild-then-exec shim
	binTargetOwnedCopy                         // our regular-file copy, vouched for by path-entry (spc-35)
	binTargetForeign                           // something else; never clobber it
)

// classifyBinTarget inspects the PATH target and reports whether it is absent,
// our owned pinned symlink, our dev shim, or a foreign occupant.
func classifyBinTarget(target, pluginRoot string) binTargetKind {
	fi, err := os.Lstat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return binTargetAbsent
		}
		return binTargetForeign // present but unstattable: treat as foreign, never clobber
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		dest, err := os.Readlink(target)
		if err != nil {
			return binTargetForeign
		}
		if resolveSymlinkDest(target, dest) == resolvePath(pluginBinaryPath(pluginRoot)) {
			return binTargetOwnedSymlink
		}
		if strandedSiblingDest(target, dest, pluginRoot) {
			return binTargetOwnedSymlink
		}
		return binTargetForeign
	}
	if isDevShimFile(target) {
		return binTargetDevShim
	}
	// A regular file is ours only when the data dir's path-entry names this
	// very entry AND the bytes still hash to the recorded value (spc-35):
	// ownership is recorded provenance, never content-guessing.
	if isOwnedCopyFile(target) {
		return binTargetOwnedCopy
	}
	return binTargetForeign
}

// strandedSiblingDest reports whether a symlink's destination is the binary of
// a sibling plugin root that no longer exists — the entry every plugin update
// strands (iss-345): the harness provisions each update into a fresh cache dir
// and deletes the old one, so the pinned link points at <old-cache-dir>/abcd
// while the current root is <new-cache-dir>. Ownership extends to exactly that
// shape and no further: the leaf must be the binary name, the destination must
// be GONE (a live link into another root is somebody's working install, never
// adopted), and the dead root must share the current root's parent directory.
// The destination itself cannot be resolved (it no longer exists, so
// EvalSymlinks fails), so its grandparent — which survives the update — is
// resolved and compared against the parent of the RESOLVED current root; a
// plugin root that is itself a symlink out of its own parent therefore never
// matches, which fails toward foreign, the refusing side. In a source
// checkout the "parent" is the directory holding the checkout, so the sibling
// scope is every sibling project dir — wider than the plugin cache, still the
// developer's own tree, and still gated on the destination being gone.
func strandedSiblingDest(symlinkPath, dest, pluginRoot string) bool {
	if pluginRoot == "" {
		return false
	}
	if !filepath.IsAbs(dest) {
		dest = filepath.Join(filepath.Dir(symlinkPath), dest)
	}
	if filepath.Base(dest) != binName {
		return false
	}
	if present, err := fsutil.Exists(dest); err != nil || present {
		return false
	}
	oldRoot := filepath.Dir(dest)
	return resolvePath(filepath.Dir(oldRoot)) == filepath.Dir(resolvePath(pluginRoot))
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// ---------------------------------------------------------------------------
// ~/.abcd/history store
// ---------------------------------------------------------------------------

// historyRoot returns ~/.abcd/history. HOME is respected so tests can redirect.
func historyRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".abcd", "history"), nil
}

// historyIndex is the ~/.abcd/history/index.json registry.
type historyIndex struct {
	Schema      int           `json:"schema"`
	Description string        `json:"description"`
	Repos       []historyRepo `json:"repos"`
}

// historyRepo is one registry entry keyed on the immutable root_commit.
type historyRepo struct {
	RootCommit   string `json:"root_commit"`
	Name         string `json:"name"`
	Github       string `json:"github"`
	Path         string `json:"path"`
	Status       string `json:"status"`
	Supersedes   string `json:"supersedes,omitempty"`
	SupersededBy string `json:"superseded_by,omitempty"`
}

const historyIndexDescription = "abcd history/lifeboat registry. Keyed on each repo's root-commit SHA (immutable under rename, GitHub-handle change, or remote move). Names, GitHub URLs, and paths are mutable labels held in each repo's entry and refreshed by ahoy."

// loadHistoryIndex reads ~/.abcd/history/index.json and scrubs any credential
// out of every entry on the way in. Returns (nil,nil) when the store is not
// bootstrapped yet.
//
// The scrub is HERE, at the single load, and not only at the derivation site:
// scrubRemoteUserinfo cleans what a new registration writes, but a credential
// that reached the store before it existed is already at rest, and the only
// thing that ever rewrites the index is a load-modify-write through this
// function. Scrubbing as it reads therefore means any rewrite drops the
// credential — including from an entry the writer had no other reason to touch
// — and no renderer can print one it merely read. The at-rest file is inspected
// by readHistoryIndexFile, which is the detector's job, not a writer's.
func loadHistoryIndex() (*historyIndex, error) {
	idx, err := readHistoryIndexFile()
	if err != nil || idx == nil {
		return nil, err
	}
	for i := range idx.Repos {
		idx.Repos[i].Github = scrubRemoteUserinfo(idx.Repos[i].Github)
	}
	return idx, nil
}

// readHistoryIndexFile reads the index exactly as it is on disk, credentials
// included. Only the detector wants this: everything else goes through
// loadHistoryIndex, which scrubs.
func readHistoryIndexFile() (*historyIndex, error) {
	root, err := historyRoot()
	if err != nil {
		return nil, err
	}
	data, err := fsutil.ReadGuarded(filepath.Join(root, "index.json"), maxAhoyFileBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var idx historyIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

// metaGithub returns the github field of a per-repo meta.json, or "" when the
// file is absent, unreadable, or holds no such string.
func metaGithub(path string) string {
	data, err := fsutil.ReadGuarded(path, maxAhoyFileBytes)
	if err != nil {
		return ""
	}
	var meta map[string]any
	if err := json.Unmarshal(data, &meta); err != nil {
		return ""
	}
	g, _ := meta["github"].(string)
	return g
}

// scrubMetaCredential rewrites a per-repo meta.json whose github field carries
// userinfo, in place, preserving every other field. It reports whether it wrote.
//
// meta.json needs its own heal because — unlike index.json, which every
// registration rewrites through loadHistoryIndex — it is written once when the
// repo first registers and never revisited, so a credential in it outlives every
// later install on its own.
func scrubMetaCredential(path string) bool {
	g := metaGithub(path)
	clean := scrubRemoteUserinfo(g)
	if g == "" || clean == g {
		return false
	}
	data, err := fsutil.ReadGuarded(path, maxAhoyFileBytes)
	if err != nil {
		return false
	}
	var meta map[string]any
	if err := json.Unmarshal(data, &meta); err != nil {
		return false
	}
	meta["github"] = clean
	return writeJSON(path, meta) == nil
}

// indexHasRoot reports whether idx registers rootSHA.
func indexHasRoot(idx *historyIndex, rootSHA string) bool {
	if idx == nil || rootSHA == "" {
		return false
	}
	for _, r := range idx.Repos {
		if r.RootCommit == rootSHA {
			return true
		}
	}
	return false
}

// indexEntry returns the entry for rootSHA, or nil.
func indexEntry(idx *historyIndex, rootSHA string) *historyRepo {
	if idx == nil {
		return nil
	}
	for i := range idx.Repos {
		if idx.Repos[i].RootCommit == rootSHA {
			return &idx.Repos[i]
		}
	}
	return nil
}

// findRefoundingCandidate returns an entry with a matching name (or non-empty
// matching github) but a different root_commit, or nil.
func findRefoundingCandidate(idx *historyIndex, id RepoIdentity) *historyRepo {
	if idx == nil {
		return nil
	}
	for i := range idx.Repos {
		e := &idx.Repos[i]
		if e.RootCommit == id.RootSHA {
			continue
		}
		nameMatch := e.Name == id.Name
		githubMatch := id.Github != "" && e.Github != "" && e.Github == id.Github
		if nameMatch || githubMatch {
			return e
		}
	}
	return nil
}

// historyLockFilename is the ~/.abcd/history lock file guarding the index.json
// load-modify-write. It sits beside index.json.
const historyLockFilename = ".index.lock"

// historyLockTimeout bounds acquisition of the history-index lock. A var (not
// const) so a test can shorten it to exercise contention without a multi-second
// wait, mirroring capture's lockTimeout.
var historyLockTimeout = 5 * time.Second

// afterHistoryReloadHook, when non-nil, fires inside withHistoryLock immediately
// after the lock is acquired and before fn runs (fn re-loads the index under the
// lock and writes it). It is a test-only seam (nil in production) used to force
// the iss-101 lost-update interleaving deterministically: the goroutine that grabs
// the lock pauses here, holding it, while another registration is admitted.
var afterHistoryReloadHook func()

// beforeHistoryIndexCreateHook, when non-nil, fires in bootstrapHistory just
// before the atomic link that publishes index.json. Test-only seam (nil in
// production) to force the iss-101 bootstrap check-then-act interleaving
// deterministically.
var beforeHistoryIndexCreateHook func()

// withHistoryLock runs fn while holding the exclusive ~/.abcd/history lock, so a
// registerRepo load-modify-write of index.json serializes across concurrent
// `abcd ahoy install` runs from different worktrees (iss-101). It routes through
// the shared fsutil.WithFileLock primitive; the lock file lives beside
// index.json. The lock is NEVER held across an interactive prompt — callers do
// any prompting before acquiring it and re-check the answer-relevant state inside
// fn after re-loading.
func withHistoryLock(fn func() error) error {
	root, err := historyRoot()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	lockPath := filepath.Join(root, historyLockFilename)
	return fsutil.WithFileLock(lockPath, historyLockTimeout, func() error {
		if afterHistoryReloadHook != nil {
			afterHistoryReloadHook()
		}
		return fn()
	})
}

// bootstrapHistory creates ~/.abcd/history/ + index.json when absent. Idempotent.
// The seed is published atomically (iss-101): a fully-formed temp file is written,
// synced, and os.Link'd into place. The link is the single-winner publish — it
// fails EEXIST if index.json already exists, so under two concurrent bootstraps
// exactly one wins and every other observes the existing file and reports no
// write. Because the file only ever appears complete (the link, never a bare
// create followed by a separate content write), a reader that races the bootstrap
// sees either no file yet or the finished index — never a 0-byte one that would
// make a concurrent loadHistoryIndex parse-fail and drop its own registration.
func bootstrapHistory() (bool, error) {
	root, err := historyRoot()
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return false, err
	}
	path := filepath.Join(root, "index.json")
	// Fast path: already seeded (the common idempotent re-run).
	if _, err := os.Stat(path); err == nil {
		return false, nil
	}
	idx := historyIndex{Schema: 1, Description: historyIndexDescription, Repos: []historyRepo{}}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return false, err
	}
	data = append(data, '\n')

	// Write a complete temp file first, then link it into place as the atomic,
	// single-winner publish. The temp is always cleaned up.
	tmp, err := os.CreateTemp(root, ".index-*.tmp")
	if err != nil {
		return false, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return false, err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return false, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}

	if beforeHistoryIndexCreateHook != nil {
		beforeHistoryIndexCreateHook()
	}
	if err := os.Link(tmpName, path); err != nil {
		if os.IsExist(err) {
			return false, nil // already seeded (or won by a concurrent run)
		}
		return false, err
	}
	return true, nil
}

// writeHistoryIndex persists idx. It must be called from inside withHistoryLock
// (registerRepo) so a re-loaded index is not clobbered by a concurrent writer.
func writeHistoryIndex(idx *historyIndex) error {
	root, err := historyRoot()
	if err != nil {
		return err
	}
	return writeJSON(filepath.Join(root, "index.json"), *idx)
}

// ---------------------------------------------------------------------------
// repo config.json
// ---------------------------------------------------------------------------

// maxAhoyFileBytes caps ahoy's marker/config/index reads. Markers (CLAUDE.md /
// AGENTS.md), config.json, and the history index are small text/JSON; 1 MiB is
// far above any real one, so the cap never rejects a legitimate file while it
// bounds a FIFO or multi-GB file planted at a cwd-influenced path (iss-97).
const maxAhoyFileBytes = 1 << 20

// configPath is the repo-scope .abcd/config.json path.
func configPath(cwd string) string {
	return filepath.Join(cwd, ".abcd", "config.json")
}

// configRelPath and rulesRelPath are the repo-.abcd writers' os.Root-relative,
// slash-separated targets. They are the containment counterparts of the
// filepath.Join(cwd, …) forms: a write resolved through an os.Root opened at the
// repo cannot follow a committed `.abcd` ancestor symlink outside the tree
// (GHSA-xrf8-4432-gw2f), the same idiom the banlist scaffold already uses.
const (
	configRelPath = ".abcd/config.json"
	rulesRelPath  = ".abcd/rules.json"
)

// readConfig reads .abcd/config.json into a generic map. Returns (nil,nil) when
// absent, and an error on malformed JSON or a guarded-read refusal (a non-regular
// leaf such as a FIFO, or a file past the size cap). Callers treat any error as
// unusable config and fail safe.
func readConfig(cwd string) (map[string]any, error) {
	data, err := fsutil.ReadGuarded(configPath(cwd), maxAhoyFileBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// subMap returns cfg[name] as a map, or an empty map.
func subMap(cfg map[string]any, name string) map[string]any {
	if cfg == nil {
		return map[string]any{}
	}
	if v, ok := cfg[name].(map[string]any); ok {
		return v
	}
	return map[string]any{}
}

// writeJSON marshals v deterministically (map keys sorted) with a trailing
// newline via the atomic writer. It guards the LEAF only, so it is reserved for
// the ~/.abcd/history writers, whose paths are the user's own home and never
// influenced by committed repo content.
func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return fsutil.WriteFileAtomicPreserveMode(path, data)
}

// marshalJSON renders v the way writeJSON persists it: deterministic, trailing
// newline. Split out so the contained writer can share the exact bytes.
func marshalJSON(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// writeRepoJSON persists v to a repo-.abcd path (slash-separated, relative to
// cwd) THROUGH an os.Root opened at cwd, so a committed `.abcd` ancestor symlink
// cannot land the write outside the working tree (GHSA-xrf8-4432-gw2f). It is the
// contained counterpart of writeJSON and the only writer the config/rules paths
// use.
func writeRepoJSON(cwd, rel string, v any) error {
	data, err := marshalJSON(v)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(cwd)
	if err != nil {
		return err
	}
	defer root.Close()
	return fsutil.WriteFileAtomicPreserveModeInRoot(root, rel, data)
}

// writeConfig persists a config map deterministically, contained under cwd.
func writeConfig(cwd string, cfg map[string]any) error {
	return writeRepoJSON(cwd, configRelPath, cfg)
}

// ---------------------------------------------------------------------------
// hook manifest verification (verify-only)
// ---------------------------------------------------------------------------

// requiredHookCommand is the substring each event's command must contain. These
// are the Go hook subcommands (`abcd hook prompt-router` / `prompt-router-reset`)
// as wired in hooks/hooks.json — the loader is a Go subcommand, not a script.
var requiredHookCommand = map[string]string{
	"UserPromptSubmit": "hook prompt-router",
	"SessionStart":     "hook prompt-router-reset",
	"PreCompact":       "hook prompt-router-reset",
}

// verifyHookManifest returns "" when hooks/hooks.json under pluginRoot is
// structurally sound, else a one-line reason. Read-only; never mutates.
func verifyHookManifest(pluginRoot string) string {
	path := filepath.Join(pluginRoot, "hooks", "hooks.json")
	// Keep the symlink pre-check: it yields a DISTINCT reason that ReadGuarded's
	// O_NOFOLLOW would otherwise collapse into a raw ELOOP -> "read failed".
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return "leaf is a symlink (refusing to follow)"
	}
	// ReadGuarded validates regular-file + size cap on the SAME descriptor it
	// reads, closing the Lstat->ReadFile swap window (iss-109). Map its sentinels
	// back to the existing reason strings, unchanged.
	data, err := fsutil.ReadGuarded(path, 256*1024)
	if err != nil {
		switch {
		case errors.Is(err, fsutil.ErrNotRegular):
			return "not a regular file"
		case errors.Is(err, fsutil.ErrTooBig):
			return "file size exceeds 256KB cap"
		case os.IsNotExist(err):
			return "file absent"
		default:
			return "read failed"
		}
	}
	hooks, reason := decodeHookEvents(data)
	if reason != "" {
		return reason
	}
	for _, event := range []string{"UserPromptSubmit", "SessionStart", "PreCompact"} {
		entries, ok := hooks[event].([]any)
		if !ok || len(entries) == 0 {
			return "missing or empty `hooks." + event + "` array"
		}
		if !eventHasCommand(entries, requiredHookCommand[event]) {
			return "`hooks." + event + "` does not reference " + requiredHookCommand[event]
		}
	}
	return ""
}

// decodeHookEvents decodes a hook manifest's `hooks` object, returning the
// per-event entries or a one-line reason. It is the single decoder for the
// manifest: verifyHookManifest checks the prompt-router events with it, and the
// guard-health check reads the PreToolUse event with it, so the two can never
// disagree about what a well-formed manifest is.
func decodeHookEvents(data []byte) (map[string]any, string) {
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, "JSON parse failed"
	}
	hooks, ok := parsed["hooks"].(map[string]any)
	if !ok {
		return nil, "missing or non-object `hooks` key"
	}
	return hooks, ""
}

// readHookEvents reads and decodes the plugin's hook manifest, returning the
// per-event entries. A manifest that cannot be read or decoded yields ok=false —
// the caller reports "not installed", which is the honest answer when nothing can
// be proven about the wiring.
func readHookEvents(pluginRoot string) (map[string]any, bool) {
	data, err := fsutil.ReadGuarded(filepath.Join(pluginRoot, "hooks", "hooks.json"), 256*1024)
	if err != nil {
		return nil, false
	}
	hooks, reason := decodeHookEvents(data)
	return hooks, reason == ""
}

// eventHasCommand reports whether any nested command string contains substring.
func eventHasCommand(entries []any, substring string) bool {
	for _, entry := range entries {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		inner, ok := m["hooks"].([]any)
		if !ok {
			continue
		}
		for _, h := range inner {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if cmd, ok := hm["command"].(string); ok && strings.Contains(cmd, substring) {
				return true
			}
		}
	}
	return false
}
