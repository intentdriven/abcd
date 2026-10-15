package ahoy

// The trust a status-line binary must earn before the harness is pointed at it
// (iss-2610050556383525, risks 1 and 6). The status line is the one thing in
// the harness's user settings that calls abcd, and it runs on every refresh in
// every session, so the binary it names is held to the bar the plugin's hook
// shims hold a PATH abcd to before they run one (hooks/hooks.json, the
// PreToolUse command): owned by the caller and writable by nobody else, in a
// directory writable by nobody else (group included, which goes past the shim's
// other-write test) and owned by the caller or root, outside the working tree
// a project controls, and recorded in ~/.abcd.noindex/path-entry by the
// documented install. One
// check is added that a hook never needs: a binary inside the plugin cache's
// VERSIONED directory (<harness-home>/plugins/cache/<marketplace>/abcd/
// <version>/abcd) is refused, because the next plugin update deletes that
// directory and leaves the line dangling in every repository.
//
// It is read-only and cheap — a handful of stats and the one bounded
// path-entry read — so detection, the session-start notice and the wiring step
// can all ask it.

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// statusLineEntryTrust reports whether path may be the binary the harness's
// status line runs and, when it may not, why, in words a person can act on.
// The reason never carries the home path. Checks run cheapest and most
// specific first, so the reason given is the one that names the real problem:
// a binary in the versioned cache is "in the versioned directory", not merely
// "not recorded".
func statusLineEntryTrust(path string) (ok bool, reason string) {
	if path == "" || !filepath.IsAbs(path) {
		return false, "it is not an absolute path, and the harness runs the status line without your shell's PATH"
	}
	if inVersionedPluginCache(path) {
		return false, versionedCacheReason
	}
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "it does not exist"
		}
		return false, "it could not be examined (" + errText(err) + ")"
	}
	// The recorded entry may be a link to the binary (an owned symlink is one
	// of the install shapes statusLineEntry admits), and what the line runs is
	// the end of its chain: every binary check below judges the target, and the
	// directory checks judge the directory of EVERY link on the way, not only
	// the first and the last, since whoever can write any one of them can
	// repoint the link it holds and choose what runs (iss-2610050844241683).
	target, dirs, reason := statusLineChain(path)
	if reason != "" {
		return false, reason
	}
	if target != filepath.Clean(path) {
		if fi, err = os.Stat(target); err != nil {
			return false, "it could not be examined (" + errText(err) + ")"
		}
	}
	if !fi.Mode().IsRegular() {
		return false, "it is not a regular file"
	}
	for _, dir := range dirs {
		if insideWorkingTree(dir) {
			return false, "it lives inside the working tree, so the project could replace it"
		}
		if reason := statusLineDirRefusal(dir); reason != "" {
			return false, reason
		}
	}
	if fi.Mode().Perm()&0o002 != 0 {
		return false, "the binary itself is world-writable"
	}
	if err := fsutil.CallersAlone(target, fi); err != nil {
		if errors.Is(err, fsutil.ErrDeclarationWritable) {
			return false, "it is writable by your group, so another account could replace it"
		}
		return false, "it is not owned by you"
	}
	recordShown := abcdhome.Display("path-entry")
	if userPathEntryPath() == "" {
		_, refused := homeScope()
		return false, recordShown + " is not read (" + refused + ")"
	}
	rec, recorded := readPathEntry()
	if !recorded {
		return false, recordShown + " records no abcd install (it is absent, incomplete, or not yours alone)"
	}
	if !sameEntry(rec.path, path) {
		return false, recordShown + " does not record it as the abcd installed here"
	}
	return true, ""
}

// statusLineChain walks the links from path to the file the status line would
// run, one hop at a time with os.Readlink, and returns that file and the
// resolved directory of every hop (the entry's own first, the target's last,
// each once). Each hop's directory is resolved with EvalSymlinks, as the
// entry's own always was, so a hop is judged where it really sits. A hop into
// the plugin cache's versioned directory refuses, since the chain dies with
// it, and so does a chain of more than maxStatusLineLinks links. The reason is
// "" when the walk completes.
func statusLineChain(path string) (target string, dirs []string, reason string) {
	cur := filepath.Clean(path)
	for links := 0; ; links++ {
		if inVersionedPluginCache(cur) {
			return "", nil, versionedCacheReason
		}
		dir, err := filepath.EvalSymlinks(filepath.Dir(cur))
		if err != nil {
			return "", nil, "its directory could not be resolved"
		}
		if !slices.Contains(dirs, dir) {
			dirs = append(dirs, dir)
		}
		cur = filepath.Join(dir, filepath.Base(cur))
		if inVersionedPluginCache(cur) {
			return "", nil, versionedCacheReason
		}
		lfi, err := os.Lstat(cur)
		if err != nil {
			return "", nil, "it could not be resolved (" + errText(err) + ")"
		}
		if lfi.Mode()&os.ModeSymlink == 0 {
			return cur, dirs, ""
		}
		if links == maxStatusLineLinks {
			return "", nil, "it is reached through more than " + strconv.Itoa(maxStatusLineLinks) + " links; point the status line at the binary or at one link to it"
		}
		dest, err := os.Readlink(cur)
		if err != nil {
			return "", nil, "it could not be resolved (" + errText(err) + ")"
		}
		if !filepath.IsAbs(dest) {
			dest = filepath.Join(dir, dest)
		}
		cur = filepath.Clean(dest)
	}
}

// statusLineDirRefusal judges one directory the entry is reached through by the
// standard the binary itself meets — no group or other write bit — owned by
// the caller or by root (fsutil.CallersOrRootsAlone), and returns why it fails,
// remedy included, or "".
func statusLineDirRefusal(dir string) string {
	dfi, err := os.Stat(dir)
	if err != nil {
		return "its directory could not be examined (" + errText(err) + ")" + dirRemedy
	}
	switch err := fsutil.CallersOrRootsAlone(dir, dfi); {
	case err == nil:
		return ""
	case !errors.Is(err, fsutil.ErrDeclarationWritable):
		return "its directory is owned by another account, so that account could replace the binary" + dirRemedy
	case dfi.Mode().Perm()&0o002 != 0:
		return "its directory is world-writable, so any account could replace the binary" + dirRemedy
	default:
		return "its directory is writable by its group, so another account could replace the binary" + dirRemedy
	}
}

// dirRemedy ends every directory refusal: the documented install puts abcd in
// a directory that passes.
const dirRemedy = "; install abcd to ~/.local/bin with `abcd ahoy install`, or, if the directory is yours, `chmod go-w` it"

// versionedCacheReason is the refusal for a binary the next plugin update
// deletes.
const versionedCacheReason = "it lives in the plugin cache's versioned directory, which the next plugin update deletes"

// inVersionedPluginCache reports whether p sits inside the harness's plugin
// cache at or below a version directory: …/plugins/cache/<marketplace>/
// <plugin>/<version>/…. The match is on path components, so it holds for any
// harness home ($CLAUDE_CONFIG_DIR included) and never for a name that merely
// contains the words.
func inVersionedPluginCache(p string) bool {
	parts := strings.Split(filepath.Clean(p), string(filepath.Separator))
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "plugins" && parts[i+1] == "cache" {
			// marketplace, plugin, version, then at least the binary itself.
			return len(parts)-(i+2) >= 4
		}
	}
	return false
}

// insideWorkingTree reports whether dir (already resolved) lies inside the
// working tree the caller stands in: the nearest checkout at or above the
// working directory, found by its .git marker without running git. A working
// directory in no checkout has no tree to be inside, and a checkout that IS the
// home folder (a dotfiles repository) is not a project that could plant a
// binary in ~/.local/bin — so neither refuses the documented install.
func insideWorkingTree(dir string) bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}
	tree := gitutil.RepoShapedRoot(cwd)
	if tree == "" {
		return false
	}
	if home := userHome(); home != "" && resolvePath(tree) == resolvePath(home) {
		return false
	}
	return under(resolvePath(tree), dir)
}

// maxStatusLineLinks bounds the links a status-line entry may be reached
// through. Neither install shape abcd writes is a chain (a copy, or one link to
// the pinned plugin binary), and a package manager adds one more at most
// (~/.local/bin/abcd to /opt/homebrew/bin/abcd to its cellar), so four leaves
// room for every honest shape while keeping the walk short.
const maxStatusLineLinks = 4
