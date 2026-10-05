package ahoy

// The trust a status-line binary must earn before the harness is pointed at it
// (iss-2610050556383525, risks 1 and 6). The status line is the one thing in
// the harness's user settings that calls abcd, and it runs on every refresh in
// every session, so the binary it names is held to the bar the plugin's hook
// shims hold a PATH abcd to before they run one (hooks/hooks.json, the
// PreToolUse command): owned by the caller and writable by nobody else, in a
// directory nobody else can write, outside the working tree a project controls,
// and recorded in ~/.abcd.noindex/path-entry by the documented install. One
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
	if !fi.Mode().IsRegular() {
		return false, "it is not a regular file"
	}
	// A link into the cache dies with the cache just the same.
	if inVersionedPluginCache(resolvePath(path)) {
		return false, versionedCacheReason
	}
	dir, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return false, "its directory could not be resolved"
	}
	if insideWorkingTree(dir) {
		return false, "it lives inside the working tree, so the project could replace it"
	}
	if dfi, err := os.Stat(dir); err != nil || dfi.Mode().Perm()&0o002 != 0 {
		return false, "its directory is world-writable"
	}
	if fi.Mode().Perm()&0o002 != 0 {
		return false, "the binary itself is world-writable"
	}
	if err := fsutil.CallersAlone(path, fi); err != nil {
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
		return false, recordShown + " records no abcd install it can vouch for (it is absent, incomplete, or not yours alone)"
	}
	if !sameEntry(rec.path, path) {
		return false, recordShown + " does not record it as the abcd installed here"
	}
	return true, ""
}

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
