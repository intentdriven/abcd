package cli

// supersededroot.go — the SILENT third shape of a binary/surface skew
// (iss-2609020113012227, refining iss-2608230943088357).
//
// staleusage.go covers the two LOUD shapes: an unknown flag and an unknown
// command, both of which give the binary an error to hang a note on. The third
// shape has no error to instrument. The verb exists, the old binary serves it,
// and it answers normally — exit 0, no diagnostic of any kind — while reporting
// something that is false of this machine. It was observed on the version
// report immediately after a release published: the page said v0.6.8 on a machine where
// v0.7.0 had just been verified.
//
// The mechanism is the path, not the binary. A plugin root is named for the
// COMMIT it was installed from, so every update mints a new root and nothing
// prunes the old ones — six had accumulated on the machine where this was found,
// four still holding their own ~18MB binary. A command page interpolates an
// absolute, hash-pinned path into its own prose, and that path is therefore
// designed to expire: between an update and the reload that re-interpolates it,
// the page names a root that is still on disk and still answers.
//
// What the disk can prove, with no network (adr-38) and no heuristic, is the
// divergence itself: the plugin root THIS SESSION resolves — through the same
// ladder every other surface uses, which prefers the environment's own
// CLAUDE_PLUGIN_ROOT over the executable's ancestors — against the plugin root
// the RUNNING BINARY sits in. When they are two different roots, the answer came
// from a root this session does not serve, and that is a fact worth stating
// beside the answer.
//
// It is a NOTE, never a refusal and never a different answer: the reported
// version, vintage and staleness are unchanged, exactly as staleusage.go leaves
// cobra's own line, exit code and JSON envelope untouched. The two surfaces that
// carry it are the two whose job is reporting which abcd is installed — the
// `--version` report (which `update --check` extends) and `ahoy`'s bare render,
// which renders the same vintage/staleness pair through the same comparator.
// Every other verb answers a question about the repository rather than about
// itself, so a note there would be noise on every invocation.

import (
	"fmt"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// supersededRootNote is the sentence a version-reporting surface adds when the
// running binary is served from a plugin root other than the one this session
// resolves, or "" when the disk proves nothing.
//
// The roots are named by their directory name — the commit the root was
// installed from, which is the whole of what identifies one — rather than by
// their absolute path: the name is the identifying token, and a full path drags
// the machine's home directory into a rendered line for no added information.
func supersededRootNote() string {
	session, ok := ahoy.ResolvePluginRoot()
	if !ok {
		return "" // no plugin root resolves: there is no divergence to report
	}
	if executableUnder(session) {
		return "" // the running binary IS the one this session serves
	}
	exe := executablePath()
	if exe == "" {
		return "" // the running binary's path is unknown: nothing to compare
	}
	serving, ok := ahoy.PluginRootContaining(exe)
	if !ok {
		// A PATH copy, a source build, a `go run` binary: outside every plugin
		// root, so there is no superseded ROOT to name. Staleness of a PATH copy
		// is the version report's existing vintage comparison, not this.
		return ""
	}
	if sameRoot(serving, session) {
		return ""
	}
	if isSourceCheckout(serving) {
		// A SOURCE CHECKOUT is a valid plugin root — hooks/ sits at the top of
		// this very repository — so a developer running the `make build`
		// artefact from the checkout while a harness session resolves its own
		// cache root satisfies the divergence test. Both halves of the note
		// would be false there: the checkout is not named for a commit it was
		// installed from, and "re-run through this session's own plugin root"
		// points at the binary AGENTS.md's dogfooding rule calls the stale one.
		// A source build's currency is the vintage comparison's business, which
		// the version report already renders. staleUsageNote keys its rebuild
		// remedy on the same isSourceCheckout test, for the same reason.
		return ""
	}
	return fmt.Sprintf(
		"this answer comes from the plugin root %s, but this session's plugin root is %s — "+
			"a plugin root is named for the commit it was installed from, so a binary path pinned in a "+
			"command page outlives the root it names and the superseded root stays on disk, still "+
			"answering. Re-run through this session's own plugin root before trusting this answer",
		termsafe.Sanitize(filepath.Base(serving)), termsafe.Sanitize(filepath.Base(session)))
}

// executablePath is the running binary's path through the same seam
// executableUnder reads, so a test that places the binary places it once.
func executablePath() string {
	exe, err := osExecutable()
	if err != nil {
		return ""
	}
	return exe
}

// sameRoot compares two plugin roots by their canonical form, so a symlinked
// route to one root and the root itself are not reported as a divergence.
func sameRoot(a, b string) bool {
	return canonicalPath(a) == canonicalPath(b)
}
