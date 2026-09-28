package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// binaryMetaFile is the root-local provenance record the bootstrap's DEGRADED
// per-root fetch still writes (spc-21 behaviour, kept for harnesses that export
// no persistent data dir). Cache-provisioned roots carry none: their provenance
// lives once in the data dir's cache/binary-meta (spc-35).
const binaryMetaFile = ".binary-meta"

// maxBinaryMetaBytes caps the read. A handful of short key=value lines
// (release_tag, release_sha, binary_sha256, fetched_at); anything larger is not
// a file the bootstrap wrote.
const maxBinaryMetaBytes = 4 << 10

// binarySkewNotice is the one-line session-start notice for a plugin surface
// that is newer than the binary serving it, or "" when nothing can be said.
//
// The plugin surface tracks the repository tip while the newest binary is the
// last tagged release, so a fix can merge without a release cut and leave the
// two apart (itd-105). The bootstrap records `unknown` for anything it could not
// resolve, and an unknown commit yields no notice: visibility that guesses is
// worse than none.
//
// The surface side of the comparison is the LIVE plugin root's basename, read
// at render time (spc-35). One cached binary now serves every plugin root, so
// a provisioning-time snapshot of "the root that fetched" would routinely
// describe a root the harness has already replaced and deleted — the very
// update this notice exists to talk about.
func binarySkewNotice() string {
	root := os.Getenv("ABCD_PLUGIN_ROOT")
	if root == "" {
		root = os.Getenv("CLAUDE_PLUGIN_ROOT")
	}
	if root == "" {
		return ""
	}
	meta := readSkewMeta(root)
	pluginSHA, releaseSHA := livePluginSHA(root), meta["release_sha"]
	// Silence here has two distinct causes and they are NOT the same news. Either
	// the two commits genuinely agree (nothing to report), or one of them never
	// parsed — and the live basename failing the 40-hex gate is the interesting
	// one, because it is downstream of itd-105's unverified warrant that the
	// harness names each plugin cache directory for the commit it was cloned
	// from. If that stops holding, this returns "" forever; the directory name
	// itself is the diagnosable evidence, visible to anyone who looks at the
	// live root — which is exactly what this function reads.
	if !resolvedSHA(pluginSHA) || !resolvedSHA(releaseSHA) || pluginSHA == releaseSHA {
		return ""
	}
	// The tag is the one rendered value that is not shape-checked upstream: it is
	// read out of an HTTP redirect. Sanitise it like every other untrusted string
	// this repo renders to a terminal.
	tag := termsafe.Sanitize(meta["release_tag"])
	if tag == "" {
		tag = "unknown"
	}
	// Non-directional on purpose: comparing two commits establishes that they
	// DIFFER, never which is ahead. A plugin root pinned to an older commit than
	// the release the binary was cut from is the same inequality read the other
	// way round, and a notice asserting "the binary is behind" would then be
	// exactly wrong. Name both directions; claim neither.
	return fmt.Sprintf("abcd: the plugin surface is at commit %s and the installed binary is release %s (commit %s) — the two are at different commits, so the surface may be running ahead of the last release, or the binary ahead of this plugin.",
		shortSHA(pluginSHA), tag, shortSHA(releaseSHA))
}

// readSkewMeta reads the provenance record the notice renders from. The
// root-local file wins when it exists: it is written only by the per-root
// fetch, so it describes exactly the binary sitting in THIS root — including a
// migrated pre-cache root whose binary stays put while the shared cache moves
// on to a newer release. Cache-provisioned roots carry no root-local record,
// and for them the shared cache meta (spc-35) is the record of the very
// artefact that was copied in. Unreadable on both paths is a nil map —
// "nothing to say".
func readSkewMeta(root string) map[string]string {
	if meta := readBinaryMeta(filepath.Join(root, binaryMetaFile)); meta != nil {
		return meta
	}
	// The data directory is believed only in the shape the harness gives it,
	// through core's one check: a relative, in-checkout or world-writable one
	// could otherwise fabricate or suppress the notice (iss-2609020630242279).
	if data := os.Getenv("CLAUDE_PLUGIN_DATA"); data != "" {
		cwd, err := os.Getwd()
		if err != nil || ahoy.PluginDataDirHazard(data, cwd) != "" {
			return nil
		}
		return readBinaryMeta(filepath.Join(data, "cache", "binary-meta"))
	}
	return nil
}

// livePluginSHA is the surface commit as the harness names it RIGHT NOW: the
// live plugin root's basename, which the commit-stamped-cache warrant says is
// the commit the root was cloned from. It is deliberately not a recorded value
// — recording it at provisioning time is what spc-35 removed, because the
// recorder's root and the session's root part ways at every plugin update. The
// caller's 40-hex gate (resolvedSHA) decides whether the name is a commit.
func livePluginSHA(root string) string {
	return filepath.Base(root)
}

// readBinaryMeta parses a bootstrap-written key=value file. An unreadable or
// absent file is a nil map, which every caller reads as "nothing to say".
func readBinaryMeta(path string) map[string]string {
	data, err := fsutil.ReadGuarded(path, maxBinaryMetaBytes)
	if err != nil {
		return nil
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), "="); ok {
			out[k] = v
		}
	}
	return out
}

// resolvedSHA reports whether a value actually names a commit, as opposed to
// recording that none was resolved. The SHAPE is the test, not the absence of
// the literal "unknown": the meta is written by a shell script a crash can
// interrupt, and the live basename is whatever the harness called a directory —
// a truncated value is neither a commit nor an admission that there is none,
// and shortSHA would abbreviate half a hash into a notice that reads exactly
// like a real one.
func resolvedSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// shortSHA abbreviates a commit for a one-line notice, leaving anything shorter
// than the abbreviation alone.
func shortSHA(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12]
}
