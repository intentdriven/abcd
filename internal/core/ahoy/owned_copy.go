package ahoy

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The PATH entry as an abcd-owned regular file (spc-35). A symlink into the
// plugin root dies at every plugin update — the harness re-clones into a fresh
// commit-stamped directory and garbage-collects the old one — so the entry is
// a COPY of the verified release artefact kept in the persistent data dir's
// cache, and ownership is RECORDED PROVENANCE: the data dir's path-entry file
// names the installed path and its SHA-256. A file that matches the record is
// ours to refresh or remove; anything else is foreign and never touched.
// Content-guessing (recognising "a binary that looks like abcd") is
// deliberately not attempted anywhere.

// maxBinaryArtefactBytes caps reads of the release binary for hashing and
// copying. The artefact is ~11 MB today; 64 MiB bounds a planted device or
// endless file without ever refusing a legitimate release.
const maxBinaryArtefactBytes = 64 << 20

// maxPathEntryBytes caps the provenance-record read: two short key=value lines.
const maxPathEntryBytes = 4 << 10

// cacheAssetPath is the verified release artefact for this platform inside the
// persistent data dir — the same name and layout hooks/bootstrap.sh writes.
func cacheAssetPath(dataDir string) string {
	return filepath.Join(dataDir, "cache", "abcd-"+runtime.GOOS+"-"+runtime.GOARCH)
}

// cacheMetaPath is the cache's provenance record (release_tag / release_sha /
// binary_sha256 / fetched_at), written by the bootstrap.
func cacheMetaPath(dataDir string) string {
	return filepath.Join(dataDir, "cache", "binary-meta")
}

// userPathEntryPath is the PATH-copy provenance record, home-scoped and
// abcd-owned (~/.abcd/path-entry, alongside the history store). It deliberately
// does NOT live in the harness data dir: CLAUDE_PLUGIN_DATA is exported only to
// hook processes, yet `ahoy install`, `ahoy uninstall`, and `abcd update` all
// run from a terminal where it is unset — so a record readable only from a hook
// could not establish ownership exactly where those verbs run, and would
// silently reclassify abcd's own binary as foreign (iss-2608210934566230,
// adr-46 decision 4). The data dir stays the CACHE's home only. Empty when the
// home directory cannot be resolved (every caller then reads "no record").
func userPathEntryPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".abcd", "path-entry")
}

// cacheRecordedSHA reads the cache meta's binary_sha256, or "" when the record
// is absent, unreadable, or not a full lowercase hex digest — a promotion can
// only re-verify against a hash that actually parses.
func cacheRecordedSHA(dataDir string) string {
	if v := metaField(cacheMetaPath(dataDir), "binary_sha256"); hexDigestOK(v) {
		return v
	}
	return ""
}

// hexDigestOK reports whether s is a full lowercase-hex SHA-256.
func hexDigestOK(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// pathEntryRecord is the parsed provenance of the owned PATH copy.
type pathEntryRecord struct {
	path       string // where the copy was installed
	sha        string // its SHA-256 at install/refresh time
	pluginRoot string // the plugin root at install/refresh time (may be empty)
}

// readPathEntry loads the provenance record, reporting ok only when both
// required fields are present and the hash parses — a truncated record vouches
// for nothing. plugin_root is optional (a legacy record predating it, or a
// degraded install, carries none); its absence never fails the read.
//
// It reads through fsutil.ReadDeclaration, the shared home-scoped declaration
// read, rather than the bare guarded read: this record decides which binary the
// hook shims EXECUTE, so a copy of it that group or other can write, or that
// another uid owns, is not this session's word and vouches for nothing — the same
// bar ~/.abcd/trusted-roots and ~/.abcd/local-transcript-roots are held to. An
// unowned record reports not-ok exactly as a truncated one does
// (iss-2609091927085132); that is NOT the accepted same-uid residual
// (iss-2609012039107700), which this check neither closes nor claims to.
func readPathEntry() (pathEntryRecord, bool) {
	path := userPathEntryPath()
	if path == "" {
		return pathEntryRecord{}, false
	}
	raw, _, err := fsutil.ReadDeclaration(path, maxPathEntryBytes)
	if err != nil {
		return pathEntryRecord{}, false
	}
	var rec pathEntryRecord
	for _, line := range strings.Split(string(raw), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch k {
		case "path":
			rec.path = v
		case "binary_sha256":
			rec.sha = v
		case "plugin_root":
			rec.pluginRoot = v
		}
	}
	if rec.path == "" || !hexDigestOK(rec.sha) {
		return pathEntryRecord{}, false
	}
	return rec, true
}

// writePathEntry records (atomically) that the file at target with the given
// hash is abcd's owned PATH copy, installed from pluginRoot. pluginRoot is the
// route home the old PATH symlink used to provide (its target sat inside the
// root); a regular-file copy severs that, so the record carries it and
// resolvePluginRoot reads it as a candidate. An empty pluginRoot records no
// such line — a degraded install has no root to record.
func writePathEntry(target, shaHex, pluginRoot string) error {
	path := userPathEntryPath()
	if path == "" {
		return os.ErrNotExist
	}
	body := "path=" + target + "\nbinary_sha256=" + shaHex + "\n"
	if pluginRoot != "" {
		body += "plugin_root=" + pluginRoot + "\n"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, []byte(body), 0o644)
}

// removePathEntry drops the provenance record; absent is fine.
func removePathEntry() {
	if path := userPathEntryPath(); path != "" {
		_ = os.Remove(path)
	}
}

// pathEntryNames reports whether the provenance record names target. It is the
// `path=` string comparison the hook shims make against `command -v abcd` and
// nothing more — adr-46 keeps hashing off the hook fast path — so it answers
// the one question every entry shape can be asked, including the two whose
// bytes ownership cannot rest on: a symlink (no bytes to hash) and the dev
// shim (bytes that are ours but are not a release artefact).
func pathEntryNames(target string) bool {
	rec, ok := readPathEntry()
	return ok && sameEntry(rec.path, target)
}

// removePathEntryFor drops the provenance record only when it names target.
// Uninstall removes ONE entry, and the record is home-scoped: a blanket delete
// would revoke the ownership of an install in another directory that this run
// never touched. The guarded form is also the only safe one now that every
// entry shape is recorded — a record left behind after its entry is gone would
// hand the ownership claim to whatever occupies that path next.
func removePathEntryFor(target string) {
	if pathEntryNames(target) {
		removePathEntry()
	}
}

// fileSHA256Hex hashes a regular file through the guarded read (no symlink
// leaf, no device, bounded size), or ok=false when it cannot.
func fileSHA256Hex(path string) (string, bool) {
	data, err := fsutil.ReadGuarded(path, maxBinaryArtefactBytes)
	if err != nil {
		return "", false
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), true
}

// isOwnedCopyFile reports whether target is the regular file abcd installed as
// its PATH entry: path-entry must name this very entry AND the file must still
// hash to the recorded value. A file that stopped matching was changed by
// something else, so it classifies foreign — refreshing or removing it would
// destroy work abcd cannot account for.
//
// The dev shim is excluded explicitly rather than by call order. Every entry
// abcd installs is now recorded, the shim included, so "the record names it and
// the bytes still match" no longer separates the copy from the shim — and this
// predicate is exported as the ownership proof `abcd update` accepts as
// permission to overwrite the file. Overwriting a shim the operator chose with
// a release binary is a silent mode switch, so the copy predicate says what it
// means: an owned copy is the verified release artefact, never the shim.
func isOwnedCopyFile(target string) bool {
	rec, ok := readPathEntry()
	if !ok || !sameEntry(rec.path, target) {
		return false
	}
	if isDevShimFile(target) {
		return false
	}
	got, ok := fileSHA256Hex(target)
	return ok && got == rec.sha
}

// IsOwnedPathCopy reports whether target is the regular file abcd installed as
// this machine's PATH entry: ~/.abcd/path-entry names that very entry and the
// bytes still hash to the recorded value. It is the exported face of the same
// predicate `ahoy` classifies with, published for `abcd update`, which needs a
// proof of ownership that no release deletion can revoke (iss-2609012000222546,
// and the itd-130 fidelity audit's ac-1 concern (b): the record was re-stamped
// after a swap but never consulted as a proof before one). It reads two local
// files and touches no network.
func IsOwnedPathCopy(target string) bool {
	return isOwnedCopyFile(target)
}

// ownedCopySourceReady reports whether a verified cache artefact exists to copy
// from — the precondition for installing (or healing to) an owned copy. When it
// does not hold, install degrades loudly to the spc-21 pinned symlink. The data
// dir is resolved for pluginRoot (a hook's environment, or the root's stamp
// from a terminal); cwd is the repository the verb runs against, which is what
// dataDirHazard needs to judge the resolved directory's shape.
func ownedCopySourceReady(cwd, pluginRoot string) bool {
	return cacheSourceReady(pluginDataDir(pluginRoot).dir, cwd)
}

// cacheSourceReady reports whether dataDir holds a cache that may be promoted:
// present (cachePresent) AND bound by the home-scoped attestation
// (cacheBindingProblem). Detection offers the owned-copy heal on exactly this
// predicate and install performs it on exactly this predicate, so the two can
// never disagree about the same directory.
func cacheSourceReady(dataDir, cwd string) bool {
	if !cachePresent(dataDir, cwd) {
		return false
	}
	_, problem := cacheBindingProblem(dataDir)
	return problem == ""
}

// cachePresent reports whether dataDir holds an artefact for this platform
// together with a parseable recorded hash to re-verify it against. An empty
// dataDir is no source at all, and neither is one of a shape the harness never
// produces (see dataDirHazard) — the check applies wherever the path came
// from, the plugin root's stamp included, because neither source examines the
// value it hands back. Presence is not trust: see cacheBindingProblem.
func cachePresent(dataDir, cwd string) bool {
	if dataDir == "" || dataDirHazard(dataDir, cwd) != "" {
		return false
	}
	if cacheRecordedSHA(dataDir) == "" {
		return false
	}
	return fileExists(cacheAssetPath(dataDir))
}

// cacheBindingProblem reports why the home-scoped attestation does not bind
// dataDir's cache, or "" when it does: the attestation exists and is
// well-formed, it names this very directory, and the cache's co-located
// binary-meta carries the attested hash. Any of the three failing means the
// directory and its record were chosen by something other than the bootstrap
// run that authenticated them — an environment variable, a rewritten cache —
// and nothing in it is a verified release artefact (GHSA-4q78-ccfv-f374). Every
// path in the reason is rendered in tilde form.
//
// On success the attestation itself is handed back, and it is the ONLY record
// a caller may act on afterwards: the co-located binary-meta is compared here
// and never read again, because a writer in the attested directory can swap
// the artefact and that record for a self-consistent forgery in the window
// between this check and the promotion (found in the security review of the
// first cut, reproduced in 0.25 s). The promotion hashes the artefact against
// the attested value, so a pair flipped after the binding fails the hash.
func cacheBindingProblem(dataDir string) (cacheAttestation, string) {
	record := "~/.abcd/" + cacheAttestationFile
	att, ok := readCacheAttestation()
	if !ok {
		return cacheAttestation{}, "no " + record + " record binds it — a session that authenticates the cache against the published release manifest writes one"
	}
	if resolvePath(att.dataDir) != resolvePath(dataDir) {
		return cacheAttestation{}, record + " names a different directory (" + displayPath(att.dataDir) + "), so this one was chosen by something other than the session that authenticated the cache"
	}
	if cacheRecordedSHA(dataDir) != att.sha {
		return cacheAttestation{}, "its recorded binary_sha256 is not the one " + record + " attests, so the cache changed after it was authenticated"
	}
	return att, ""
}

// RefreshPathEntryDigest re-records the provenance hash for the owned PATH
// copy after `abcd update` swapped the file at target — the one other verb
// that legitimately changes those bytes, and only after proving the new
// content against a published release's own checksums. Without this the entry
// update just refreshed would classify foreign forever after. A record that
// does not name target, a malformed digest, or no record at all is a no-op:
// this refreshes provenance, it never creates it.
func RefreshPathEntryDigest(target, shaHex string) {
	shaHex = strings.ToLower(shaHex)
	if !hexDigestOK(shaHex) {
		return
	}
	rec, ok := readPathEntry()
	if !ok || !sameEntry(rec.path, target) {
		return
	}
	// Preserve the recorded plugin_root: `abcd update` re-stamps the digest, not
	// the route home, so the terminal-side root resolution keeps working.
	_ = writePathEntry(rec.path, shaHex, rec.pluginRoot)
}
