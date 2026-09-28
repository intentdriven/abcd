// Package launch is abcd's transport-agnostic launch engine: it assembles the
// release bundle under a default-deny taxonomy, runs the native secret+PII scan,
// checks manifest lockstep, and previews newest-per-line retention — all as a
// dry-run that renders decisions without writing an artefact or touching the
// network. It performs no printing and no os.Exit, so it is fully testable and
// reusable across surfaces.
//
// The load-bearing invariant (adr-18/adr-28): the .abcd/** namespace and every
// other denied namespace can NEVER enter the bundle. This is a STRUCTURAL deny,
// not an allowlist toggle — no include pattern can promote a denied path.
package launch

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// DenyNamespaces are namespace names that never ship. The structural deny
// (adr-18) binds EVERY path component case-insensitively (see
// pathContainsDeniedSegment), not just the first segment: a denied name nested
// under an included tree, or spelled in a different case, is denied all the same
// (GHSA-g2v7-wfmv-v28r, #335). NOT overridable by any allowlist. Mirrors
// launch_resolve DENY_NAMESPACES.
var DenyNamespaces = map[string]struct{}{
	".git": {}, ".abcd": {}, ".flow": {}, ".work": {}, ".specstory": {}, "memory": {},
}

// ExcludedReason is why a candidate was benignly excluded (never fails a ship).
type ExcludedReason string

const (
	ExcludedGitignored      ExcludedReason = "gitignored"
	ExcludedUnmatchedGlob   ExcludedReason = "unmatched_glob"
	ExcludedDeniedNamespace ExcludedReason = "denied_namespace"
)

// RejectedReason is why a candidate was rejected (any entry fails a ship).
type RejectedReason string

const (
	RejectedDeny            RejectedReason = "deny"
	RejectedSymlinkEscape   RejectedReason = "symlink_escape"
	RejectedSymlinkCycle    RejectedReason = "symlink_cycle"
	RejectedHardlinkDenied  RejectedReason = "hardlink_denied"
	RejectedHardlinkOffrepo RejectedReason = "hardlink_offrepo"
	RejectedDuplicate       RejectedReason = "duplicate"
	RejectedControlChar     RejectedReason = "control_char"
	RejectedPlatformBinary  RejectedReason = "platform_binary"
	RejectedMissingLiteral  RejectedReason = "missing_literal"
	RejectedFSError         RejectedReason = "fs_error"
)

// IncludedFile is a resolved payload file. Paths are repo-relative POSIX;
// ResolvedPath is the absolute on-disk (dereferenced) path every reader opens
// the file through, and it never reaches machine output (iss-81):
// DisplayResolvedPath is the same file named relative to the repository, which
// is what a report carries as resolved_path (iss-2609261954288630).
type IncludedFile struct {
	LogicalPath         string `json:"logical_path"`
	ResolvedPath        string `json:"-"`
	DisplayResolvedPath string `json:"resolved_path"`
	GitMode             string `json:"git_mode"` // "100644" | "100755"
}

// ExcludedFile is a benign exclusion.
type ExcludedFile struct {
	LogicalPath string         `json:"logical_path"`
	Reason      ExcludedReason `json:"reason"`
}

// RejectedFile is a violation.
type RejectedFile struct {
	LogicalPath string            `json:"logical_path"`
	Reason      RejectedReason    `json:"reason"`
	Details     map[string]string `json:"details,omitempty"`
}

// Bundle is the classified resolution outcome.
type Bundle struct {
	Included []IncludedFile `json:"files"`
	Excluded []ExcludedFile `json:"excluded"`
	Rejected []RejectedFile `json:"rejected"`
	Warnings []string       `json:"warnings"`
}

// HasViolation reports whether any rejected[] entry exists. ship hard-fails on
// true; dry-run reports it but still exits 0.
func (b Bundle) HasViolation() bool { return len(b.Rejected) > 0 }

// ScriptsClosureDenyDirs / ScriptsClosureDenySuffixes are the closure's own
// default-deny (dev-only names / compiled suffixes); a scripts/ path matching
// them is a benign excluded(denied_namespace) prune, not a resolution error.
var (
	scriptsDenyDirs     = map[string]struct{}{"__pycache__": {}, ".git": {}, ".mypy_cache": {}, ".pytest_cache": {}, "ralph": {}, "_intent_lint": {}}
	scriptsDenySuffixes = []string{".pyc", ".pyo"}
)

// ResolveBundle walks repoRoot, matches candidates against includes, and
// classifies each into Included / Excluded / Rejected under the ordered
// algorithm. includes==nil loads the committed config via LoadIncludes (a
// preflight fault is returned as an error).
func ResolveBundle(repoRoot string, includes []string) (Bundle, error) {
	return resolveBundle(repoRoot, includes, defaultClosureFn)
}

// resolveBundle is the injectable-closure implementation (ClosureFn is the one
// open dependency, spec §1 step 10).
func resolveBundle(repoRoot string, includes []string, closureFn ClosureFn) (Bundle, error) {
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return Bundle{}, err
	}
	if real, err := filepath.EvalSymlinks(absRoot); err == nil {
		absRoot = real
	}

	if includes == nil {
		includes, err = LoadIncludes(absRoot)
		if err != nil {
			return Bundle{}, err
		}
	}

	// Reject a malformed glob include (e.g. an invalid char-class range like
	// [z-a]) as a preflight fault up front, rather than panicking mid-walk when
	// the pattern is first compiled — every other config fault here is graceful.
	for _, inc := range includes {
		if isGlob(inc) {
			if err := validateGlobInclude(inc); err != nil {
				return Bundle{}, err
			}
		}
	}

	var closure map[string]struct{}
	if anyReachesScripts(includes) && closureFn != nil {
		closure, err = closureFn(absRoot)
		if err != nil {
			return Bundle{}, preflight("scripts closure unreadable: %v", err)
		}
	}

	r := &resolver{
		root:         absRoot,
		includes:     includes,
		closure:      closure,
		matchedGlobs: map[string]struct{}{},
		inode:        buildInodeMap(absRoot),
	}

	// Missing-literal check: a literal include with no on-disk entry (Lstat, so a
	// dangling symlink counts as present) → rejected(missing_literal).
	for _, inc := range includes {
		if isGlob(inc) || inc == "." || inc == "" {
			continue
		}
		if _, err := os.Lstat(filepath.Join(absRoot, filepath.FromSlash(inc))); err != nil {
			r.result.Rejected = append(r.result.Rejected, RejectedFile{LogicalPath: inc, Reason: RejectedMissingLiteral})
		}
	}

	// Walk + structural passes, collecting survivors pending the ignore pass.
	r.walkDir("", absRoot, map[string]struct{}{})

	// Unmatched globs → excluded(unmatched_glob).
	for _, inc := range includes {
		if isGlob(inc) {
			if _, ok := r.matchedGlobs[inc]; !ok {
				r.result.Excluded = append(r.result.Excluded, ExcludedFile{LogicalPath: inc, Reason: ExcludedUnmatchedGlob})
			}
		}
	}

	// Ignore pass (batched) + duplicate resolution → Included.
	r.finalize()

	sortBundle(&r.result)
	return r.result, nil
}

// resolver carries mutable state through the walk.
type resolver struct {
	root         string
	includes     []string
	closure      map[string]struct{}
	inode        *inodeMap
	matchedGlobs map[string]struct{}
	survivors    []candidate
	result       Bundle
}

// candidate is a survivor of the structural passes, pending the ignore pass.
type candidate struct {
	logical  string
	resolved string
	dev, ino uint64
	gitMode  string
	deref    bool
}

func (r *resolver) walkDir(rel, absDir string, ancestors map[string]struct{}) {
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return // an unwalkable subtree already flagged the inode map uncertain
	}
	for _, e := range entries {
		name := e.Name()
		childRel := name
		if rel != "" {
			childRel = rel + "/" + name
		}
		childAbs := filepath.Join(absDir, name)

		if hasControlChar(childRel) {
			r.result.Rejected = append(r.result.Rejected, RejectedFile{LogicalPath: childRel, Reason: RejectedControlChar})
			continue
		}
		info, err := os.Lstat(childAbs)
		if err != nil {
			continue
		}
		mode := info.Mode()
		switch {
		case mode&os.ModeSymlink != 0:
			r.handleSymlink(childRel, childAbs, ancestors)
		case mode.IsDir():
			if pathContainsDeniedSegment(childRel) {
				// Structural deny BEFORE ignore: a denied dir reached by a broad
				// include is excluded(denied_namespace); otherwise silently pruned.
				// Either way it is never descended — .abcd/** cannot enter here.
				if r.anyIncludeMatches(childRel) {
					r.result.Excluded = append(r.result.Excluded, ExcludedFile{LogicalPath: childRel, Reason: ExcludedDeniedNamespace})
				}
				continue
			}
			r.walkDir(childRel, childAbs, ancestors)
		case mode.IsRegular():
			r.classifyRegular(childRel, childAbs, info, false)
		default:
			if r.firstMatchAndMark(childRel) != "" {
				r.result.Rejected = append(r.result.Rejected, RejectedFile{LogicalPath: childRel, Reason: RejectedFSError})
			}
		}
	}
}

// classifyRegular applies include-match, denied, scripts-closure and hardlink
// passes to one regular file, appending a survivor when it passes.
func (r *resolver) classifyRegular(rel, abs string, info os.FileInfo, deref bool) {
	source := r.firstMatchAndMark(rel)
	if source == "" {
		return // default-deny: not requested at all
	}
	if pathContainsDeniedSegment(rel) {
		r.result.Excluded = append(r.result.Excluded, ExcludedFile{LogicalPath: rel, Reason: ExcludedDeniedNamespace})
		return
	}
	// A released platform binary never ships in the payload: the plugin's
	// bootstrap provisions abcd by checksum-verified download from the release,
	// so a bundled copy would be an unverified, permanently stale second source
	// for the very binary that then runs as the shell guard (itd-154). Today the
	// artefacts are also gitignored and reached by no include — but that is a
	// config an edit can undo, so the deny binds any candidate an include DOES
	// reach, and it rejects rather than excluding: naming one is a mistake worth
	// failing on, not a candidate to drop in silence. It sits after the include
	// match rather than before it so an artefact nobody asked for stays an
	// ordinary default-deny miss instead of a reported violation.
	if isPlatformBinaryName(path.Base(rel)) {
		r.result.Rejected = append(r.result.Rejected, RejectedFile{LogicalPath: rel, Reason: RejectedPlatformBinary})
		return
	}
	// For a dereferenced candidate the LOGICAL path is benign but the real target
	// may sit under a denied namespace reached through a symlink chain the
	// immediate-target check at handleSymlink could not see (e.g. a link to the
	// repo root, then a nested walk into .git/objects — whose blobs the hardlink
	// inode map deliberately exempts). Re-run the structural deny on the real path.
	if deref && r.realPathDenied(abs) {
		r.result.Rejected = append(r.result.Rejected, RejectedFile{LogicalPath: rel, Reason: RejectedDeny})
		return
	}
	if firstSegment(rel) == "scripts" && r.closure != nil {
		if _, in := r.closure[rel]; !in {
			if r.anyLiteralFileInclude(rel) && !scriptsDenied(rel) {
				r.result.Rejected = append(r.result.Rejected, RejectedFile{
					LogicalPath: rel, Reason: RejectedFSError,
					Details: map[string]string{"kind": "scripts_not_in_runtime_closure"},
				})
				return
			}
			r.result.Excluded = append(r.result.Excluded, ExcludedFile{LogicalPath: rel, Reason: ExcludedDeniedNamespace})
			return
		}
	}

	dev, ino := inodeOf(info)
	// Hardlink alias map, fail-closed: any uncertainty rejects fs_error.
	if r.inode.uncertain {
		r.result.Rejected = append(r.result.Rejected, RejectedFile{LogicalPath: rel, Reason: RejectedFSError})
		return
	}
	if r.inode.aliasDenied(dev, ino) {
		r.result.Rejected = append(r.result.Rejected, RejectedFile{LogicalPath: rel, Reason: RejectedHardlinkDenied})
		return
	}

	gitMode := "100644"
	if info.Mode()&0o111 != 0 {
		gitMode = "100755"
	}
	r.survivors = append(r.survivors, candidate{
		logical: rel, resolved: abs, dev: dev, ino: ino, gitMode: gitMode, deref: deref,
	})
}

// included is the Included entry for one surviving candidate: the working
// absolute path every reader opens, and the same file named relative to the
// repository for the report.
func (r *resolver) included(c candidate) IncludedFile {
	return IncludedFile{
		LogicalPath:         c.logical,
		ResolvedPath:        c.resolved,
		DisplayResolvedPath: fsutil.DisplayPath(r.root, c.resolved),
		GitMode:             c.gitMode,
	}
}

// handleSymlink resolves a symlink structurally (escape/cycle/deny) and, when
// accepted, dereferences it: a file is classified under its logical path; a
// directory is walked with its contents emitted under the symlink's prefix. A
// symlink is only classified when an include could reach it (default-deny).
func (r *resolver) handleSymlink(rel, abs string, ancestors map[string]struct{}) {
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if !r.anyIncludeMatches(rel) && !r.includeMayReachDir(rel) {
			return
		}
		reason := RejectedFSError
		if strings.Contains(err.Error(), "too many links") {
			reason = RejectedSymlinkCycle
		}
		r.result.Rejected = append(r.result.Rejected, RejectedFile{LogicalPath: rel, Reason: reason})
		return
	}
	relToRoot, err := filepath.Rel(r.root, real)
	if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		if r.anyIncludeMatches(rel) || r.includeMayReachDir(rel) {
			r.result.Rejected = append(r.result.Rejected, RejectedFile{LogicalPath: rel, Reason: RejectedSymlinkEscape})
		}
		return
	}
	realRel := filepath.ToSlash(relToRoot)
	if pathContainsDeniedSegment(realRel) {
		// A symlink whose target realpath is under a denied namespace is a
		// smuggling attempt — reject(deny) even if the logical path is benign.
		if r.anyIncludeMatches(rel) || r.includeMayReachDir(rel) {
			r.result.Rejected = append(r.result.Rejected, RejectedFile{LogicalPath: rel, Reason: RejectedDeny})
		}
		return
	}
	tinfo, err := os.Stat(real)
	if err != nil {
		return
	}
	if tinfo.IsDir() {
		if !r.includeMayReachDir(rel) {
			return
		}
		if _, seen := ancestors[real]; seen {
			r.result.Rejected = append(r.result.Rejected, RejectedFile{LogicalPath: rel, Reason: RejectedSymlinkCycle})
			return
		}
		next := map[string]struct{}{real: {}}
		for k := range ancestors {
			next[k] = struct{}{}
		}
		r.walkSymlinkTarget(rel, real, next)
		return
	}
	if tinfo.Mode().IsRegular() {
		r.classifyRegular(rel, real, tinfo, true)
	}
}

// walkSymlinkTarget walks a dereferenced symlink-target directory, emitting each
// regular file under logicalPrefix and recursing (with a cycle guard) into
// nested symlink dirs so a nested target under a denied namespace is still
// rejected rather than silently skipped.
func (r *resolver) walkSymlinkTarget(logicalPrefix, realDir string, ancestors map[string]struct{}) {
	entries, err := os.ReadDir(realDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		childLogical := logicalPrefix + "/" + name
		childAbs := filepath.Join(realDir, name)
		if hasControlChar(childLogical) {
			r.result.Rejected = append(r.result.Rejected, RejectedFile{LogicalPath: childLogical, Reason: RejectedControlChar})
			continue
		}
		info, err := os.Lstat(childAbs)
		if err != nil {
			continue
		}
		mode := info.Mode()
		switch {
		case mode&os.ModeSymlink != 0:
			r.handleSymlink(childLogical, childAbs, ancestors)
		case mode.IsDir():
			// Re-apply the structural deny on the REAL path at every level of a
			// dereferenced walk, so a symlink to (or into) the repo root can never
			// descend into .git/** or .abcd/** the way walkDir already prevents for
			// in-tree directories.
			if r.realPathDenied(childAbs) {
				if r.includeMayReachDir(childLogical) {
					r.result.Rejected = append(r.result.Rejected, RejectedFile{LogicalPath: childLogical, Reason: RejectedDeny})
				}
				continue
			}
			r.walkSymlinkTarget(childLogical, childAbs, ancestors)
		case mode.IsRegular():
			r.classifyRegular(childLogical, childAbs, info, true)
		}
	}
}

// realPathDenied reports whether a real (already-dereferenced) absolute path
// resolves outside the repo root or under a denied namespace. It is the deny gate
// for dereferenced walks, where the logical path is benign but the real target is
// what actually ships. Fail-closed: an unresolvable relation counts as denied.
func (r *resolver) realPathDenied(abs string) bool {
	rel, err := filepath.Rel(r.root, abs)
	if err != nil {
		return true
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return true
	}
	return pathContainsDeniedSegment(rel)
}

// finalize applies the batched ignore pass and duplicate resolution, promoting
// surviving candidates to Included.
func (r *resolver) finalize() {
	// Batch the ignore probe over every survivor's logical path AND, for a
	// dereferenced candidate, the real target's repo-relative path — the target is
	// the content that actually ships, so a gitignored file aliased by a benign
	// symlink name must still be excluded.
	resolvedRel := make(map[string]string, len(r.survivors))
	paths := make([]string, 0, len(r.survivors))
	seenPath := map[string]struct{}{}
	addPath := func(p string) {
		if _, ok := seenPath[p]; ok {
			return
		}
		seenPath[p] = struct{}{}
		paths = append(paths, p)
	}
	for _, c := range r.survivors {
		addPath(c.logical)
		if c.deref {
			if rr, err := filepath.Rel(r.root, c.resolved); err == nil {
				rr = filepath.ToSlash(rr)
				if rr != ".." && !strings.HasPrefix(rr, "../") {
					resolvedRel[c.resolved] = rr
					addPath(rr)
				}
			}
		}
	}
	ignored, err := ignoreChecker(r.root, paths)
	if err != nil {
		// Fail closed (mirrors the uncertain-inode-map gate at classifyRegular):
		// git is present and root IS a repo, but the gitignore probe itself errored,
		// so we cannot prove a survivor is not gitignored. Reject every survivor
		// rather than promote a possibly-ignored secret to Included.
		for _, c := range r.survivors {
			r.result.Rejected = append(r.result.Rejected, RejectedFile{
				LogicalPath: c.logical, Reason: RejectedFSError,
				Details: map[string]string{"kind": "gitignore_check_failed"},
			})
		}
		return
	}

	// Group by logical path for duplicate-by-provenance resolution.
	byLogical := map[string][]candidate{}
	var order []string
	for _, c := range r.survivors {
		_, logicalIgnored := ignored[c.logical]
		_, targetIgnored := ignored[resolvedRel[c.resolved]]
		if logicalIgnored || (c.deref && targetIgnored) {
			r.result.Excluded = append(r.result.Excluded, ExcludedFile{LogicalPath: c.logical, Reason: ExcludedGitignored})
			continue
		}
		if _, seen := byLogical[c.logical]; !seen {
			order = append(order, c.logical)
		}
		byLogical[c.logical] = append(byLogical[c.logical], c)
	}

	for _, logical := range order {
		group := byLogical[logical]
		if len(group) == 1 {
			c := group[0]
			r.result.Included = append(r.result.Included, r.included(c))
			continue
		}
		// Same logical path from multiple sources: same inode → dedup with a
		// warning; distinct inode → rejected(duplicate).
		first := group[0]
		sameInode := true
		for _, c := range group[1:] {
			if c.dev != first.dev || c.ino != first.ino {
				sameInode = false
				break
			}
		}
		if sameInode {
			r.result.Warnings = append(r.result.Warnings, "duplicate provenance for "+logical+" (same inode); kept one")
			r.result.Included = append(r.result.Included, r.included(first))
		} else {
			r.result.Rejected = append(r.result.Rejected, RejectedFile{LogicalPath: logical, Reason: RejectedDuplicate})
		}
	}
}

// ignoreChecker is the bundle gate's gitignore probe. It is a package var so a
// test can substitute a stub that simulates a real git failure — the fail-closed
// path is otherwise hard to trigger deterministically.
var ignoreChecker = checkIgnoredStrict

// checkIgnoredStrict is the release gate's fail-CLOSED gitignore probe. Unlike
// gitutil.CheckIgnored (fail-OPEN, appropriate for convention checks), it
// distinguishes three outcomes so a secret-hygiene gate never silently ships a
// gitignored file when git could not answer (adr-18 default-deny spirit):
//
//   - root is not a git working tree, or git is absent: no gitignore semantics
//     apply — returns an empty set and nil error, so a plain (non-repo) temp dir
//     still resolves. This is "nothing ignored", not a hard failure.
//   - root IS a repo and git answers: exit 0 lists the ignored subset, exit 1
//     means nothing is ignored — both return nil error.
//   - root IS a repo but the check-ignore probe itself fails (any exit other than
//     0/1, or a spawn/IO error): returns an error so finalize fails closed rather
//     than promote unproven files to Included.
func checkIgnoredStrict(root string, candidates []string) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	if len(candidates) == 0 {
		return out, nil
	}
	// Probe first so a directory that is simply not a repo (or git-absent) carries
	// no gitignore semantics — never a fail-closed rejection. Only a repo whose own
	// check-ignore then errors is a real failure.
	if !gitutil.InRepo(root) {
		return out, nil
	}
	// The same exec pins gitutil.isolatedGit forces (core.hooksPath=/dev/null,
	// core.fsmonitor=false; core.quotePath=false for path fidelity). This probe's
	// --no-index skips the index refresh that consults core.fsmonitor, so the pin
	// is defence-in-depth here rather than the live hole (GHSA-h2gm-w3hm-8xpq) —
	// but a probe pointed at a possibly-hostile clone must not depend on one flag
	// to stay non-executing. A clone's own .git/config is fully trusted by git.
	cmd := exec.Command("git", "-C", root,
		"-c", "core.hooksPath=/dev/null",
		"-c", "core.fsmonitor=false",
		"-c", "core.quotePath=false",
		"-c", "core.excludesFile=",
		"check-ignore", "-z", "--no-index", "-v", "--stdin")
	// Scrub the environment: appending to os.Environ() left GIT_DIR/GIT_WORK_TREE/
	// GIT_CONFIG_* intact, and those override `-C root`, silently redirecting this
	// gitignore probe at a DIFFERENT repository — so a gitignored secret could read
	// as "not ignored" and be promoted into the release bundle.
	cmd.Env = gitutil.IsolatedEnv()
	cmd.Stdin = strings.NewReader(strings.Join(candidates, "\x00") + "\x00")
	data, err := cmd.Output()
	if err != nil {
		// In a repo, exit 1 == no candidate is ignored (a normal answer). Any other
		// exit, or a spawn/IO error, means git could not answer — fail closed.
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return out, nil
		}
		return nil, fmt.Errorf("git check-ignore failed under %s: %w", root, err)
	}
	fields := strings.Split(string(data), "\x00")
	if len(fields) > 0 && fields[len(fields)-1] == "" {
		fields = fields[:len(fields)-1]
	}
	// -v -z emits four fields per record: source, linenum, pattern, pathname.
	for i := 0; i+3 < len(fields); i += 4 {
		pattern := fields[i+2]
		pathname := fields[i+3]
		if strings.HasPrefix(pattern, "!") {
			continue // negation → the path is NOT ignored
		}
		out[pathname] = struct{}{}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Include matching (segment-aware glob, RE2-safe)
// ---------------------------------------------------------------------------

var globMeta = "*?["

func hasGlobMeta(s string) bool { return strings.ContainsAny(s, globMeta) }

func isGlob(pattern string) bool { return hasGlobMeta(pattern) }

func firstSegment(rel string) string {
	if i := strings.Index(rel, "/"); i >= 0 {
		return rel[:i]
	}
	return rel
}

// segmentDenied reports whether one path COMPONENT case-insensitively matches a
// denied namespace. The compare is EqualFold, not an exact map lookup, so a case
// variant (.ABCD, .Git, MEMORY) is denied exactly as its canonical spelling is
// (GHSA-g2v7-wfmv-v28r / #335, the case-fold axis).
func segmentDenied(seg string) bool {
	for denied := range DenyNamespaces {
		if strings.EqualFold(seg, denied) {
			return true
		}
	}
	return false
}

// pathContainsDeniedSegment reports whether ANY component of the repo-relative
// POSIX path rel is a denied namespace. The structural deny (adr-18) binds every
// segment, not only the first: a denied name nested under an included tree
// (docs/.abcd/…, docs/.git/config) is as much a smuggling path as a top-level
// one (GHSA-g2v7-wfmv-v28r, the nested-segment axis), and the compare is
// case-insensitive per segmentDenied.
func pathContainsDeniedSegment(rel string) bool {
	for _, seg := range strings.Split(rel, "/") {
		if segmentDenied(seg) {
			return true
		}
	}
	return false
}

// firstMatchAndMark returns the first include covering rel and marks every glob
// that covers it as matched (so a file also covered by an earlier include does
// not leave a later glob falsely reported unmatched).
func (r *resolver) firstMatchAndMark(rel string) string {
	first := ""
	for _, inc := range r.includes {
		if matchesInclude(rel, inc) {
			if first == "" {
				first = inc
			}
			if isGlob(inc) {
				r.matchedGlobs[inc] = struct{}{}
			}
		}
	}
	return first
}

func (r *resolver) anyIncludeMatches(rel string) bool {
	for _, inc := range r.includes {
		if matchesInclude(rel, inc) {
			return true
		}
	}
	return false
}

// anyLiteralFileInclude reports whether any include is a literal file path equal
// to rel (an explicit ship request for that exact file).
func (r *resolver) anyLiteralFileInclude(rel string) bool {
	for _, inc := range r.includes {
		if !isGlob(inc) && inc == rel {
			return true
		}
	}
	return false
}

// includeMayReachDir reports whether any include could match something at or
// under dir (used to decide whether to deref a symlink dir).
func (r *resolver) includeMayReachDir(dir string) bool {
	for _, inc := range r.includes {
		if matchesInclude(dir, inc) {
			return true
		}
		if !isGlob(inc) && strings.HasPrefix(inc, dir+"/") {
			return true
		}
		if isBroad(inc) {
			return true
		}
		if isGlob(inc) && strings.HasPrefix(inc, dir+"/") {
			return true
		}
	}
	return false
}

// isBroad reports whether an include may reach a denied namespace during the
// walk (a root wildcard, or a first-segment glob).
func isBroad(pattern string) bool {
	switch pattern {
	case ".", "", "**", "*":
		return true
	}
	return isGlob(firstSegment(pattern))
}

func anyReachesScripts(includes []string) bool {
	for _, inc := range includes {
		first := firstSegment(inc)
		if first == "scripts" {
			return true
		}
		switch inc {
		case ".", "", "**", "*":
			return true
		}
		if isGlob(first) {
			if ok, _ := filepath.Match(first, "scripts"); ok {
				return true
			}
		}
	}
	return false
}

// matchesInclude reports whether repo-relative rel is covered by pattern. A
// literal dir pattern covers its whole subtree; a literal file matches exactly.
// Globs are segment-aware: ** spans separators, a single * / ? / [...] never
// crosses /.
func matchesInclude(rel, pattern string) bool {
	if pattern == "." || pattern == "" {
		return true
	}
	if isGlob(pattern) {
		return globMatch(rel, pattern)
	}
	return rel == pattern || strings.HasPrefix(rel, pattern+"/")
}

// globMatch does a segment-aware glob match. A trailing /** also matches the
// base dir and its whole subtree.
func globMatch(rel, pattern string) bool {
	if strings.HasSuffix(pattern, "/**") {
		base := pattern[:len(pattern)-3]
		if base != "" && !isGlob(base) {
			return rel == base || strings.HasPrefix(rel, base+"/")
		}
	}
	re, guards := globToRegexp(pattern)
	m := re.FindStringSubmatch(rel)
	if m == nil {
		return false
	}
	// Guarded groups are positive char classes that must never cross a separator.
	for _, g := range guards {
		if g < len(m) && strings.Contains(m[g], "/") {
			return false
		}
	}
	return true
}

// globRegexpCache memoises compiled globs. It is guarded by globRegexpMu so the
// transport-agnostic core can resolve bundles concurrently without a data race
// (iss-31). A concurrent miss may compile the same pattern twice; that is benign
// (both results are equivalent), so the fast path takes only a read lock.
var (
	globRegexpMu    sync.RWMutex
	globRegexpCache = map[string]compiledGlob{}
)

type compiledGlob struct {
	re     *regexp.Regexp
	guards []int
}

// globToRegexp compiles a glob to an anchored RE2 regex where only ** crosses /.
// RE2 has no lookahead, so the single-segment separator guard is expressed via
// [^/] for * and ?, and (for a positive char class that could otherwise match /)
// via a capturing group whose captured text is post-checked for a / — the guard
// group indices are returned.
func globToRegexp(pattern string) (*regexp.Regexp, []int) {
	globRegexpMu.RLock()
	c, ok := globRegexpCache[pattern]
	globRegexpMu.RUnlock()
	if ok {
		return c.re, c.guards
	}
	expr, guards := buildGlobExpr(pattern)
	re, err := regexp.Compile(expr)
	if err != nil {
		// A malformed glob (e.g. an invalid char-class range like [z-a]) is
		// rejected up front by resolveBundle's preflight validation; if one still
		// reaches here, degrade to a never-matching pattern rather than panicking.
		re, guards = neverMatch, nil
	}
	globRegexpMu.Lock()
	globRegexpCache[pattern] = compiledGlob{re: re, guards: guards}
	globRegexpMu.Unlock()
	return re, guards
}

// neverMatch matches no input at all (a character required after end-of-text):
// the safe fallback when a glob fails to compile.
var neverMatch = regexp.MustCompile(`\z.`)

// validateGlobInclude reports a PreflightError when pattern does not compile to a
// valid RE2 regex (e.g. an invalid char-class range), so the caller can reject it
// gracefully before the walk instead of panicking on first use.
func validateGlobInclude(pattern string) error {
	expr, _ := buildGlobExpr(pattern)
	if _, err := regexp.Compile(expr); err != nil {
		return preflight("include pattern %q is a malformed glob: %v", pattern, err)
	}
	return nil
}

// buildGlobExpr translates a glob into an anchored RE2 pattern string where only
// ** crosses /, returning the guard-group indices for positive char classes. It
// does not compile — globToRegexp/validateGlobInclude compile and cache.
func buildGlobExpr(pattern string) (string, []int) {
	var b strings.Builder
	var guards []int
	group := 0
	b.WriteString(`^`)
	i, n := 0, len(pattern)
	for i < n {
		ch := pattern[i]
		switch {
		case ch == '*':
			if i+1 < n && pattern[i+1] == '*' {
				if i+2 < n && pattern[i+2] == '/' {
					b.WriteString(`(?:.*/)?`)
					i += 3
				} else {
					b.WriteString(`.*`)
					i += 2
				}
				continue
			}
			b.WriteString(`[^/]*`)
			i++
		case ch == '?':
			b.WriteString(`[^/]`)
			i++
		case ch == '[':
			cls, negated, adv, ok := parseCharClass(pattern, i)
			if !ok {
				b.WriteString(`\[`)
				i++
				continue
			}
			if negated {
				// A negated class must also exclude /; add it to the negation.
				b.WriteString(`[^` + cls + `/]`)
			} else {
				// A positive class is wrapped in a capturing group so a match that
				// somehow spans / (e.g. a range crossing 0x2F) can be rejected.
				group++
				guards = append(guards, group)
				b.WriteString(`(` + `[` + cls + `]` + `)`)
			}
			i = adv
		case strings.IndexByte(`.^$+{}()|\`, ch) >= 0:
			b.WriteByte('\\')
			b.WriteByte(ch)
			i++
		default:
			b.WriteByte(ch)
			i++
		}
	}
	b.WriteString(`$`)
	return b.String(), guards
}

// parseCharClass parses a [...] class starting at i, returning its body (without
// the leading ^/! for a negated class), whether it is negated, the index after
// the closing ], and ok=false when unterminated.
func parseCharClass(pattern string, i int) (body string, negated bool, adv int, ok bool) {
	j := i + 1
	n := len(pattern)
	if j < n && (pattern[j] == '!' || pattern[j] == '^') {
		negated = true
		j++
	}
	if j < n && pattern[j] == ']' { // literal ] as first member
		j++
	}
	for j < n && pattern[j] != ']' {
		j++
	}
	if j >= n {
		return "", false, i + 1, false
	}
	start := i + 1
	if negated {
		start = i + 2
	}
	return pattern[start:j], negated, j + 1, true
}

// platformBinaryRe matches the basename of a built abcd binary — the
// `abcd-<goos>-<goarch>` names the release workflow publishes, plus the `.exe`
// spelling, AND the bare `abcd` that `go build ./cmd/abcd` produces. The bare
// name matters most of the three: it is the name the bootstrap's own refusal
// text tells a user to build, and the name the binary runs under inside the
// plugin root, so a control that covered only the cross-compiled spellings had
// its hole at the most likely file. The match is the WHOLE basename: a document
// about an artefact (`docs/abcd-darwin-arm64.md`) is prose, not a binary, and
// denying by substring would take it too.
var platformBinaryRe = regexp.MustCompile(`^abcd(-[a-z0-9]+-[a-z0-9]+)?(\.exe)?$`)

// isPlatformBinaryName reports whether base is a released platform artefact's
// file name.
func isPlatformBinaryName(base string) bool {
	return platformBinaryRe.MatchString(base)
}

// scriptsDenied reports whether a scripts/-tree path matches the closure's own
// default-deny (dev-only dir names / compiled suffixes).
func scriptsDenied(rel string) bool {
	for _, seg := range strings.Split(rel, "/") {
		if _, ok := scriptsDenyDirs[seg]; ok {
			return true
		}
	}
	for _, sfx := range scriptsDenySuffixes {
		if strings.HasSuffix(rel, sfx) {
			return true
		}
	}
	return false
}

func hasControlChar(text string) bool {
	for _, r := range text {
		if r <= 0x1F || r == 0x7F {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Hardlink inode map (fail-closed)
// ---------------------------------------------------------------------------

type inodeMap struct {
	aliases      map[[2]uint64][]string
	deniedInodes map[[2]uint64]struct{}
	uncertain    bool
}

func (m *inodeMap) aliasDenied(dev, ino uint64) bool {
	_, ok := m.deniedInodes[[2]uint64{dev, ino}]
	return ok
}

// buildInodeMap builds the (st_dev, st_ino) alias map over the repo's
// regular-file tree (excluding .git/objects), fail-closed: any Lstat error or
// unwalkable subtree marks the map uncertain so no candidate is admitted on
// unproven alias-safety.
func buildInodeMap(root string) *inodeMap {
	m := &inodeMap{aliases: map[[2]uint64][]string{}, deniedInodes: map[[2]uint64]struct{}{}}
	gitObjects := filepath.Join(root, ".git", "objects")
	var walk func(dir string)
	walk = func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			m.uncertain = true
			return
		}
		for _, e := range entries {
			abs := filepath.Join(dir, e.Name())
			if abs == gitObjects {
				continue // content-addressed blobs, never alias-bearing
			}
			info, err := os.Lstat(abs)
			if err != nil {
				m.uncertain = true
				continue
			}
			if info.IsDir() {
				walk(abs)
				continue
			}
			if !info.Mode().IsRegular() {
				continue
			}
			dev, ino := inodeOf(info)
			key := [2]uint64{dev, ino}
			rel, _ := filepath.Rel(root, abs)
			relSlash := filepath.ToSlash(rel)
			m.aliases[key] = append(m.aliases[key], relSlash)
			if pathContainsDeniedSegment(relSlash) {
				m.deniedInodes[key] = struct{}{}
			}
		}
	}
	walk(root)
	return m
}

func inodeOf(info os.FileInfo) (dev, ino uint64) {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Dev), uint64(st.Ino)
	}
	return 0, 0
}

func sortBundle(b *Bundle) {
	sort.SliceStable(b.Included, func(i, j int) bool { return b.Included[i].LogicalPath < b.Included[j].LogicalPath })
	sort.SliceStable(b.Excluded, func(i, j int) bool {
		if b.Excluded[i].LogicalPath != b.Excluded[j].LogicalPath {
			return b.Excluded[i].LogicalPath < b.Excluded[j].LogicalPath
		}
		return b.Excluded[i].Reason < b.Excluded[j].Reason
	})
	sort.SliceStable(b.Rejected, func(i, j int) bool {
		if b.Rejected[i].LogicalPath != b.Rejected[j].LogicalPath {
			return b.Rejected[i].LogicalPath < b.Rejected[j].LogicalPath
		}
		return b.Rejected[i].Reason < b.Rejected[j].Reason
	})
}

// ExcludedSymlink is a link in the archived tree. An archive carries a link as
// the path it names, not as content, so there is nothing of it to scan, and
// reading through it would scan whatever the working tree's target is instead.
const ExcludedSymlink ExcludedReason = "symlink"

// ArchiveTreeDescription names the tree a non-plugin kind's preview scans, for
// the report line that says which tree was scanned.
const ArchiveTreeDescription = "the tree the release tag would archive (git archive's view of HEAD, export-ignore honoured), minus the record namespace"

// ResolveArchiveBundle is the bundle of a non-plugin artefact kind that declares
// no payload include config (itd-2609150819432059, decision 8): the files an
// archive of HEAD would carry, classified under the same structural deny a
// plugin payload is held to. A path with a denied segment is
// excluded(denied_namespace), exactly as the plugin resolver excludes it; a link
// is excluded(symlink); a control character in a path is rejected, as it is in a
// plugin payload. Every other file is included, read from the working tree the
// way a plugin payload's files are, so an uncommitted edit is what the
// dirty-tree gate reports rather than something this listing hides.
func ResolveArchiveBundle(repoRoot string) (Bundle, error) {
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return Bundle{}, err
	}
	if real, err := filepath.EvalSymlinks(absRoot); err == nil {
		absRoot = real
	}
	entries, err := gitutil.ArchiveTree(absRoot, "HEAD")
	if err != nil {
		return Bundle{}, preflight("the tree the release tag would archive could not be listed: %v", err)
	}
	var b Bundle
	for _, e := range entries {
		switch {
		case hasControlChar(e.Path):
			b.Rejected = append(b.Rejected, RejectedFile{LogicalPath: e.Path, Reason: RejectedControlChar})
		case pathContainsDeniedSegment(e.Path):
			b.Excluded = append(b.Excluded, ExcludedFile{LogicalPath: e.Path, Reason: ExcludedDeniedNamespace})
		case e.Mode == "120000":
			b.Excluded = append(b.Excluded, ExcludedFile{LogicalPath: e.Path, Reason: ExcludedSymlink})
		default:
			b.Included = append(b.Included, IncludedFile{
				LogicalPath: e.Path, ResolvedPath: filepath.Join(absRoot, filepath.FromSlash(e.Path)), GitMode: e.Mode,
			})
		}
	}
	sortBundle(&b)
	return b, nil
}
