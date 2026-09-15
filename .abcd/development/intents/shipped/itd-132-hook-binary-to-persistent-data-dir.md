---
id: itd-132
slug: hook-binary-to-persistent-data-dir
spec_id: spc-35
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-105]
severity: major
impact: fix
promoted_from: iss-2608210934566221
---

# The hook binary survives plugin updates

## Press Release

> Updating the abcd plugin is now a non-event. The checksum-verified hook
> binary is no longer a casualty of the harness's plugin-update lifecycle —
> the commit-stamped cache directory that every update replaces and
> garbage-collection later deletes. The verified release artefact is kept
> once in the harness's persistent per-plugin data directory; an update
> copies it into the fresh plugin root instead of re-downloading ~11MB, so
> the first-hook-after-update window in which hooks run without their binary
> shrinks from once per plugin update to once per released binary. And the
> `abcd` command on PATH keeps working across any number of updates: it is a
> regular file abcd owns and refreshes, not a symlink into a directory the
> harness will delete. Each plugin root still runs its own binary, verified
> exactly as before, and every steady-state session pays one file test and
> no network.

## Why This Matters

The plugin cache directory is harness territory: re-cloned on every update,
documented as not preserving extra files, orphaned and garbage-collected
~14 days later. Storing the downloaded binary there meant fighting the
platform's lifecycle — the 2026-08-21 post-mortem (iss-2608210934566221)
showed the full cost in one update: a re-download, a first-hook window with
no binary, a binary copy per plugin version, and a PATH symlink 14 days from
dangling (iss-2608210934566222). The harness provides a persistent
per-plugin data directory (`${CLAUDE_PLUGIN_DATA}`, exported to all hook
processes, scope-agnostic, documented path shape) that survives updates and
is deleted only on full uninstall — the platform-sanctioned home for the
downloaded artefact.

## Decisions

- **Reverses itd-130's 2026-08-20 decision** that "the persistent-data-dir
  alternative is rejected, not deferred": the shape ruled below adopts that
  alternative as a download cache. Recorded as a reversal, not a refinement,
  at the maintainer's confirmation on 2026-09-01 (iss-2609012111150528).
- **Shape (maintainer-ruled at the planning interview, 2026-08-21).** The
  data dir is a **download cache, not an execution home**: bootstrap fetches
  and verifies the release artefact into `${CLAUDE_PLUGIN_DATA}` once, and
  copies a checksum-matched artefact into each plugin root; execution stays
  per-root, preserving per-root version pairing. The PATH entry becomes a
  **copied regular file** owned and refreshed by abcd (`ahoy
  install`/bootstrap), never a symlink into a harness-owned directory. The
  relocate-execution alternative was rejected for creating a shared mutable
  binary across roots (skew semantics, cross-root locking, dev-shim
  collision, unbounded at-rest tampering window — adversarial review,
  2026-08-21).
- **Routing (itd-84 SPLIT, human-confirmed 2026-08-21).** Capability — this
  intent. Trust rule (persistence must not weaken the spc-21 verification
  posture, at fetch time *and at rest*; SessionEnd performs no network
  work) — ADR + brief invariant, drafted alongside the spec; captured as
  iss-2608210934566226. Stance (store durable state where the platform
  documents it survives) — principle candidate; captured as
  iss-2608210934566227. Plumbing — spec detail.
- **Supersession (maintainer-confirmed, 2026-08-21).** itd-132 supersedes
  spc-21's "update into a fresh cache dir heals by re-fetch" acceptance
  criterion; the replacement contract is "an update never re-fetches unless
  the released binary changed". The spc-21 *verification* posture is
  untouched.
- **Refresh detector (maintainer-ruled, 2026-08-21).** Local state only:
  a plugin update (provisioning stamp differs) triggers one re-resolve of
  the latest release tag; fetch only when the tag changed. The known gap —
  a release cut with no plugin update never triggers — is **accepted**: the
  skew notice surfaces it, and `abcd update` (itd-130) remains the explicit
  refresh path.
- **Dev builds stay out of the trusted location.** The `--dev` shim builds
  and runs from the source checkout; a locally built binary is never
  written into the data-dir cache or over a verified artefact, and
  provenance metadata never describes a build it does not match.
- **Impact: `fix`** (maintainer-ruled) — the promise is repairing failure
  modes of the existing bootstrap contract, not new capability.

## Out of Scope

- The SessionEnd hook ordering fix (never bootstrap at exit) —
  iss-2608210934566223, fixed test-first on its own branch
  (`fix/iss-sessionend-never-bootstraps`), independent of this intent. This
  intent shrinks that window's *frequency*; the ordering fix removes the
  download from the exit path entirely, and both stand on their own.
- The missed-transcript recovery sweep (iss-2608210934566224) and the
  statusline surface (iss-2608210934566225) — held in the ledger as seeds.

## Scope Conditions

None stated.

## Acceptance Criteria

- Given a provisioned install with a cached release artefact, When the
  plugin updates into a fresh plugin root (simulated in tests as a second
  plugin root appearing), Then the next hook event provisions that root by
  checksum-verified copy from the cache — no network fetch, no
  missing-binary message.
- Given a plugin update whose latest release differs from the cached
  artefact's recorded release, When the local-state detector fires, Then
  the new artefact is fetched and checksum-verified into the cache under
  the unchanged spc-21 verification posture, and the root is provisioned
  from it.
- Given a steady-state session (root binary present), When any hook fires,
  Then the cost is one file test and no network.
- Given `ahoy install`, When the PATH entry is written, Then it is a
  regular file owned by abcd that survives plugin updates and
  cache-directory deletion (simulated in tests as the original plugin root
  being removed); a legacy abcd-owned symlink into a plugin root is
  detected as a heal-able gap and replaced.
- Given any promotion of a cached or adopted artefact (into a plugin root,
  onto PATH, or into the cache at migration), When the copy is made, Then
  the artefact is re-verified against its recorded `binary_sha256` first; a
  mismatch refuses loudly and installs nothing.
- Given a session on a plugin root other than the one that last fetched,
  When the session-start surface renders, Then the version-skew notice is
  computed against the *live* plugin root, never against a recorded
  provisioning-time root.
- Given a harness that provides no persistent data directory (dev/local
  installs — unverified harness behaviour), When hooks run, Then behaviour
  degrades loudly to the current per-root fetch, never silently.

## Open Questions

1. **Cache-write concurrency** (spec detail): the bootstrap lock and temp
   dirs are per-plugin-root today; concurrent roots fetching into the one
   shared cache need the lock and temp dir relocated into the data dir
   (same-filesystem rename atomicity preserved).
2. **Uninstall scope** (spec detail): `ahoy uninstall` removes the
   abcd-owned PATH file; whether it also empties the data-dir cache or
   leaves that to the harness's uninstall-from-all-scopes deletion — decide
   in the spec with the ADR.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-acde3e9ce729 -->
Fidelity review — receipt rcp-acde3e9ce729 (verifier abcd:intent-auditor claude-fable-5-1).

Provenance: abcd:intent-auditor@claude-fable-5-1 · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:8f74b5e66e6de0e6aedbdb44e1f9861c227e39519dd6bf804cc8747b89a91ba9
Input attestations: diff:06c3114c^1..06c3114c (PR #410: e9af4054, ee3a7306, bdc5f2ab)@sha256:879c957d0bad0bfe1e28f5eef8a4312c25945c4c8a2ea417ac225b0fc3dab6dd; tree:HEAD 8f68ffb34558752618f5f1032aec356dfc22ce68 (git archive HEAD, worktree ship-itd-130-132)@sha256:03a31448816716d6b9711d6996ca503423194d7268f42861ee052934110e04d7;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 4 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: A second fresh root is provisioned by a re-hashed copy out of the cache with zero artefact downloads and no missing-binary line (TestBootstrapSecondRootIsProvisionedFromCache); the concern is that 'no network fetch' is delivered as 'no ARTEFACT fetch' — the online equal-tag path still performs a release resolve and a checksums.txt GET to authenticate the cache (adr-46 decision 3, a signed-off divergence), and only the offline path is network-free.
  evidence: hooks/bootstrap.sh:410 — "if [ -n \"$cache_mode\" ] && [ -f \"$cache_binary\" ]; then"
  evidence: hooks/bootstrap.sh:648 — "cp \"$cache_binary\" \"$root_tmp/abcd\""
  evidence: hooks/bootstrap.sh:653 — "[ \"$got_sha\" = \"$expected_sha\" ] ||"
  evidence: internal/surface/cli/bootstrap_cache_test.go:188 — "func TestBootstrapSecondRootIsProvisionedFromCache"
  evidence: internal/surface/cli/bootstrap_cache_test.go:234 — "the second root must be provisioned by copy, not re-download"
  evidence: internal/surface/cli/bootstrap_cache_test.go:154 — "func TestBootstrapCacheHitSurvivesOfflineResolve"
  evidence: hooks/bootstrap.sh:422 — "-w '%{redirect_url}' \"$releases_url/latest\""
  evidence: hooks/bootstrap.sh:456 — "curl ... -o \"$auth_tmp/checksums.txt\""
  evidence: internal/surface/cli/bootstrap_cache_test.go:129 — "an online cache hit must fetch the published checksums.txt"
- ac-2 — MET: The refresh detector re-resolves the latest tag on every missing-root run, compares it with the cached release_tag, and on a different tag downloads asset + checksums.txt from the one resolved release, verifies, publishes into the cache by rename, rewrites binary-meta, and provisions the root from the fresh cache; TestBootstrapNewReleaseRefreshesCacheAndPathEntry pins root, cache, and meta all holding the new artefact, and the spc-21 posture tests (origins constant, environment origins ignored) still hold.
  evidence: hooks/bootstrap.sh:452 — "elif [ \"$resolved_tag\" = \"$cached_tag\" ]; then"
  evidence: hooks/bootstrap.sh:518 — "curl -q -fsSL --proto '=https' --proto-redir '=https' --max-time 120 -o \"$tmp/$asset\""
  evidence: hooks/bootstrap.sh:535 — "(cd \"$tmp\" && $verify) > /dev/null 2>&1 ||"
  evidence: hooks/bootstrap.sh:572 — "mv -f \"$tmp/$asset\" \"$cache_binary\""
  evidence: internal/surface/cli/bootstrap_cache_test.go:243 — "func TestBootstrapNewReleaseRefreshesCacheAndPathEntry"
  evidence: internal/surface/cli/bootstrap_cache_test.go:268 — "the cache must hold the NEW artefact"
  evidence: internal/surface/cli/bootstrap_test.go:1248 — "func TestBootstrapFetchOriginsAreConstants"
  evidence: internal/surface/cli/bootstrap_test.go:1547 — "func TestBootstrapIgnoresEveryEnvironmentOrigin"
- ac-3 — MET_WITH_CONCERNS: UserPromptSubmit/PreToolUse/PreCompact gate the bootstrap on a single `-x $r/abcd` test and never invoke it when the binary is present (TestBinaryHooksSteadyStateBypassesTheSalvage), and the bootstrap fast path touches no network (TestBootstrapFastPathTouchesNoNetwork); the concern is that SessionStart always invokes bootstrap.sh, whose cache-mode fast path adds a second stat of the root's .binary-meta, so a steady-state SessionStart pays a few file tests rather than literally one — and the cache-mode steady state for a cache-provisioned root (data dir set, no .binary-meta) is pinned only indirectly, via the migration test's silent second run.
  evidence: hooks/hooks.json:8 — "if [ ! -x \"$r/abcd\" ] && [ -x \"$r/hooks/bootstrap.sh\" ]"
  evidence: hooks/hooks.json:19 — "if [ -x \"$CLAUDE_PLUGIN_ROOT/hooks/bootstrap.sh\" ]; then ... b=$(\"$CLAUDE_PLUGIN_ROOT/hooks/bootstrap.sh\""
  evidence: hooks/bootstrap.sh:193 — "if [ -f \"$binary\" ] && [ -x \"$binary\" ]; then"
  evidence: hooks/bootstrap.sh:205 — "[ -f \"$root_meta\" ] || exit 0"
  evidence: internal/surface/cli/hooks_selfprovision_test.go:217 — "func TestBinaryHooksSteadyStateBypassesTheSalvage"
  evidence: internal/surface/cli/bootstrap_test.go:369 — "func TestBootstrapFastPathTouchesNoNetwork"
  evidence: internal/surface/cli/bootstrap_cache_test.go:531 — "the second run must be a silent no-op"
- ac-4 — MET_WITH_CONCERNS: installOwnedEntry writes a 0755 regular-file copy of the verified cache artefact and records path+hash+plugin_root in ~/.abcd/path-entry; TestInstallWritesOwnedCopyFromCache deletes the plugin root and proves the copy still executes and still classifies owned, and detectPathSymlink raises symlink.legacy for an owned symlink into the root, which TestInstallHealsLegacySymlinkToOwnedCopy shows replaced by the copy. Two named concerns: (a) the cache lookup is keyed solely on CLAUDE_PLUGIN_DATA (pluginDataDir), which a terminal does not export, so `ahoy install` run the way the bootstrap notice and the install guide direct — by absolute path from the user's own terminal — finds no cache and degrades (loudly) to the spc-21 symlink; the owned copy lands only when the verb runs with the harness variable present. (b) The criterion's Given is `ahoy install`, so it does not cover the README/install-guide one-liner, which writes the same ~/.local/bin/abcd with `install -m 0755` and no path-entry record: that copy survives updates as a static file but is never refreshed by the bootstrap (gated on the record), classifies foreign to `ahoy`, and is refreshed only by an explicit `abcd update`. The 2026-09-01 observation is therefore a press-release-level divergence, not a failure of ac-4 as written.
  evidence: internal/core/ahoy/apply.go:852 — "func (a *applyCtx) installOwnedEntry(target string, kind binTargetKind) {"
  evidence: internal/core/ahoy/apply.go:894 — "fsutil.WriteFileAtomic(target, data, 0o755)"
  evidence: internal/core/ahoy/owned_copy.go:140 — "func writePathEntry(target, shaHex, pluginRoot string) error {"
  evidence: internal/core/ahoy/detect.go:470 — "ID: \"symlink.legacy\", Category: ConfigChange, Scope: \"machine\","
  evidence: internal/core/ahoy/owned_copy_test.go:67 — "func TestInstallWritesOwnedCopyFromCache"
  evidence: internal/core/ahoy/owned_copy_test.go:113 — "the owned copy must keep executing after the plugin root is deleted"
  evidence: internal/core/ahoy/owned_copy_test.go:124 — "func TestInstallHealsLegacySymlinkToOwnedCopy"
  evidence: internal/core/ahoy/data_dir.go:13 — "return os.Getenv(\"CLAUDE_PLUGIN_DATA\")"
  evidence: internal/core/ahoy/apply.go:853 — "if !ownedCopySourceReady() {"
  evidence: internal/core/ahoy/owned_copy_test.go:201 — "func TestInstallWithoutCacheDegradesLoudlyToSymlink"
  evidence: hooks/bootstrap.sh:749 — "run this once — the path is absolute because abcd is not on your PATH yet ... %s ahoy install"
  evidence: docs/how-to/install.md:70 — "running it once by its absolute path — `'<plugin-root>/abcd' ahoy install`"
  evidence: README.md:106 — "install -m 0755 \"$b\" \"$HOME/.local/bin/abcd\""
  evidence: hooks/bootstrap.sh:589 — "if [ -n \"$path_entry\" ] && [ -f \"$path_entry\" ]; then"
  evidence: internal/core/ahoy/detect.go:451 — "ID: \"symlink.foreign\", Category: ConfigChange, Scope: \"machine\","
  evidence: internal/core/update/update.go:139 — "case ahoy.UpdateTargetFile:"
- ac-5 — MET_WITH_CONCERNS: Every promotion re-hashes against the recorded binary_sha256 before anything lands: cache->root (refuses loudly, TestBootstrapCorruptCacheRefusesLoudly), cache->PATH in `ahoy install` (refuses loudly with no record written, TestInstallRefusesCorruptCacheArtefact), the bootstrap's PATH refresh (post-copy re-hash before the rename), and the migration seed (hash against the root's .binary-meta); the cache itself is additionally authenticated against the published manifest online. The concern is the word 'loudly': the migration-seed mismatch is silently ignored (exit 0, by spec Design 4 and pinned so by TestBootstrapMigrationIgnoresMismatchedRootBinary), and the bootstrap PATH-refresh mismatch branch silently discards the temp copy with no notice line and no test.
  evidence: hooks/bootstrap.sh:653 — "[ \"$got_sha\" = \"$expected_sha\" ] ||"
  evidence: hooks/bootstrap.sh:654 — "refuse \"the cached artefact at $cache_binary does not match its recorded SHA-256 checksum"
  evidence: internal/surface/cli/bootstrap_cache_test.go:357 — "func TestBootstrapCorruptCacheRefusesLoudly"
  evidence: internal/core/ahoy/apply.go:875 — "if got != want {"
  evidence: internal/core/ahoy/owned_copy_test.go:169 — "func TestInstallRefusesCorruptCacheArtefact"
  evidence: hooks/bootstrap.sh:611 — "if [ \"$new_sha\" = \"$binary_sha256\" ] &&"
  evidence: hooks/bootstrap.sh:627 — "rm -f \"$path_tmp\""
  evidence: hooks/bootstrap.sh:250 — "[ \"$got\" = \"$want\" ] || exit 0"
  evidence: internal/surface/cli/bootstrap_cache_test.go:546 — "func TestBootstrapMigrationIgnoresMismatchedRootBinary"
  evidence: internal/surface/cli/bootstrap_cache_test.go:400 — "func TestBootstrapAuthenticatesCacheAgainstPublishedManifest"
  evidence: .abcd/development/decisions/adrs/0046-persistence-never-weakens-the-verification-posture.md:47 — "Every promotion of a persisted artefact re-verifies against its recorded"
- ac-6 — MET: binarySkewNotice takes the surface commit from livePluginSHA — the live CLAUDE_PLUGIN_ROOT basename read at render time — and the cache meta no longer carries plugin_sha (asserted absent in the cache-provisioning test); TestHookSessionStartSkewComparesTheLiveRoot seeds a meta whose recorded plugin_sha equals the release commit while the live root differs and requires the notice to name the live root's commit.
  evidence: internal/surface/cli/skew.go:47 — "pluginSHA, releaseSHA := livePluginSHA(root), meta[\"release_sha\"]"
  evidence: internal/surface/cli/skew.go:99 — "func livePluginSHA(root string) string {"
  evidence: hooks/bootstrap.sh:256 — "The cache meta is the spc-21 record minus plugin_sha"
  evidence: internal/surface/cli/bootstrap_cache_test.go:222 — "the cache meta must not record plugin_sha"
  evidence: internal/surface/cli/skew_test.go:91 — "func TestHookSessionStartSkewComparesTheLiveRoot"
  evidence: internal/surface/cli/skew_test.go:104 — "the notice must be computed against the LIVE plugin root, never a recorded provisioning-time root"
- ac-7 — MET: With CLAUDE_PLUGIN_DATA unset or its cache dir unmakeable, cache_mode stays empty, the run takes the spc-21 per-root fetch with the per-root lock and temp dir, writes the root-local .binary-meta, and appends the degrade_note to the success notice; TestBootstrapDegradesLoudlyWithoutDataDir asserts the install, the 'persistent plugin data' wording, and the root-local record, and the ahoy side degrades loudly to the pinned symlink (TestInstallWithoutCacheDegradesLoudlyToSymlink).
  evidence: hooks/bootstrap.sh:307 — "degrade_note=' The persistent plugin data directory is unavailable, so the binary was fetched into this plugin root only"
  evidence: hooks/bootstrap.sh:311 — "if [ -n \"$data_dir\" ]; then"
  evidence: hooks/bootstrap.sh:662 — "# Degraded install (no usable data dir, or a hash that failed to parse):"
  evidence: internal/surface/cli/bootstrap_cache_test.go:448 — "func TestBootstrapDegradesLoudlyWithoutDataDir"
  evidence: internal/surface/cli/bootstrap_cache_test.go:461 — "the degradation must be said out loud, never a silent fallback"
  evidence: internal/core/ahoy/apply.go:858 — "a.refuse(\"no verified release artefact is available in the persistent plugin data directory, so the PATH entry was written as a symlink"
  evidence: internal/core/ahoy/owned_copy_test.go:201 — "func TestInstallWithoutCacheDegradesLoudlyToSymlink"

Gap audit:
- honoured:
  - The verified release artefact is kept once in the persistent data dir and a fresh plugin root is provisioned from it by a re-verified copy instead of re-downloading
    evidence: hooks/bootstrap.sh:637 — "if [ -n \"$cache_mode\" ] && { [ -n \"$use_cache\" ] || [ \"$expected_sha\" != unknown ]; }; then"
    evidence: internal/surface/cli/bootstrap_cache_test.go:188 — "func TestBootstrapSecondRootIsProvisionedFromCache"
  - An update never re-fetches unless the released binary changed (supersedes spc-21's heal-by-refetch criterion)
    evidence: hooks/bootstrap.sh:448 — "if [ -n \"$cached_sha\" ]; then"
    evidence: internal/surface/cli/bootstrap_cache_test.go:243 — "func TestBootstrapNewReleaseRefreshesCacheAndPathEntry"
  - The cache's trust is established against the published checksums.txt when online and named as unauthenticated when offline (adr-46 decision 3)
    evidence: hooks/bootstrap.sh:460 — "if [ -n \"$published\" ] && [ \"$published\" = \"$cached_sha\" ]; then"
    evidence: hooks/bootstrap.sh:492 — "provisioned from an unauthenticated cache while offline"
    evidence: internal/surface/cli/bootstrap_cache_test.go:400 — "func TestBootstrapAuthenticatesCacheAgainstPublishedManifest"
  - Each plugin root still runs its own binary, verified exactly as before (spc-21 posture untouched: constant origins, HTTPS pin, -q-first curl, proxy/CA scrub)
    evidence: hooks/bootstrap.sh:44 — "repo_url=\"https://github.com/intentdriven/abcd\""
    evidence: hooks/bootstrap.sh:67 — "unset HTTPS_PROXY https_proxy HTTP_PROXY http_proxy ALL_PROXY all_proxy CURL_HOME"
    evidence: internal/surface/cli/bootstrap_test.go:1248 — "func TestBootstrapFetchOriginsAreConstants"
  - SessionEnd performs no network work and never bootstraps
    evidence: hooks/hooks.json:50 — "if [ -f \"$r/abcd\" ] && [ -x \"$r/abcd\" ]; then exec \"$r/abcd\" hook session-end; fi"
    evidence: internal/surface/cli/hooks_selfprovision_test.go:292 — "func TestSessionEndNeverBootstraps"
  - The PATH-copy provenance record is reachable without the harness environment and carries the plugin root so terminal verbs still resolve it
    evidence: internal/core/ahoy/owned_copy.go:58 — "return filepath.Join(home, \".abcd\", \"path-entry\")"
    evidence: internal/core/ahoy/store.go:104 — "if rec, ok := readPathEntry(); ok && rec.pluginRoot != \"\" {"
    evidence: internal/core/ahoy/owned_copy_test.go:409 — "func TestTerminalOwnedCopySurvivesClearedHarnessEnv"
    evidence: internal/core/ahoy/store_symlink_test.go:58 — "func TestResolvePluginRootThroughOwnedCopyRecord"
  - One-way, idempotent migration seeds the cache from a verified pre-cache root binary and refuses a directory planted at the cache path
    evidence: hooks/bootstrap.sh:226 — "if [ -e \"$cache_binary\" ] && [ ! -f \"$cache_binary\" ]; then"
    evidence: internal/surface/cli/bootstrap_cache_test.go:493 — "func TestBootstrapMigratesVerifiedRootBinaryIntoCache"
    evidence: internal/surface/cli/bootstrap_cache_test.go:574 — "func TestBootstrapMigrationRefusesDirectoryAtCacheBinary"
  - The bootstrap refreshes the registered PATH copy in the same run a new release lands, re-verified and re-stamped, and re-stamps plugin_root on cache hits
    evidence: hooks/bootstrap.sh:580 — "# Refresh the abcd-owned PATH copy in the same run"
    evidence: hooks/bootstrap.sh:707 — "if [ -n \"$path_entry\" ] && [ -f \"$path_entry\" ]; then"
    evidence: internal/surface/cli/bootstrap_cache_test.go:614 — "func TestBootstrapCacheModeSkipsPathRefreshOnCacheHit"
  - ADR-46 and brief invariant 12 record the trust rule; the plugin and CLI docs describe the owned copy rather than a symlink
    evidence: .abcd/development/brief/02-constraints/03-invariants.md:37 — "12. **Persisting the hook binary never weakens the verification posture**"
    evidence: commands/ahoy.md:136 — "`abcd ahoy install` without `--dev` switches back to the pinned owned copy."
    evidence: docs/how-to/install.md:40 — "cache (`$CLAUDE_PLUGIN_DATA`), and a plugin update — which lands in a fresh,"
- diverged:
  - "an update copies it into the fresh plugin root instead of re-downloading" / ac-1 "no network fetch" — delivered as no ARTEFACT download; the online equal-tag path still makes a release resolve and a checksums.txt GET (adr-46 decision 3)
    evidence: hooks/bootstrap.sh:422 — "-w '%{redirect_url}' \"$releases_url/latest\""
    evidence: hooks/bootstrap.sh:456 — "curl ... -o \"$auth_tmp/checksums.txt\""
    evidence: internal/surface/cli/bootstrap_cache_test.go:129 — "an online cache hit must fetch the published checksums.txt"
  - "the abcd command on PATH keeps working across any number of updates: it is a regular file abcd owns and refreshes" — true only for a copy `ahoy install` registered while CLAUDE_PLUGIN_DATA was exported; from a terminal (the invocation the notice and install guide direct) the cache is invisible and `ahoy install` writes the spc-21 symlink, and a README one-liner copy is unregistered: it keeps working as a static file, is never bootstrap-refreshed, classifies foreign to `ahoy`, and is refreshed only by an explicit `abcd update`
    evidence: internal/core/ahoy/data_dir.go:13 — "return os.Getenv(\"CLAUDE_PLUGIN_DATA\")"
    evidence: internal/core/ahoy/apply.go:853 — "if !ownedCopySourceReady() {"
    evidence: hooks/bootstrap.sh:749 — "%s ahoy install"
    evidence: README.md:106 — "install -m 0755 \"$b\" \"$HOME/.local/bin/abcd\""
    evidence: hooks/bootstrap.sh:589 — "if [ -n \"$path_entry\" ] && [ -f \"$path_entry\" ]; then"
    evidence: internal/core/ahoy/store.go:485 — "return binTargetForeign"
    evidence: internal/core/update/update.go:139 — "case ahoy.UpdateTargetFile:"
  - "every steady-state session pays one file test and no network" — no network holds everywhere; SessionStart always invokes bootstrap.sh, whose cache-mode fast path adds a stat of the root's .binary-meta, so the count is a few file tests rather than one
    evidence: hooks/hooks.json:19 — "b=$(\"$CLAUDE_PLUGIN_ROOT/hooks/bootstrap.sh\" 2>&1 >/dev/null </dev/null)"
    evidence: hooks/bootstrap.sh:203 — "[ -n \"$data_dir\" ] || exit 0"
    evidence: hooks/bootstrap.sh:205 — "[ -f \"$root_meta\" ] || exit 0"
  - ac-5 "a mismatch refuses loudly" — loud at cache->root and at `ahoy install`'s cache->PATH; the migration-seed mismatch is silently ignored (spec Design 4) and the bootstrap PATH-refresh mismatch is silently discarded with no notice text
    evidence: hooks/bootstrap.sh:250 — "[ \"$got\" = \"$want\" ] || exit 0"
    evidence: hooks/bootstrap.sh:626 — "else"
    evidence: hooks/bootstrap.sh:627 — "rm -f \"$path_tmp\""
    evidence: .abcd/development/specs/closed/spc-35-hook-binary-to-persistent-data-dir.md:145 — "on mismatch or missing meta, ignore it and fetch fresh"
- missing:
  - Registration (or any refresh path short of an explicit `abcd update`) for the README/install-guide one-liner copy: nothing writes ~/.abcd/path-entry for it and RefreshPathEntryDigest never creates a record, so the press release's "owns and refreshes" does not reach that route
    evidence: README.md:106 — "install -m 0755 \"$b\" \"$HOME/.local/bin/abcd\""
    evidence: internal/core/ahoy/owned_copy.go:207 — "this refreshes provenance, it never creates it"
  - A test pinning the bootstrap PATH-refresh mismatch branch (post-copy re-hash fails): the sibling tests cover match, foreign file, and absent path only
    evidence: hooks/bootstrap.sh:611 — "if [ \"$new_sha\" = \"$binary_sha256\" ] &&"
    evidence: internal/surface/cli/bootstrap_cache_test.go:296 — "func TestBootstrapNewReleaseLeavesForeignPathFileAlone"
    evidence: internal/surface/cli/bootstrap_cache_test.go:330 — "func TestBootstrapNewReleaseSkipsAbsentPathCopy"
  - A test pinning the cache-mode steady state for a cache-provisioned root (data dir set, binary present, no .binary-meta) at zero network; the no-network fast-path test runs without a data dir and the migration test's second run has a .binary-meta root
    evidence: hooks/bootstrap.sh:205 — "[ -f \"$root_meta\" ] || exit 0"
    evidence: internal/surface/cli/bootstrap_test.go:369 — "func TestBootstrapFastPathTouchesNoNetwork"