package ahoy

import (
	"path/filepath"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// abcdBinaryKey is the clone-local git config key the committed commit-msg hook
// finds abcd through, and the only place it looks (defaults/commit-msg).
const abcdBinaryKey = "abcd.binary"

// localAbcdBinary reads this clone's own abcd.binary, "" when it is unset. An
// error is git failing to answer at all — no repository, or a config it cannot
// read — never the key being absent.
func localAbcdBinary(cwd string) (string, error) {
	return gitutil.Run(cwd, "config", "--local", "--default", "", "--get", abcdBinaryKey)
}

// stepOutboundBinary tells the commit-message check where abcd is
// (iss-2610152024267547, the product thinker's "Installer sets it up" of
// 2026-10-15): on the explicit --outbound alone, it writes this clone's
// `git config --local abcd.binary` to the absolute path of the PATH copy whose
// build provenance this run verified. It runs after the PATH-entry steps, so it
// names the entry they actually left.
//
// It points at that copy rather than placing one of its own: the copy is the
// one binary abcd already proves (installOwnedEntry, adoptOwnedCopy), so there
// is no second copy to keep verified. The key is clone-local git config, never
// committed, so it changes nothing for another clone, and nothing for this one
// beyond the opt-in the person just made.
//
// It fails closed, writing nothing and saying so, where the check could not
// trust what it would be pointed at: no copy this run verified (a cold cache, a
// provenance check that failed, a foreign entry), the --dev shim, a relative
// path, an entry that is no longer the copy the provenance record names, and a
// copy inside the working tree, which the hook itself never runs. A value the
// person already set is theirs and is left as it is; it is not printed, since a
// directory name on it can be a private name.
//
// It never writes core.hooksPath. That key selects every hook git runs in the
// clone, not this one, and the name guard and the attribution prompt beside it
// leave it to the person too (hooksPathHint).
func (a *applyCtx) stepOutboundBinary() {
	if !a.outbound {
		return
	}
	const did = "abcd.binary was not written, so the commit-message check warns and lets each commit through until this clone names an abcd"
	target := a.binTarget
	switch {
	case a.devMode:
		a.refuse(did + ": the PATH entry is the --dev shim, which rebuilds abcd from source rather than being a release copy whose build provenance was verified; name an abcd you trust by hand with git config --local abcd.binary /absolute/path/to/abcd")
		return
	case target == "" || !filepath.IsAbs(target):
		a.refuse(did + ": this run has no absolute PATH entry whose provenance it verified to point it at; re-run abcd ahoy install --outbound once the PATH entry is installed")
		return
	case insideRepo(a.cwd, target):
		a.refuse(did + ": the PATH entry " + displayPath(target) + " lies inside this working tree, and the check never runs an abcd the working tree carries; install abcd outside it (omit --bin-dir, or name a directory outside the repository) and re-run abcd ahoy install --outbound")
		return
	case a.verifiedCopy != target || classifyBinTarget(target, a.det.pluginRoot) != binTargetOwnedCopy:
		a.refuse(did + ": this run verified no abcd copy at " + displayPath(target) + " to point it at (the notes above say why); re-run abcd ahoy install --outbound once the PATH entry is a copy whose build provenance verifies")
		return
	}
	cur, err := localAbcdBinary(a.cwd)
	if err != nil {
		a.refuse(did + ": this clone's git config could not be read (" + errText(err) + ")")
		return
	}
	if cur == target {
		return
	}
	if cur != "" {
		a.inform("abcd.binary was left as it is: this clone already names an abcd for the commit-message check, and install does not replace a setting the person made. To point it at the verified copy instead, run git config --local abcd.binary " + displayPath(target))
		return
	}
	if _, err := gitutil.Run(a.cwd, "config", "--local", abcdBinaryKey, target); err != nil {
		a.refuse(did + ": git config could not write it (" + errText(err) + ")")
		return
	}
	a.note(writeOutboundBinary, ".git/config")
}
