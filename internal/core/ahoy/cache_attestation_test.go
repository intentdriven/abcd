package ahoy

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// GHSA-4q78-ccfv-f374 (iss-2609012039102770), option B as ruled on 2026-09-15:
// the persistent data dir reaches the owned PATH copy only through a
// home-scoped ATTESTATION the bootstrap writes after authenticating the cache
// against the published release manifest. CLAUDE_PLUGIN_DATA is taken from
// the environment as given, and the cache's binary-meta sits beside the
// artefact it vouches for, so whoever chooses the directory chooses both the
// bytes and the record — the old promotion re-hashed one against the other
// and called that verification. The attestation is the record the environment
// does not choose: `ahoy install` promotes a cache only when the attestation
// names that very directory AND the co-located record carries the attested
// hash, and the artefact still hashes to it.

// attestationBody renders the record the bootstrap writes for a cache holding
// body under dataDir.
func attestationBody(dataDir string, body []byte) string {
	sum := sha256.Sum256(body)
	return "data_dir=" + dataDir + "\nbinary_sha256=" + hex.EncodeToString(sum[:]) +
		"\ncache_trust=manifest\nattested_at=2026-09-15T00:00:00Z\n"
}

// writeUserCacheAttestation writes the home-scoped attestation verbatim.
func writeUserCacheAttestation(t *testing.T, body string) {
	t.Helper()
	p := userCacheAttestationPath()
	if p == "" {
		t.Fatal("no home-scoped cache-attestation location resolved")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// attestDataCache writes the attestation the bootstrap would have written for
// a cache holding body under data.
func attestDataCache(t *testing.T, data string, body []byte) {
	t.Helper()
	writeUserCacheAttestation(t, attestationBody(data, body))
}

// assertNoOwnedCopy: the target is not a regular file holding the cache
// artefact, and no path-entry vouches for the artefact's digest.
func assertNoOwnedCopy(t *testing.T, target string, res InstallResult) {
	t.Helper()
	if fi, err := os.Lstat(target); err == nil && fi.Mode().IsRegular() {
		if got, _ := os.ReadFile(target); string(got) == string(cacheArtefact) {
			t.Fatalf("install promoted the cache artefact into %s without a binding attestation; notes: %v", target, res.Notes)
		}
	}
	if rec, ok := readPathEntry(); ok {
		sum := sha256.Sum256(cacheArtefact)
		if rec.sha == hex.EncodeToString(sum[:]) {
			t.Fatalf("install recorded the unattested cache artefact's digest as this machine's abcd; notes: %v", res.Notes)
		}
	}
}

// TestInstallRefusesUnattestedEnvDataDir is the advisory's reproduction: an
// environment-supplied data dir holding a self-consistent artefact plus
// binary-meta pair, and no attestation in the home. The old promotion hashed
// the artefact against its neighbour and installed it 0755 as the owned PATH
// copy with provenance recorded. Now nothing is promoted, nothing vouches for
// it, and the note says which record is missing.
func TestInstallRefusesUnattestedEnvDataDir(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	data := t.TempDir()
	seedDataCacheAt(t, data, cacheArtefact)
	t.Setenv("CLAUDE_PLUGIN_DATA", data)

	res, err := Install(adoptableRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(binDir, "abcd")
	assertNoOwnedCopy(t, target, res)
	joined := notesJoined(res.Notes)
	if !strings.Contains(joined, cacheAttestationFile) {
		t.Errorf("the refusal must name the missing attestation; notes = %v", res.Notes)
	}
	if !strings.Contains(joined, "CLAUDE_PLUGIN_DATA") {
		t.Errorf("the refusal must say which source proposed the directory; notes = %v", res.Notes)
	}
	if strings.Contains(joined, home) {
		t.Errorf("notes must render home paths in tilde form, never absolute; notes = %v", res.Notes)
	}
	// Install degrades exactly as it does with no cache at all: the pinned
	// symlink, said out loud.
	if fi, err := os.Lstat(target); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("an unattested cache must degrade to the spc-21 symlink: %v (%v)", fi, err)
	}
	if !strings.Contains(joined, "symlink") {
		t.Errorf("the degradation must be named; notes = %v", res.Notes)
	}
}

// TestInstallRefusesEnvDataDirAttestedElsewhere: the attestation names the
// harness's real directory; an environment pointing at another directory
// holding the same bytes and record is still refused — the binding is to the
// directory, not merely to a hash the attacker can copy.
func TestInstallRefusesEnvDataDirAttestedElsewhere(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	real := t.TempDir()
	seedDataCacheAt(t, real, cacheArtefact)
	attestDataCache(t, real, cacheArtefact)
	attacker := t.TempDir()
	seedDataCacheAt(t, attacker, cacheArtefact)
	t.Setenv("CLAUDE_PLUGIN_DATA", attacker)

	res, err := Install(adoptableRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertNoOwnedCopy(t, filepath.Join(binDir, "abcd"), res)
	joined := notesJoined(res.Notes)
	if !strings.Contains(joined, cacheAttestationFile) || !strings.Contains(joined, "names a different") {
		t.Errorf("the refusal must say the attestation names a different directory; notes = %v", res.Notes)
	}
}

// TestInstallRefusesAttestedHashMismatch: the attestation names this very
// directory, but the co-located record no longer carries the attested hash —
// the cache was rewritten after the bootstrap authenticated it. Refused, and
// the mismatch is named.
func TestInstallRefusesAttestedHashMismatch(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	data := t.TempDir()
	seedDataCacheAt(t, data, cacheArtefact)
	attestDataCache(t, data, []byte("the release the bootstrap authenticated"))
	t.Setenv("CLAUDE_PLUGIN_DATA", data)

	res, err := Install(adoptableRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertNoOwnedCopy(t, filepath.Join(binDir, "abcd"), res)
	joined := notesJoined(res.Notes)
	if !strings.Contains(joined, cacheAttestationFile) || !strings.Contains(joined, "binary_sha256") {
		t.Errorf("the refusal must name the hash mismatch against the attestation; notes = %v", res.Notes)
	}
}

// TestInstallPromotesAttestedCacheByEitherRoute: a data dir the attestation
// names, whose record carries the attested hash, is promoted into the owned
// copy — reached through the hook's environment or through the plugin root's
// .data-dir stamp from a terminal. The same two routes without an attestation
// are refused: the stamp is a route, and the attestation is the trust.
func TestInstallPromotesAttestedCacheByEitherRoute(t *testing.T) {
	routes := map[string]func(t *testing.T, pluginRoot, data string){
		"env": func(t *testing.T, _ string, data string) {
			t.Setenv("CLAUDE_PLUGIN_DATA", data)
		},
		"stamp": func(t *testing.T, pluginRoot, data string) {
			t.Setenv("CLAUDE_PLUGIN_DATA", "")
			if err := os.WriteFile(filepath.Join(pluginRoot, dataDirStampFile), []byte("data_dir="+data+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, route := range routes {
		for _, attested := range []bool{true, false} {
			label := name + "/unattested"
			if attested {
				label = name + "/attested"
			}
			t.Run(label, func(t *testing.T) {
				home, pluginRoot := setupUserScope(t)
				binDir := filepath.Join(home, ".local", "bin")
				t.Setenv("PATH", binDir)
				data := t.TempDir()
				seedDataCacheAt(t, data, cacheArtefact)
				if attested {
					attestDataCache(t, data, cacheArtefact)
				}
				route(t, pluginRoot, data)

				res, err := Install(adoptableRepo(t), installOpts(), RefusingPrompter{})
				if err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(binDir, "abcd")
				if !attested {
					assertNoOwnedCopy(t, target, res)
					if !strings.Contains(notesJoined(res.Notes), cacheAttestationFile) {
						t.Errorf("the refusal must name the attestation; notes = %v", res.Notes)
					}
					return
				}
				fi, err := os.Lstat(target)
				if err != nil {
					t.Fatalf("install did not create %s: %v (notes %v)", target, err, res.Notes)
				}
				if fi.Mode()&os.ModeSymlink != 0 {
					t.Fatalf("an attested cache must be promoted to the owned copy, not degraded; notes %v", res.Notes)
				}
				if got, err := os.ReadFile(target); err != nil || string(got) != string(cacheArtefact) {
					t.Errorf("the owned copy must hold the attested artefact; got %q (%v)", got, err)
				}
				rec, ok := readPathEntry()
				sum := sha256.Sum256(cacheArtefact)
				if !ok || !sameEntry(rec.path, target) || rec.sha != hex.EncodeToString(sum[:]) {
					t.Errorf("path-entry must record the promoted copy; got %+v (ok=%v)", rec, ok)
				}
				if joined := notesJoined(res.Notes); strings.Contains(joined, cacheAttestationFile) || strings.Contains(joined, "symlink") {
					t.Errorf("no refusal may be reported for an attested cache; notes = %v", res.Notes)
				}
			})
		}
	}
}

// TestDetectOffersNoHealFromUnattestedCache: the symlink.legacy gap promises a
// heal to the owned copy, so it is offered only when install would actually
// perform it — never from a cache no attestation binds, or detection and
// install would disagree about the same directory.
func TestDetectOffersNoHealFromUnattestedCache(t *testing.T) {
	home, pluginRoot := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	linkOwned(t, filepath.Join(binDir, "abcd"), pluginRoot)
	data := t.TempDir()
	seedDataCacheAt(t, data, cacheArtefact)
	t.Setenv("CLAUDE_PLUGIN_DATA", data)

	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	if g := gapByID(det.Gaps, "symlink.legacy"); g != nil {
		t.Fatalf("detection offers a heal from an unattested cache: %+v", *g)
	}

	attestDataCache(t, data, cacheArtefact)
	det, err = Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	if g := gapByID(det.Gaps, "symlink.legacy"); g == nil {
		t.Fatalf("once attested, the same cache must be offered as the heal: %+v", det.Gaps)
	}
}

// TestReadCacheAttestationIgnoresMalformed: the record is read through the
// guarded bounded read like path-entry, and anything short of a well-formed
// manifest-trust attestation vouches for nothing — a truncated, relative,
// offline, oversize, or symlinked record all read as absent.
func TestReadCacheAttestationIgnoresMalformed(t *testing.T) {
	sum := sha256.Sum256(cacheArtefact)
	sha := hex.EncodeToString(sum[:])
	good := "data_dir=/harness/data\nbinary_sha256=" + sha + "\ncache_trust=manifest\n"
	cases := map[string]string{
		"relative data_dir": "data_dir=harness/data\nbinary_sha256=" + sha + "\ncache_trust=manifest\n",
		"no data_dir":       "binary_sha256=" + sha + "\ncache_trust=manifest\n",
		"short hash":        "data_dir=/harness/data\nbinary_sha256=" + sha[:63] + "\ncache_trust=manifest\n",
		"uppercase hash":    "data_dir=/harness/data\nbinary_sha256=" + strings.ToUpper(sha) + "\ncache_trust=manifest\n",
		"offline trust":     "data_dir=/harness/data\nbinary_sha256=" + sha + "\ncache_trust=offline\n",
		"no trust":          "data_dir=/harness/data\nbinary_sha256=" + sha + "\n",
		"empty":             "",
		"oversize":          good + strings.Repeat("padding=x\n", maxPathEntryBytes/10+1),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			setupHermetic(t)
			writeUserCacheAttestation(t, body)
			if rec, ok := readCacheAttestation(); ok {
				t.Errorf("a malformed attestation must read as absent, got %+v", rec)
			}
		})
	}

	t.Run("absent", func(t *testing.T) {
		setupHermetic(t)
		if rec, ok := readCacheAttestation(); ok {
			t.Errorf("no record must read as absent, got %+v", rec)
		}
	})

	t.Run("symlinked record", func(t *testing.T) {
		home, _ := setupHermetic(t)
		real := filepath.Join(t.TempDir(), "elsewhere")
		if err := os.WriteFile(real, []byte(good), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(real, userCacheAttestationPath()); err != nil {
			t.Fatal(err)
		}
		if rec, ok := readCacheAttestation(); ok {
			t.Errorf("a symlinked record must not be followed, got %+v", rec)
		}
	})

	t.Run("well-formed", func(t *testing.T) {
		setupHermetic(t)
		writeUserCacheAttestation(t, good+"attested_at=2026-09-15T00:00:00Z\n")
		rec, ok := readCacheAttestation()
		if !ok || rec.dataDir != "/harness/data" || rec.sha != sha || rec.trust != "manifest" {
			t.Errorf("a well-formed attestation must parse; got %+v (ok=%v)", rec, ok)
		}
	})
}
