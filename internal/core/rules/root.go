package rules

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// TrustedRootsRelPath is the home-scoped opt-in that re-admits a marker root the
// caller does not own, relative to the user's home directory.
const TrustedRootsRelPath = ".abcd/trusted-roots"

// TrustedRootsDisplay is how that file is NAMED in a diagnostic: the tilde form,
// never the expanded path, so a refusal a user pastes into a shell works and no
// message carries the developer-identity home path (iss-81, fsutil.RedactHome).
const TrustedRootsDisplay = "~/" + TrustedRootsRelPath

// maxTrustedRootsBytes caps the declaration read. A hand-maintained list of
// checkout paths is a handful of lines; 64 KiB bounds a planted device or an
// endless file without ever refusing a real one.
const maxTrustedRootsBytes = 64 << 10

// ownerUID is the package's view of fsutil.OwnerUID, held as a var for the same
// reason fsutil keeps caseFoldingFS: the foreign-owner branch cannot be provoked
// on a host where the test process can create only its own files, so
// substituting the lookup is the only way a detector can prove the refusal — and,
// with it left alone, that a root the caller really owns is admitted unchanged.
var ownerUID = fsutil.OwnerUID

// ResolveRoot resolves the repo root the per-repo configuration under .abcd/ is
// read from — rules.json and config.json here, guard.json for the shell guard,
// which shares this resolver so a session's rules and its guard can never come
// from two different places.
//
// The resolution is bounded at the git working tree (GHSA-vvqc-3mv2-5p49). The
// toplevel git reports for cwd is resolved FIRST; the walk then runs from cwd
// upward but stops at that toplevel inclusive, so the nearest .abcd directory
// INSIDE the tree wins (a repo that disabled a domain, or the whole loader,
// stays disabled from any nested directory — iss-66/B12) and a tree with no
// .abcd resolves to its own toplevel. A .abcd above the working tree is never
// reached: an unbounded walk let a directory planted in the shared temp dir
// govern the injected rules and disarm the guard of every repository beneath
// it. The per-repo file is a repo-scope setting by design (the configuration
// chapter and itd-3 both reject a global rules.json), so the bound is the
// documented contract, not a new policy.
//
// The exact scope of what that closes: any ancestor a session's own working
// tree sits inside; a plant in a shared directory that is not a repository —
// the one-command `: > /tmp/.git` or `mkdir /tmp/.git` beside a planted
// /tmp/.abcd, which the fallback below refuses because a marker must look like
// a repository; and a REAL repository planted in a shared ancestor by another
// uid (`git init /tmp`, or a repository laid in a root-owned mode-1777
// directory), which the fallback refuses on OWNERSHIP — see
// foreignOwnerRefusal, and TrustedRootsRelPath for the explicit opt-in that
// re-admits a foreign-uid checkout the caller means to trust. One residual
// stays open, recorded rather than silently assumed shut:
//
//   - iss-2609020219198779 — the user-scope ~/.abcd when the home directory is
//     ITSELF a git working tree (dotfiles-in-home). The toplevel for a session
//     in a non-repo directory beneath such a home is the home, so ~/.abcd
//     governs it as the REPO layer as well as the user layer (spc-23), and its
//     guard.json and config.json with it. Closing it needs a decision on
//     whether a home-directory toplevel is a legitimate repo-scope root.
//
// "Not a repository" and "a repository git will not answer for" are DIFFERENT
// outcomes and only the first resolves to cwd with no walk. abcd runs git under
// an isolated environment (GIT_CONFIG_GLOBAL=/dev/null, GIT_CONFIG_NOSYSTEM=1)
// which also discards the developer's safe.directory exceptions, so rev-parse
// fails inside a checkout owned by another uid; it fails outright when the host
// launches the hook with git off its PATH. Collapsing those onto cwd dropped
// the repository's own configuration layer with no error anywhere: from a
// subdirectory the root became the subdirectory, so guard.json's hazards
// became allows, the rules kill switch stopped applying, and the private
// banlist store went missing. A repo-shaped tree therefore falls back to the
// .git marker (gitutil.RepoShapedRoot), and only a directory with no repository
// above it keeps the no-walk cwd: outside a working tree there is no boundary
// to stop at, and a walk that stops nowhere is the defect.
//
// The marker walk is a NAME check that runs to the filesystem root, so it is
// not by itself a bound: an unprivileged `: > /tmp/.git` would otherwise hand
// every session in a plain directory beneath /tmp to a planted /tmp/.abcd, git
// having refused to answer for a directory that is not a repository. The
// fallback root is accepted only when its .git is a PLAUSIBLE repository — a
// directory carrying HEAD, or a regular file beginning "gitdir: "
// (plausibleRepository below, which states the shapes exactly) — and anything
// else takes the non-repo route: cwd, no walk, nothing above it consulted. That
// discriminator is what git reads, not a trust boundary on its own: laying out a
// real repository still passes it. The trust boundary is the ownership gate that
// runs next — a marker root whose owner is not the caller is refused, loudly,
// unless the caller has declared it in the user-scope home
// (iss-2609020259564193). Both bounds apply only where git will not answer;
// where it does, the toplevel it named stands whoever owns it.
//
// A nested repository or submodule resolves its own toplevel either way and
// does not inherit the superproject's .abcd — correct scoping, visible to
// submodule workflows.
func ResolveRoot(cwd string) string { return Resolve(cwd).Root }

// Resolution is ResolveRoot's answer together with the diagnostics the
// resolution produced. Notes is empty in the ordinary case; a non-empty entry is
// a refusal or a downgrade a front door MUST print out of band (stderr, never
// the injected context), because a resolution that silently declined to read a
// repository's own configuration is the exact shape the bound exists to prevent
// (.abcd/development/principles/loud-staging.md).
type Resolution struct {
	Root  string
	Notes []string
}

// Resolve is ResolveRoot with its diagnostics kept. ResolveRoot is the shorthand
// for the callers that have nowhere to print them.
func Resolve(cwd string) Resolution {
	// git reports the physical path; resolve cwd the same way so the ancestor
	// chain from cwd actually passes through top (macOS /var -> /private/var).
	dir, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		dir = filepath.Clean(cwd)
	}
	top, err := gitutil.Run(cwd, "rev-parse", "--show-toplevel")
	if err != nil || top == "" {
		// The marker walk runs on the symlink-resolved path, so the bound it
		// returns is already on the same chain the walk below climbs.
		marker := gitutil.RepoShapedRoot(dir)
		if marker == "" || !plausibleRepository(marker) {
			return Resolution{Root: cwd} // no repository to bound at: no boundary, no walk.
		}
		if refusal := foreignOwnerRefusal(marker, cwd); len(refusal) > 0 {
			// Fail closed on the same route a non-repository takes: cwd, no
			// walk, nothing above it consulted — the refusal is only loud
			// because it also declines to read a checkout that may well be the
			// caller's own work.
			return Resolution{Root: cwd, Notes: refusal}
		}
		top = marker
	}
	if real, err := filepath.EvalSymlinks(top); err == nil {
		top = real
	}
	for inside(dir, top) {
		if fi, err := os.Stat(filepath.Join(dir, ".abcd")); err == nil && fi.IsDir() {
			return Resolution{Root: dir}
		}
		if dir == top {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return Resolution{Root: top}
}

// inside reports whether dir is top or lies beneath it.
func inside(dir, top string) bool {
	if dir == top {
		return true
	}
	prefix := top
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return strings.HasPrefix(dir, prefix)
}

// plausibleRepository reports whether root's .git entry has the shape of a
// repository rather than merely carrying the name. It is the discriminator
// ResolveRoot applies to gitutil.RepoShapedRoot's answer; the check lives here
// and not there because RepoShapedRoot's other callers want the crude marker
// (they ask "could anything here be committed?", where a name is enough).
//
// Two shapes pass, the two git itself reads: a .git DIRECTORY carrying HEAD (an
// ordinary checkout — HEAD is the first file git looks for, and no repository
// lacks it), and a regular .git FILE beginning "gitdir: " (a linked worktree or
// a submodule, whose working-tree root is the directory that file sits in).
// Everything else is not a repository for this purpose: an empty file, an empty
// or HEAD-less directory, a dangling symlink, a socket or a device. The .git
// path is stat'd, not lstat'd, so a symlink to a real repository still passes
// and a dangling one does not.
//
// Only the first eight bytes of a .git file are read, so a large file planted
// under the name cannot make the resolver buffer it.
func plausibleRepository(root string) bool {
	marker := filepath.Join(root, ".git")
	fi, err := os.Stat(marker)
	if err != nil {
		return false
	}
	if fi.IsDir() {
		_, err := os.Stat(filepath.Join(marker, "HEAD"))
		return err == nil
	}
	if !fi.Mode().IsRegular() {
		return false
	}
	f, err := os.Open(marker)
	if err != nil {
		return false
	}
	defer f.Close()
	var head [len("gitdir: ")]byte
	if _, err := io.ReadFull(f, head[:]); err != nil {
		return false
	}
	return string(head[:]) == "gitdir: "
}

// foreignOwnerRefusal reports why marker must not govern a session at cwd, as
// the diagnostics a front door prints, or nil when it may. It runs ONLY on the
// git-refused fallback: where git answers, the toplevel it names is a repository
// git itself vouched for, and narrowing that would break every legitimate
// foreign-uid checkout on a host whose git is configured for one.
//
// The policy (iss-2609020259564193). plausibleRepository stops the one-command
// plant — `: > /tmp/.git` — but not a real repository: `git init` in a shared
// directory an unprivileged user can write produces a genuine checkout, and
// git's refusal on ownership under abcd's isolated environment
// (GIT_CONFIG_GLOBAL=/dev/null discards safe.directory) is the SAME signal in
// that attack as in the case the fallback exists for. No property of the tree
// can separate them, because the two trees are identical; the only thing that
// differs is whether the caller MEANT to trust it. So a marker root the caller
// does not own is refused by default and re-admitted only by an explicit
// declaration the caller made — which is the one bit of information the tree
// cannot supply about itself.
//
// An unreadable owner is refused too: "I could not learn who owns this" and "I
// own this" are different answers, and a fail-closed gate must not spell them
// the same way.
func foreignOwnerRefusal(marker, cwd string) []string {
	caller := uint32(os.Getuid())
	owner, err := ownerUID(marker)
	if err == nil && owner == caller {
		return nil
	}
	admitted, note := trustedRootDeclared(marker)
	if admitted {
		return nil
	}
	var notes []string
	if note != "" {
		notes = append(notes, note)
	}
	because := fmt.Sprintf("it is owned by uid %d, not by this session's uid %d", owner, caller)
	if err != nil {
		because = "its owning uid could not be read (" + termsafe.Sanitize(err.Error()) + ")"
	}
	return append(notes, fmt.Sprintf(
		"rules: REFUSED %s as this session's configuration root — %s, and git would not answer for it, "+
			"so %s and .abcd/guard.json there were NOT read and nothing above %s governs this session "+
			"(injected rules and the loader kill switch fall back to the bundled defaults under the user scope's %s, the hazard registry to the bundled defaults). "+
			"If that checkout really is yours to trust — a foreign-uid checkout, a container bind mount, a shared CI checkout — "+
			"declare it once, from an account you control: mkdir -p ~/.abcd && printf '%%s\\n' '%s' >> %s",
		termsafe.Sanitize(marker), because, RepoRelPath, termsafe.Sanitize(cwd), UserDisplayPath,
		termsafe.Sanitize(marker), TrustedRootsDisplay))
}

// trustedRootDeclared reports whether the caller has declared marker trustworthy
// in the user-scope home, and — when a declaration exists but was not honoured —
// why, so an opt-in that silently did nothing is never mistaken for one that was
// never written.
//
// The declaration is home-scoped and NOT an environment variable, deliberately.
// The bar the opt-in has to clear is that the untrusted tree cannot assert it:
// a `.abcd/` file inside the foreign root declaring itself trusted would be
// circular, so the declaration has to come from somewhere the caller controls.
// An env var clears the letter of that bar and not its spirit — a repository can
// ship the shell, direnv or task-runner configuration that sets the variable,
// and an operator whose shell auto-loads it has the tree declaring itself
// trusted one indirection out. abcd has already ruled on this shape once:
// dataDirHazard refuses an env-supplied data directory for the same reason, and
// GHSA-4q78-ccfv-f374's recorded remedy is to move the trust floor "from env to
// home write", which adr-46 decision 4 already treats as the ownership root.
// Writing ~/.abcd/trusted-roots needs write access to the caller's own home —
// the same authority the caller already holds over everything abcd trusts, so
// the opt-in grants an attacker nothing they did not already have.
//
// It follows the ~/.abcd/path-entry idiom rather than inventing one: a
// home-scoped, abcd-owned, line-oriented record, read through the guarded
// bounded read, where an absent or unvouched-for record vouches for nothing. It
// is a SIBLING file rather than a section of path-entry because path-entry
// records exactly one thing (the owned PATH copy's path and hash) and is parsed
// by ahoy; a second record family in it would couple this resolver to that
// parser.
//
// A declaration this process does not own, or one anyone can write, is not the
// caller's word and re-admits nothing.
func trustedRootDeclared(marker string) (bool, string) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return false, ""
	}
	path := filepath.Join(home, filepath.FromSlash(TrustedRootsRelPath))
	// The three-part guard is fsutil.ReadDeclaration's, not this function's: the
	// three home-scoped declaration records differ in what they declare, never in
	// what makes a declaration trustworthy, and the copy that skipped two of the
	// checks was the one whose consequence is code execution
	// (iss-2609091927085132). Only the WORDING stays here.
	raw, refusal, err := fsutil.ReadDeclaration(path, maxTrustedRootsBytes)
	switch refusal {
	case fsutil.DeclarationOK:
	case fsutil.DeclarationAbsent:
		return false, "" // no declaration is the ordinary case, not a diagnostic.
	case fsutil.DeclarationNotRegular:
		return false, ignoredDeclaration("it is not a regular file")
	case fsutil.DeclarationWritableByOthers:
		return false, ignoredDeclaration("it is writable by others, so its contents are not necessarily yours")
	case fsutil.DeclarationForeignOwner:
		return false, ignoredDeclaration("it is not owned by this session's uid")
	default:
		return false, ignoredDeclaration("it could not be read (" + termsafe.Sanitize(err.Error()) + ")")
	}
	fold := fsutil.CaseFoldingFS()
	want := fsutil.FoldPath(marker, fold)
	for _, line := range strings.Split(string(raw), "\n") {
		entry := strings.TrimSpace(line)
		if entry == "" || strings.HasPrefix(entry, "#") {
			continue
		}
		if !filepath.IsAbs(entry) {
			continue // a relative entry names a different directory per caller.
		}
		// Both spellings: the entry as written, and symlink-resolved, because
		// marker is already resolved and a declared path may not be.
		for _, cand := range []string{filepath.Clean(entry), resolveOrClean(entry)} {
			if fsutil.FoldPath(cand, fold) == want {
				return true, ""
			}
		}
	}
	return false, ""
}

// ignoredDeclaration renders the one-line reason a present declaration was not
// honoured. It names the file in tilde form so the message carries no
// developer-identity home path.
func ignoredDeclaration(why string) string {
	return "rules: IGNORED " + TrustedRootsDisplay + " — " + why + "; it re-admitted nothing"
}

// resolveOrClean is EvalSymlinks with a lexical fallback, so a declared path
// that does not exist (a bind mount not currently mounted) still compares.
func resolveOrClean(p string) string {
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	return filepath.Clean(p)
}
