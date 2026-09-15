---
id: itd-130
slug: abcd-update-completes-a-chosen-update-in-one-verb-it-fetches
spec_id: spc-32
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-105, itd-108]
severity: major
impact: additive
---

# `abcd update` completes a chosen update in one verb

## Press Release

> **Updating abcd is now one verb, and it is always the user's verb.** `abcd
> version --check` has said "update available: v0.6.0 -> v0.6.1" since the
> staleness work landed — and then left the user to find the README one-liner.
> With this intent, `abcd update` completes what `--check` reports: it names
> the release it resolved (or takes an explicit tag), fetches the binary for
> this platform over a pinned transport, verifies it against the same
> release's `checksums.txt` before anything is replaced, swaps the
> PATH-installed copy atomically, and prints a receipt naming the origin, the
> old and new versions, and the digest it verified. On a terminal it shows
> download progress; piped or hooked, it is silent and loud only on failure.
> Nothing about the [[adr-38-implicit-checks-are-disk-only]] posture moves:
> abcd still never looks for updates on its own — the check is a verb the
> user runs, and now the fix is too.

> "The tool told me I was stale and then wished me luck," said Alice, who had
> typed `abcd version --check` on the machine where they first installed the
> one-liner. "Now the next line is `abcd update`, and the receipt tells me
> what it verified before it touched anything." Bob, whose platform team pins
> the developer tooling on every machine they hand out, cared about the
> refusals as much as the swaps: "Eight error messages in the binary say
> 'upgrade abcd'. Now there is a verb they can name — and on a plugin install
> it correctly says 'take a plugin update' instead of touching the plugin
> root." Carol, who reviews what runs on their team's machines, read the
> receipt line: "Fetched from the pinned release origin, verified against the
> release's own checksums, or it refuses. That is the same bar the bootstrap
> already meets — now it is one implementation, not three."

## Why This Matters

[[adr-38-implicit-checks-are-disk-only]] built the grammar this verb
completes: implicit operations never touch the network (tier 1), the network
answers only an explicit ask whose documented meaning IS that fetch (tier 2),
and provisioning completes a chosen update rather than discovering one
(tier 3). `abcd version --check` is tier 2 and already exists
(`internal/core/vintage/release.go` resolves the tag from the unfollowed
`releases/latest` redirect under the urlguard policy). The action half is
missing: the binary tells users to "upgrade abcd" in eight schema-too-new
error sites and gives them no verb to do it.

The provisioning logic exists today in three places at two bars:
`hooks/bootstrap.sh` (hardened: pinned origins, proxy/CA scrubbing,
tag-pinned binary+manifest, same-origin checksum refusal, atomic rename),
the README one-liner (checksummed but hand-run), and — without this intent —
nothing at all for the machine whose `~/.local/bin/abcd` is a real copy that
no plugin update will ever refresh. One canonical primitive says the fourth
implementation must not happen: the fetch/verify/swap core lands once in
`internal/core`, the CLI renders it, and the end-state this verb enables is
`hooks/bootstrap.sh` demoted to the cold-start trampoline — the only state a
Go updater can never serve is the state where no binary exists yet. Windows
is the forcing function: the `.sh` files were always placeholders for
getting mac/linux right, a compiled verb is the only updater that ever runs
on Windows, and the host harness's own updater already demonstrates the
rename-dance this verb needs there.

The install-shape dispatch is the part a naive updater gets wrong, and the
repo record already names each shape: a plugin-root binary is refreshed by
the plugin update itself ([[itd-108]]'s one-cut coherence — an updater that
bumped it independently would recreate the exact surface/binary skew that
intent exists to make structurally impossible, so this verb REFUSES there
and names the host's plugin-update path); an abcd-owned PATH symlink
stranded by a plugin update is healed by `ahoy install` (iss-345); the dev
shim is a chosen install mode, never silently replaced with a release
binary; a real copy at `~/.local/bin` is this verb's home case, owned by
provenance (see Decisions); and a package-manager path is refused with the
manager's own upgrade command (Homebrew recorded-and-parked in the decision
log, 2026-08-20 — the refusal ships with the verb, the tap ships later;
detection is a resolved-path-under-brew-prefix test, never a substring
match).

## Decisions (grilled 2026-08-20)

The maintainer resolved the open questions at the planning interview; these
are commitments, not options.

- **Ownership of a real copy is provenance: digest-matches-a-release.** At
  update time the existing copy is hashed and treated as abcd's when its
  digest appears in a published release's `checksums.txt`. No new state, no
  receipt file; the network use is already granted (the user typed the
  verb). A copy matching no release stays foreign and is refused with the
  occupant described.
- **Reversed 2026-08-21 by itd-132 (confirmed by the maintainer 2026-09-01).**
  The persistent-data-dir alternative rejected below was adopted one day later
  as spc-35's download cache with a copied PATH file; itd-132 carries the
  typed link. The paragraph that follows stays as the record of what was
  ruled on 2026-08-20.
- **The binary's home stays the plugin root.** [[itd-108]]'s one-cut
  coherence (fresh root → matching binary) is preserved structurally;
  cold-start once per plugin update is the accepted cost — the 2026-08-20
  experiments showed it cheap and reliable. The persistent-data-dir
  alternative is rejected, not deferred.
- **Verification ships at the checksums bar; in-process attestation is
  deferred.** Same-origin `checksums.txt` (the [[itd-105]] bar) gates the
  swap. In-process build-provenance verification is recorded as a future
  extension, twin to [[itd-108]]'s deferred offline signing key.
- **The swap uses `minio/selfupdate`.** Dependency sign-off given at the
  2026-08-20 interview (the AGENTS.md gate): minimal Apply-on-stream with
  checksum verification and the Windows rename-dance built in; abcd keeps
  its own release resolution and transport. `go get` lands with the
  implementation, not before.
- **The trampoline demotion is a follow-on rung, not this cut.**
  Script-first: the verb ships alone and proves itself. When the script
  delegates, the seam is hardened as specified here: delegate only to a
  binary passing an ownership check, keep the raw fetch path as the
  fallback when delegation fails, and the release-asset layout becomes a
  frozen contract — an old PATH binary provisioning a new plugin root must
  keep working across cuts.
- **No [[adr-38-implicit-checks-are-disk-only]] amendment.** The recorded
  reading: `abcd update <tag>` satisfies tier 3 literally (a named update,
  completed); bare `abcd update` is a tier-2 verb whose documented meaning
  is the fetch, and it names the resolved tag before acting (TTY
  confirmation; in the receipt always). The ADR's text is untouched.

## What's In Scope

- The fetch/verify/swap core in `internal/core` returning a structured
  report; the CLI surface renders it. The origin agreement already stated at
  `internal/core/vintage/release.go` (checker and provisioner resolve
  "latest" identically) extends to the updater.
- An explicit transport policy, specified rather than inherited: the
  updater's client ignores proxy and CA overrides from the environment (the
  seam `hooks/bootstrap.sh` scrubs — `HTTPS_PROXY`, `SSL_CERT_FILE`,
  `SSL_CERT_DIR` — because same-origin checksums are vacuous when one config
  surface supplies both payload and manifest), pins its redirect policy to
  the release origin's own asset hosts, streams the body under a size bound,
  and runs under the urlguard SSRF policy like the checker.
- Verification bar: same-origin `checksums.txt` for the resolved release,
  refusing on mismatch. The swap is atomic and same-directory via
  `minio/selfupdate`; the running binary is never overwritten in place; a
  failed download or verify leaves no partial file.
- Target selection: an explicit tag argument, or bare `abcd update`
  resolving `releases/latest` and naming the resolved tag before acting
  (TTY confirmation; named in the receipt always).
- Install-shape dispatch, keyed on what actually runs (the first PATH
  occupant / the running executable, not one blessed location): plugin-root
  → refuse, name the host's plugin-update path; dev shim → refuse, name the
  shim's contract (it tracks the source tip; `ahoy install` switches modes);
  owned dangling symlink → name `abcd ahoy install` (the iss-345 heal);
  real copy whose digest matches a published release → swap; foreign
  occupant or package-manager path → refuse with the occupant described and
  the right command printed; a shadowed entry is reported as shadowed, never
  as a completed update. Every refusal is loud and names its remedy.
- Progress on a TTY, silence otherwise; the receipt (origin, tag, digest,
  old→new) prints in both modes.

## What's Out of Scope

- Ambient discovery in any form — a background check, a session-start nudge,
  an auto-apply. Forever out, unless [[adr-38-implicit-checks-are-disk-only]]
  is superseded in the open.
- The Homebrew tap itself (parked in the decision log 2026-08-20; this verb
  ships the install-channel refusal that makes the tap safe to add later).
- The plugin surface's delivery — [[itd-108]] owns it; this verb never
  touches a plugin root.
- The trampoline demotion of `hooks/bootstrap.sh` (follow-on rung per the
  Decisions above) and the Windows cold-start shim (its own, later intent;
  this verb's core must merely be portable to it).
- In-process attestation verification (deferred per the Decisions above).

## Scope Conditions

None stated.

## Acceptance Criteria

> _Confirmed by the maintainer at the 2026-08-20 planning interview: all
> nine accepted as written._

- **Given** a pinned `~/.local/bin/abcd` copy at vN and a published release
  vN+1, **when** the user runs `abcd update`, **then** the binary is replaced
  only after its SHA-256 matches the same release's `checksums.txt`, the
  receipt names the origin, tag, digest, and old→new versions, and the exit
  code is 0.
- **Given** a checksum mismatch, **when** the update runs, **then** no file
  is replaced, the refusal names both digests, and the exit code is non-zero.
- **Given** a hostile environment (`HTTPS_PROXY`, `SSL_CERT_FILE`,
  `SSL_CERT_DIR` pointing at attacker-controlled infrastructure), **when**
  the update runs, **then** the fetch either ignores those overrides or
  refuses — it never verifies an attacker's payload against an attacker's
  checksums.
- **Given** the binary resolves into a plugin root, **when** the user runs
  `abcd update`, **then** it refuses and names the host's plugin-update path,
  and the plugin-root binary is untouched.
- **Given** the dev shim occupies the PATH entry, **when** the user runs
  `abcd update`, **then** it refuses, names the shim's track-latest contract,
  and changes nothing.
- **Given** another `abcd` precedes the updated entry on PATH, **when** the
  update completes, **then** the receipt reports the shadowing occupant
  rather than claiming the machine now runs the new version.
- **Given** no network, **when** the update runs, **then** it fails loudly,
  names exactly what was not done, and leaves no partial file anywhere.
- **Given** the zero-network test harness, **when** any verb other than
  `update` (and `version --check`) runs, **then** no release-origin request
  is observed — the ADR-38 seam extended to the new verb.
- **Given** stdout is not a TTY, **when** the update downloads, **then** no
  progress output is emitted; the receipt still prints.

## Prerequisites

- **iss-232 lands first**: `ahoy install`'s staleness refusal keys on
  BuildInfo `vcs.revision`; an updater that ever ships a differently-built
  binary flips that check for every pinned user. The guard asserting the
  shipped binary carries `vcs.revision` precedes the first `abcd update`
  release.
- **iss-345 is the detection-side companion**, staged in the same 2026-08-20
  change-set: the entry a plugin update strands classifies as abcd-owned and
  heals via `ahoy install` instead of reading as a foreign occupant.
- **[[itd-108]]'s central open question is narrowed by the 2026-08-20
  experiments recorded there** (local rig, harness v2.1.237: the archive is
  re-fetched on every plugin update in both the digest-versioned and
  declared-version shapes; the install decision is version-keyed, so a cut
  that bumps the version propagates). The real-cut confirmation itd-108
  still owes at Cut B stands; this intent's plugin-root refusal is designed
  against the confirmed shape and does not depend on that residue.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-264f7b144576 -->
Fidelity review — receipt rcp-264f7b144576 (verifier abcd:intent-auditor claude-fable-5-1).

Provenance: abcd:intent-auditor@claude-fable-5-1 · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:4bf1e64fd14d21590dfec0a2f3695e50d399893721bd5b3ec7f41dfd12f232ec
Input attestations: commit:HEAD of worktree ship-itd-130-132 = origin/main 8f68ffb3 with itd-130 moved to intents/shipped/ and spc-32 to specs/closed/@git-sha1:8f68ffb34558752618f5f1032aec356dfc22ce68; diff:delivering commits dccabb13, 413ac263, 1d9b8ed5 (2026-08-20) through PR #459 (7e00e1de) and the v0.6.9 security pass; audited as the tree at HEAD@git-tree-sha1:a94f7c23b7c4e19969b168d4dfeab2977d3341a9; request:.abcd/.work.local/reviews/rcp-264f7b144576.request.md@sha256:4bf1e64fd14d21590dfec0a2f3695e50d399893721bd5b3ec7f41dfd12f232ec; dependency:github.com/minio/selfupdate@v0.6.0 apply.go (module cache)@go.mod:6 pins v0.6.0;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 5 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: Apply fetches the tag's checksums.txt first, proves the on-disk file by provenance, then hands selfupdate.Apply the manifest digest so the stream is verified before the rename; the Report carries origin/tag/digest/old->new and the CLI returns nil (exit 0) on a swap; TestApplySwapsVerifiedBinary pins it. Concerns: (a) 'a pinned copy at vN' is provable only while vN's checksums.txt is still served AND vN is within the newest 12 atom-feed releases (provenanceN=12; a 404 is 'cannot prove'), so after the 2026-08-30 deletion of every pre-v0.6.9 release object a copy at any earlier cut is refused unprovenanced-file (the 2026-09-01 v0.6.6 refusal) — this honours the intent's own ownership Decision literally but narrows the criterion's Given; (b) the spc-35 path-entry provenance record is re-stamped after the swap but never consulted as a second proof before it; (c) no CLI-level test asserts exit 0 on the swap path and the core test does not assert Origin.
  evidence: internal/core/update/update.go:315 — "sums, found, err := u.fetchChecksums(tag)"
  evidence: internal/core/update/update.go:339 — "oldVer, proven, err := u.deriveTargetVersion(targetHex, tag, sums)"
  evidence: internal/core/update/update.go:370 — "selfupdate.Apply(reader, selfupdate.Options{TargetPath: target, Checksum: sum})"
  evidence: internal/core/update/update.go:312 — "rep := Report{Origin: u.origin, Tag: tag, Asset: u.assetName, ..."
  evidence: internal/core/update/update.go:233 — "provenanceN:  12,"
  evidence: internal/core/update/update.go:491 — "return nil, false, nil  // 404 -> found=false"
  evidence: internal/surface/cli/update.go:86 — "return nil"
  evidence: internal/surface/cli/update.go:80 — "ahoy.RefreshPathEntryDigest(tgt.Path, rep.Digest)"
  evidence: internal/core/update/update_test.go:233 — "rep.Tag != \"v0.6.2\" || rep.OldVersion != \"v0.6.1\" || rep.Digest == \"\""
  evidence: .abcd/work/DECISIONS.md:2198 — "Every release object older than v0.6.9 is deleted from the forge"
  evidence: .abcd/development/intents/shipped/itd-130-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md:95 — "A copy matching no release stays foreign and is refused"
- ac-2 — MET: selfupdate verifies the streamed bytes against the manifest digest BEFORE creating the .new file, so a mismatch writes nothing; its error names both digests ('Expected: %x, got: %x') and Apply wraps it; the CLI returns the error (non-zero exit); TestApplyChecksumMismatchLeavesTargetUntouched asserts the target is byte-identical and the directory holds only the target.
  evidence: internal/core/update/update_test.go:241 — "func TestApplyChecksumMismatchLeavesTargetUntouched"
  evidence: internal/core/update/update_test.go:258 — "entries, _ := os.ReadDir(filepath.Dir(target))"
  evidence: $GOMODCACHE/github.com/minio/selfupdate@v0.6.0/apply.go:276 — "Updated file has wrong checksum. Expected: %x, got: %x"
  evidence: $GOMODCACHE/github.com/minio/selfupdate@v0.6.0/apply.go:55 — "if err = opts.verifyChecksum(newBytes); err != nil { return err }"
  evidence: internal/core/update/update.go:374 — "verifying %s against the release manifest: %w"
  evidence: internal/surface/cli/update.go:71 — "if err != nil { return err }"
  evidence: go.mod:6 — "github.com/minio/selfupdate v0.6.0"
- ac-3 — MET: newUpdater unsets HTTPS_PROXY/HTTP_PROXY/ALL_PROXY (both cases) and SSL_CERT_FILE/SSL_CERT_DIR at construction, before the root pool is loaded and pinned, with Proxy nil by construction and redirects confined to the origin's hosts; TestApplyIgnoresProxyAndCAEnv plants an unreachable proxy and attacker CA paths and still reaches the real origin, and TestNewUpdaterScrubsCABeforeAnyFetch proves the scrub precedes the first handshake; the receipt names what was ignored.
  evidence: internal/core/update/update.go:163 — "var scrubbedEnv = []string{"
  evidence: internal/core/update/update.go:220 — "ignored := scrubEnv()"
  evidence: internal/core/update/update.go:252 — "Proxy:               nil,"
  evidence: internal/core/update/update.go:255 — "TLSClientConfig:     &tls.Config{RootCAs: rootCAs},"
  evidence: internal/core/update/update.go:264 — "if !u.redirectHost[req.URL.Host] {"
  evidence: internal/core/update/update_test.go:310 — "func TestApplyIgnoresProxyAndCAEnv"
  evidence: internal/core/update/update_test.go:391 — "func TestNewUpdaterScrubsCABeforeAnyFetch"
  evidence: internal/core/update/update_test.go:338 — "func TestApplyRefusesCrossHostRedirect"
- ac-4 — MET: An owned symlink into the plugin root classifies UpdateTargetPluginRoot (and a bare plugin invocation with an empty PATH too); Plan returns the plugin-root refusal naming '/plugin update abcd'; the CLI dispatches before the updater exists. TestUpdateRefusesPluginRootEntry drives the real CLI on a hermetic root and asserts the refusal text, zero updater constructions, and the symlink untouched.
  evidence: internal/core/update/update.go:115 — "case ahoy.UpdateTargetPluginRoot:"
  evidence: internal/core/update/update.go:119 — "Remedy: \"take a plugin update in the host (e.g. /plugin update abcd)\""
  evidence: internal/surface/cli/update.go:45 — "if r := update.Plan(tgt); r != nil {"
  evidence: internal/core/ahoy/update_target.go:82 — "case first.kind == binTargetOwnedSymlink:"
  evidence: internal/core/ahoy/update_target.go:56 — "if filepath.Dir(resolvePath(exe)) == resolvePath(pluginRoot) {"
  evidence: internal/surface/cli/update_test.go:57 — "func TestUpdateRefusesPluginRootEntry"
  evidence: internal/surface/cli/update_test.go:100 — "the refusal touched the entry"
  evidence: internal/core/ahoy/update_target_test.go:13 — "func TestUpdateTargetPluginRootSymlink"
- ac-5 — MET: A dev shim at the first PATH entry classifies UpdateTargetDevShim through ahoy's own classifier (TestUpdateTargetDevShim renders the real shim); Plan refuses naming the track-latest contract ('rebuilds abcd from the source tip on every call') and the mode switch, and the CLI returns before any updater or write exists.
  evidence: internal/core/update/update.go:121 — "case ahoy.UpdateTargetDevShim:"
  evidence: internal/core/update/update.go:124 — "is the track-latest dev shim: it rebuilds abcd from the source tip on every call"
  evidence: internal/core/update/update.go:125 — "switch modes first: `abcd ahoy install` re-pins the entry"
  evidence: internal/core/ahoy/update_target.go:78 — "case first.kind == binTargetDevShim:"
  evidence: internal/core/ahoy/update_target_test.go:28 — "func TestUpdateTargetDevShim"
  evidence: internal/core/update/update_test.go:26 — "func TestPlanRefusesDevShim"
  evidence: internal/surface/cli/update.go:42 — "Dispatch first, network never"
- ac-6 — MET_WITH_CONCERNS: The resolver records an abcd-owned entry shadowed behind the first occupant (LaterOwned) and the foreign-symlink refusal names it ('a working abcd install sits shadowed behind it at ...'); TestPlanRefusesForeignAndNamesShadowedInstall and TestUpdateTargetReportsLaterOwnedEntry pin the two halves. Concerns: only the FIRST PATH occupant is ever the target, so an update never 'completes' on a shadowed entry — the false 'machine now runs vN+1' claim is unreachable by construction rather than reported; spc-32's row 'proceed on the owned entry but the receipt reports the shadowing occupant' was not delivered (it refuses instead); and when the first occupant is an unprovenanced REGULAR FILE (exactly the fixture the ahoy test builds) Plan returns nil, LaterOwned is dropped, and Apply's unprovenanced-file refusal never mentions the shadowed install.
  evidence: internal/core/ahoy/update_target.go:70 — "for _, e := range entries[1:] { if e.owned() && !e.dangling { tgt.LaterOwned = e.path"
  evidence: internal/core/update/update.go:135 — "if t.LaterOwned != \"\" { detail += \"; a working abcd install sits shadowed behind it at \""
  evidence: internal/core/update/update.go:139 — "case ahoy.UpdateTargetFile: ... return nil"
  evidence: internal/core/update/update.go:345 — "rep.Refusal = &Refusal{ Shape: \"unprovenanced-file\""
  evidence: internal/core/update/update_test.go:40 — "func TestPlanRefusesForeignAndNamesShadowedInstall"
  evidence: internal/core/ahoy/update_target_test.go:110 — "func TestUpdateTargetReportsLaterOwnedEntry"
  evidence: .abcd/development/specs/closed/spc-32-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md:61 — "proceed on the owned entry but the receipt reports the shadowing occupant"
- ac-7 — MET_WITH_CONCERNS: Every failure surfaces as an error naming the step not done ('could not resolve the latest release', 'fetching <tag> checksums', 'downloading <tag> <asset>'), and TestApplyNoNetworkFailsLoudNoPartialFile asserts an unreachable origin errors and leaves only the target in its directory; selfupdate reads the whole body into memory before creating '.<name>.new' and verifies the checksum before that open, so a mid-download truncation or mismatch creates no file. Concerns: the only pinned no-network case fails before any download begins; a failure in selfupdate's io.Copy to the .new file (disk full, permission) leaves '.abcd.new' behind with no unlink, and spc-32's promised 'download failure unlinks the temp file (defer-cleanup asserted by test)' has no test.
  evidence: internal/core/update/update_test.go:293 — "func TestApplyNoNetworkFailsLoudNoPartialFile"
  evidence: internal/core/update/update_test.go:302 — "entries, _ := os.ReadDir(filepath.Dir(target))"
  evidence: internal/core/update/update.go:293 — "could not resolve the latest release: %w"
  evidence: internal/core/update/update.go:317 — "fetching %s checksums: %w"
  evidence: internal/core/update/update.go:360 — "downloading %s %s: %w"
  evidence: $GOMODCACHE/github.com/minio/selfupdate@v0.6.0/apply.go:48 — "if newBytes, err = ioutil.ReadAll(update); err != nil {"
  evidence: $GOMODCACHE/github.com/minio/selfupdate@v0.6.0/apply.go:71 — "newPath := filepath.Join(updateDir, fmt.Sprintf(\".%s.new\", filename))"
  evidence: $GOMODCACHE/github.com/minio/selfupdate@v0.6.0/apply.go:78 — "_, err = io.Copy(fp, bytes.NewReader(newBytes))"
  evidence: .abcd/development/specs/closed/spc-32-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md:87 — "download failure unlinks the temp file (defer-cleanup asserted by test walking the target dir)"
- ac-8 — MET_WITH_CONCERNS: The network-capable updater is reachable only through the package var newUpdater in cli/update.go, whose sole call site is the update verb after dispatch; TestUpdateSeamUntouchedByOtherVerbs counts zero constructions across `version` and `rules`, the refusal tests count zero on the refusal paths, and the pre-existing TestOnlyVersionCheckTouchesTheNetwork still holds the release-fetcher seam to exactly one fetch. Concern: the extended harness enumerates two verbs (plus the three implicit paths of the older invariant), not the whole cobra tree — the guarantee for every other verb rests on the single-caller structure, not on an observed run.
  evidence: internal/surface/cli/update.go:23 — "var newUpdater = update.NewGitHubUpdater"
  evidence: internal/surface/cli/update.go:51 — "u := newUpdater()"
  evidence: internal/surface/cli/update_test.go:107 — "func TestUpdateSeamUntouchedByOtherVerbs"
  evidence: internal/surface/cli/update_test.go:113 — "for _, args := range [][]string{{\"version\"}, {\"rules\"}} {"
  evidence: internal/surface/cli/update_test.go:50 — "if calls != 0 {"
  evidence: internal/surface/cli/version.go:21 — "var newReleaseFetcher = vintage.NewGitHubReleaseFetcher"
  evidence: internal/surface/cli/version_check_test.go:23 — "func TestOnlyVersionCheckTouchesTheNetwork"
- ac-9 — MET_WITH_CONCERNS: Progress is attached only when stderr is a terminal and is written to stderr; the receipt is rendered to stdout unconditionally in both text and --json modes, so a piped stdout never carries a progress byte. Concerns: the gate is stderr's TTY-ness, not stdout's as the criterion states — with stdout piped and stderr a terminal, progress still renders (on stderr); and no test asserts non-TTY silence, although spc-32 promised 'non-TTY test asserts stdout carries only the receipt' (no cli test references progress or isTTY).
  evidence: internal/surface/cli/update.go:66 — "var progress io.Writer; if isTTY(os.Stderr) { progress = cmd.ErrOrStderr() }"
  evidence: internal/surface/cli/update.go:82 — "renderUpdateReport(cmd.OutOrStdout(), *asJSON, rep)"
  evidence: internal/core/update/update.go:364 — "if progress != nil { reader = &progressReader{"
  evidence: internal/core/update/update.go:606 — "fmt.Fprintf(p.out, \"\\r  downloading %s: %3d%%\", p.label, pct)"
  evidence: .abcd/development/specs/closed/spc-32-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md:93 — "Non-TTY silence — progress writer selected on term.IsTerminal; non-TTY test asserts stdout carries only the receipt"
  evidence: internal/surface/cli/update_test.go:1 — "package cli  // no test in this file references progress, isTTY or IsTerminal"

Gap audit:
- honoured:
  - One verb completes a chosen update: fetch, verify against the same release's checksums.txt, atomic swap, receipt — core in internal/core/update returning a structured Report, CLI renders it
    evidence: internal/core/update/update.go:308 — "func (u *Updater) Apply(target, tag string, progress io.Writer) (Report, error)"
    evidence: internal/surface/cli/update.go:111 — "func renderUpdateReport(w io.Writer, asJSON bool, rep update.Report)"
  - Pinned transport: no proxy, no env CA overrides, redirects only onto the origin's asset hosts re-checked under urlguard, bounded bodies
    evidence: internal/core/update/update.go:247 — "u.client = &http.Client{ ... Proxy: nil"
    evidence: internal/core/update/update.go:202 — "u.redirectHost[\"objects.githubusercontent.com\"] = true"
    evidence: internal/core/update/update.go:539 — "if resp.ContentLength > limit {"
  - Install-shape dispatch keyed on the first PATH occupant, every refusal loud with a named remedy (plugin-root, dev-shim, owned-dangling, foreign, absent, package-manager, unclassified)
    evidence: internal/core/update/update.go:89 — "func Plan(t ahoy.UpdateTarget) *Refusal"
    evidence: internal/core/ahoy/update_target.go:50 — "func ResolveUpdateTarget() UpdateTarget"
  - Homebrew refusal by prefix test on the RESOLVED path, never a substring match
    evidence: internal/core/update/update.go:75 — "var brewCellarPrefixes = []string{"
    evidence: internal/core/update/update_test.go:60 — "func TestPlanRefusesBrewCellarPath"
  - adr-38 posture untouched: no ambient check, the verb is the only ask; bare `abcd update` names the resolved tag before acting (TTY confirmation, receipt always)
    evidence: internal/surface/cli/update.go:58 — "if requested == \"\" && !yes && isTTY(os.Stdin) && isTTY(os.Stderr) {"
    evidence: internal/surface/cli/update.go:59 — "resolved latest release: %s — proceed? [y/N]"
  - The swap uses minio/selfupdate (signed-off dependency), verification before rename with rollback reporting
    evidence: go.mod:6 — "github.com/minio/selfupdate v0.6.0"
    evidence: internal/core/update/update.go:371 — "if rerr := selfupdate.RollbackError(err); rerr != nil {"
  - Wired on both surfaces: the plugin command and the CLI reference document the verb
    evidence: commands/update.md:19 — "\"${CLAUDE_PLUGIN_ROOT}/abcd\" update --yes --json"
    evidence: docs/reference/cli/commands.md:911 — "### `abcd update`"
  - Old version is DERIVED from the release the on-disk bytes belong to, never read from the running binary
    evidence: internal/core/update/update.go:393 — "func (u *Updater) deriveTargetVersion(targetHex, installTag string, installSums map[string]string)"
    evidence: internal/core/update/update_test.go:361 — "func TestApplyDerivesOldVersionFromAnOlderRelease"
- diverged:
  - 'The persistent-data-dir alternative is rejected, not deferred' (itd-130 Decisions) — itd-132/spc-35 later adopted the persistent data dir as the download cache and made the PATH entry an abcd-owned copied regular file with a recorded provenance file; itd-130's text was never revised and its Decision now misstates the delivered architecture
    evidence: .abcd/development/intents/shipped/itd-130-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md:100 — "The persistent-data-dir alternative is rejected, not deferred."
    evidence: .abcd/development/intents/shipped/itd-132-hook-binary-to-persistent-data-dir.md:48 — "The data dir is a **download cache, not an execution home**"
    evidence: internal/core/ahoy/owned_copy.go:14 — "The PATH entry as an abcd-owned regular file (spc-35)."
  - Ownership by provenance is narrower than 'a pinned copy at vN': provable only while vN's checksums.txt is still published and vN sits within the newest 12 feed entries; the 2026-08-30 deletion of every pre-v0.6.9 release object makes every older pinned copy an unprovenanced-file refusal
    evidence: internal/core/update/update.go:233 — "provenanceN:  12,"
    evidence: internal/core/update/update.go:485 — "A missing release (404) is found=false, not an error: provenance treats it as \"cannot prove\""
    evidence: .abcd/work/DECISIONS.md:2198 — "v0.1.0, v0.2.0, v0.3.0, v0.4.2, v0.5.1, v0.6.7, v0.6.8; the v0.6.0–v0.6.6 tags never had one"
  - Shadowed entry: spc-32 promised to proceed on the owned entry and annotate the receipt; delivered is a refusal on the foreign first occupant that names the shadowed install, and the unprovenanced-regular-file shape drops the shadow entirely
    evidence: .abcd/development/specs/closed/spc-32-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md:61 — "proceed on the owned entry but the receipt reports the shadowing occupant"
    evidence: internal/core/update/update.go:133 — "case ahoy.UpdateTargetForeign:"
    evidence: internal/core/update/update.go:139 — "case ahoy.UpdateTargetFile: ... return nil"
  - 'Now the next line is abcd update' / 'eight error messages ... now there is a verb they can name': the verb exists, but `version --check`'s verdict and the schema-too-new error sites still say only 'update available' / 'upgrade abcd' and never name `abcd update`
    evidence: internal/surface/cli/version.go:122 — "res.Verdict = \"update available: \" + currentVersion + \" -> \" + res.Latest"
    evidence: internal/core/lifeboat/embark.go:203 — "this abcd knows up to v%d — upgrade abcd"
    evidence: internal/core/release/ingest.go:312 — "this abcd knows up to v%d — upgrade abcd"
  - Progress silence is keyed on stderr being a TTY, not on stdout as promised; stdout-piped/stderr-terminal invocations still show progress on stderr
    evidence: internal/surface/cli/update.go:67 — "if isTTY(os.Stderr) {"
    evidence: .abcd/development/intents/shipped/itd-130-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md:205 — "**Given** stdout is not a TTY, **when** the update downloads, **then** no progress output is emitted"
- missing:
  - A non-TTY silence test ('non-TTY test asserts stdout carries only the receipt') — no cli test references progress or the TTY gate
    evidence: .abcd/development/specs/closed/spc-32-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md:93 — "non-TTY test asserts stdout carries only the receipt"
    evidence: internal/surface/cli/update_test.go:16 — "func runUpdateHermetic(t *testing.T, args ...string) (string, error, int)  // drives refusals only"
  - A partial-file test for a failure AFTER the download starts (mid-stream truncation or a write failure into '.abcd.new'); the delivered no-network test fails before any download begins and selfupdate does not unlink .new on a copy failure
    evidence: .abcd/development/specs/closed/spc-32-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md:87 — "download failure unlinks the temp file (defer-cleanup asserted by test walking the target dir)"
    evidence: internal/core/update/update_test.go:295 — "o.srv.Close() // the origin is unreachable"
    evidence: $GOMODCACHE/github.com/minio/selfupdate@v0.6.0/apply.go:78 — "_, err = io.Copy(fp, bytes.NewReader(newBytes))"
  - The CA-canary assertion spc-32 promised ('asserted via a canary file whose read would be observable') — the tests assert the env is unset, not that a planted CA file is never read
    evidence: .abcd/development/specs/closed/spc-32-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md:78 — "asserted via a canary file whose read would be observable"
    evidence: internal/core/update/update_test.go:396 — "fetcher := envRecordingFetcher{onCall: func() {"