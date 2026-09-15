package ahoy

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The cache attestation (GHSA-4q78-ccfv-f374, iss-2609012039102770, option B
// as ruled on 2026-09-15): a home-scoped record binding the persistent data
// dir's cache to a trust the environment cannot supply.
//
// CLAUDE_PLUGIN_DATA is read from the environment as given, and the cache's
// binary-meta sits beside the artefact it vouches for, equally writable by
// whoever chose the directory. So re-hashing the artefact against that record
// proved only that a file matched its own neighbour: with the variable pointed
// at a directory of their choosing, an attacker's self-consistent pair was
// promoted 0755 as the owned PATH copy, provenance recorded. The bootstrap is
// the one process that runs with the harness's real data dir and, when
// online, has just authenticated the cache against the published release
// manifest (adr-46 decision 3), so it records that fact where the environment
// does not reach — a sibling of ~/.abcd/path-entry, the home write adr-46
// decision 4 already treats as the ownership root. A promotion out of the
// cache now needs three things to agree: the attestation names the directory
// being promoted from, the co-located record carries the attested hash, and
// the artefact hashes to it. The trust floor moves from "the environment" to
// "a write into the caller's own home".
//
// The record is a claim of manifest trust and nothing weaker: the bootstrap
// writes it only after the manifest check passed for the very bytes now in
// the cache, and an offline run — which trusts the cache at corruption
// evidence only — neither writes nor rewrites it. It binds the CACHE, never
// the plugin-root binary: a source checkout as the plugin root with a locally
// built binary is untouched, which is why the rejected cross-check against
// that binary is not what this does.

// cacheAttestationFile is the record's name under ~/.abcd.
const cacheAttestationFile = "cache-attestation"

// afterCacheBound is a test seam, nil in production: it runs after the
// binding check has accepted a data dir and before the artefact is read, which
// is the window a writer in the attested directory can use to swap the
// artefact and its record for a self-consistent forgery. The promotion is
// correct only if nothing read in that window decides anything — the artefact
// is hashed against the ATTESTED value, never against the record beside it —
// and the seam is how a test occupies the window deterministically.
var afterCacheBound func(dataDir string)

// cacheAttestation is the parsed record: the data dir the bootstrap was
// handed by the harness, the hash the published manifest vouched for, and the
// trust vocabulary the bootstrap's cache_trust already uses.
type cacheAttestation struct {
	dataDir string
	sha     string
	trust   string
}

// userCacheAttestationPath is ~/.abcd/cache-attestation, beside path-entry
// and for the same reason: `ahoy install` runs from a terminal as well as
// from a hook, and the record must be readable wherever the promotion runs.
// Empty when the home directory cannot be resolved (every caller then reads
// "no attestation").
func userCacheAttestationPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".abcd", cacheAttestationFile)
}

// readCacheAttestation loads the record through the same guarded, bounded
// read path-entry uses, reporting ok only for a well-formed manifest-trust
// attestation: an absolute data_dir, a full lowercase-hex binary_sha256, and
// cache_trust=manifest. Anything less — absent, truncated, over the record
// cap, a symlinked leaf, a relative directory, an offline trust — vouches for
// nothing, so it reads as no attestation at all.
func readCacheAttestation() (cacheAttestation, bool) {
	path := userCacheAttestationPath()
	if path == "" {
		return cacheAttestation{}, false
	}
	raw, err := fsutil.ReadGuarded(path, maxPathEntryBytes)
	if err != nil {
		return cacheAttestation{}, false
	}
	var rec cacheAttestation
	for _, line := range strings.Split(string(raw), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch k {
		case "data_dir":
			rec.dataDir = v
		case "binary_sha256":
			rec.sha = v
		case "cache_trust":
			rec.trust = v
		}
	}
	if !filepath.IsAbs(rec.dataDir) || hasControlChar(rec.dataDir) || !hexDigestOK(rec.sha) || rec.trust != "manifest" {
		return cacheAttestation{}, false
	}
	return rec, true
}

// hasControlChar reports whether s carries a byte the bootstrap strips before
// it writes a value into a line-oriented record (\000-\037 and \177). No
// record the bootstrap wrote holds one, so a data_dir that does was written
// by something else and is refused rather than parsed — the same class the
// script's meta_field strips on read.
func hasControlChar(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] == 0x7f {
			return true
		}
	}
	return false
}
