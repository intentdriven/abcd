package ahoy

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// dataDirStampFile is the root-local record of the persistent data directory a
// plugin root was provisioned from: one `data_dir=<path>` line that
// hooks/bootstrap.sh writes beside the binary it copies out of the cache. It
// exists because CLAUDE_PLUGIN_DATA reaches hook processes only, while the
// bootstrap's notice sends the reader to a terminal to run `ahoy install` —
// which has the plugin root (it is invoked through it) and nothing else
// (iss-2609012111168716). The stamp is a route to the cache, never a trust
// claim: every promotion out of the cache still re-hashes the artefact against
// the cache's recorded binary_sha256 (adr-46 decision 2).
const dataDirStampFile = ".data-dir"

// dataDirLookup is the outcome of resolving the persistent data dir: the
// directory when a source answered, and in either case the story of what was
// consulted, so a degradation can say which sources it tried rather than
// leaving the reader to guess which one was missing.
type dataDirLookup struct {
	dir   string // the resolved directory, or "" when no source answered
	story string // how the sources answered, in the order they were consulted
}

// explainMissingCache renders the story for the degradation note when the
// resolved directory (if any) holds no verified artefact.
func (l dataDirLookup) explainMissingCache() string {
	if l.dir == "" {
		return l.story
	}
	return l.story + ", which holds no cache artefact with a recorded checksum"
}

// pluginDataDir resolves the harness's persistent per-plugin data directory
// (spc-35): it survives plugin updates and is deleted only on full uninstall,
// which makes it the platform-sanctioned home for the bootstrap's download
// cache. Sources, in order:
//
//  1. CLAUDE_PLUGIN_DATA, which the harness exports to hook processes — taken
//     as given, exactly as the bootstrap takes it.
//  2. The plugin root's .data-dir stamp, which the bootstrap wrote from that
//     very variable when it provisioned the root — the terminal's route. The
//     recorded path is followed only when it is absolute and an existing
//     directory; the artefact it leads to is re-verified by every caller that
//     copies it.
//
// The documented path shape is deliberately never derived from the plugin
// root, and the harness's own configuration is never read: a wrong guess
// would plant a trusted artefact in an untracked location. With no source
// answering, dir is empty and every caller degrades loudly to the pre-cache
// behaviour, naming the story.
func pluginDataDir(pluginRoot string) dataDirLookup {
	if dir := os.Getenv("CLAUDE_PLUGIN_DATA"); dir != "" {
		return dataDirLookup{dir: dir, story: "CLAUDE_PLUGIN_DATA names " + displayPath(dir)}
	}
	story := "CLAUDE_PLUGIN_DATA is unset"
	if pluginRoot == "" {
		return dataDirLookup{story: story + " and no plugin root is resolved, so its " + dataDirStampFile + " record could not be read"}
	}
	recorded := metaField(filepath.Join(pluginRoot, dataDirStampFile), "data_dir")
	if recorded == "" {
		return dataDirLookup{story: story + " and the plugin root " + displayPath(pluginRoot) + " carries no " + dataDirStampFile + " record"}
	}
	story += " and the plugin root's " + dataDirStampFile + " record names " + displayPath(recorded)
	if !filepath.IsAbs(recorded) || !isDir(recorded) {
		return dataDirLookup{story: story + ", which is not an existing absolute directory"}
	}
	return dataDirLookup{dir: recorded, story: story}
}

// metaField reads the first value recorded for key in a bootstrap-written
// key=value record (the cache's binary-meta, a root's .binary-meta or
// .data-dir), or "" when the file is absent, unreadable, over the record cap,
// or carries no such line. One reader for one record shape.
func metaField(path, key string) string {
	data, err := fsutil.ReadGuarded(path, maxPathEntryBytes)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), "="); ok && k == key {
			return v
		}
	}
	return ""
}

// dataDirHazard reports why dataDir cannot be trusted as the harness's
// persistent data directory, or "" when it has the shape that directory always
// has: an absolute path, outside the repository being installed, not
// world-writable. Neither source pluginDataDir consults examines the value it
// hands back — CLAUDE_PLUGIN_DATA is read from the environment as given, and
// the plugin root's .data-dir stamp is checked only for being an existing
// absolute directory — and the owned-copy promotion re-verifies the cache only
// against the record beside it, so a value of any other shape would let
// whoever chose it — a
// relative value resolves against the checkout the verb runs in, an
// in-checkout value is committed bytes, a world-writable cache is any local
// user's — bless their own bytes as the owned PATH binary (sub-finding of
// GHSA-4q78-ccfv-f374). The harness never produces these shapes, so refusing
// them costs a real install nothing; binding the cache to an attestation the
// env cannot supply is the parent record's open decision and is not attempted
// here.
func dataDirHazard(dataDir, cwd string) string {
	if dataDir == "" {
		return ""
	}
	if !filepath.IsAbs(dataDir) {
		return "it is a relative path, which resolves against whatever directory the verb happens to run in"
	}
	if abs, err := filepath.Abs(cwd); err == nil && under(resolvePath(abs), resolvePath(dataDir)) {
		return "it lies inside the repository being installed, so its cache would be committed bytes"
	}
	for _, dir := range []string{dataDir, filepath.Join(dataDir, "cache")} {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() && fi.Mode().Perm()&0o002 != 0 {
			return "it is world-writable (" + displayPath(dir) + "), so any local user could replace both the artefact and its recorded hash"
		}
	}
	return ""
}
