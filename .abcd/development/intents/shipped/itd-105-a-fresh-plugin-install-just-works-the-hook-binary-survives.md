---
id: itd-105
slug: a-fresh-plugin-install-just-works-the-hook-binary-survives
spec_id: spc-21
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: [itd-67]
severity: critical
impact: additive
grill_session_id: f050596a-e4fe-49e4-92b3-38f90604b3fb
grilled_at: 2026-07-29
warrants_assumed:
- "GitHub releases remain the sole binary distribution channel; every release publishes abcd-<os>-<arch> plus checksums.txt over the same bytes."
- "Hook commands run under POSIX sh on macOS and Linux with curl and shasum/sha256sum available."
- "The harness re-clones each plugin update into a fresh commit-stamped cache directory; nothing persists across that boundary except what the bootstrap re-creates."
- "Same-origin checksum verification (HTTPS + checksums.txt from the same release) is the accepted default trust bar; full trust is building from source, documented as the escape hatch — not signature infrastructure."
---

# A fresh plugin install just works: the hook binary survives install and every update

## Press Release

> **Installing the abcd plugin from its marketplace now yields a working hook
> surface on the very first session — and keeps working across every plugin
> update.** Today the hooks call `$CLAUDE_PLUGIN_ROOT/abcd`, but the plugin
> root is a git clone of the repo, and the repo (correctly) commits no binary:
> every fresh install and every update produces a plugin root with no `abcd`
> in it, so every hook fails with "No such file or directory" until someone
> hand-copies a binary in. With this intent, the gap closes itself: a
> committed POSIX-sh bootstrap runs at session start, and when the plugin-root
> binary is missing it downloads the latest release binary for the host
> OS/arch, verifies its SHA-256 against the release checksum manifest, and
> installs it into the plugin root — refusing on any mismatch. An update never
> reintroduces the gap, because the bootstrap heals each fresh commit-stamped
> directory the same way. Nobody clones the repo to use abcd: the binary is
> all a user needs, and the surface now fetches it itself.

> "I installed the plugin exactly as the README says, opened a session in a
> clean repo, and got two hook errors before I typed anything," said Alice,
> trying abcd on a second machine. "And when I turned on auto-update so a
> teammate's fixes would reach me, every update broke it again. Now the first
> session after install just works, and so does the first session after every
> update — I never think about the binary at all."

## Why This Matters

The distribution story ([[itd-67-installable-versioned-plugin]]) made the repo
its own marketplace, and the README documents both halves: install the
checksum-verified release binary to PATH, and install the plugin from the
marketplace. But the two halves do not meet. `hooks/hooks.json` invokes
`"$CLAUDE_PLUGIN_ROOT/abcd"` — a path inside the harness's plugin cache — while
the PATH install puts the binary in `/usr/local/bin`. Nothing in the repo,
the payload, or the `ahoy` machinery provisions the plugin-root binary: `ahoy
install` only creates the PATH symlink *pointing at* `$pluginRoot/abcd`,
assuming it already exists. A fresh marketplace install therefore fails every
hook (`SessionStart`, `UserPromptSubmit`, the Bash guard) in every session,
and the guard failure means shell commands run unguarded — the surface
degrades exactly where it claims to protect.

The update path makes this structural rather than one-off. The harness caches
each installed plugin version in a directory stamped with the source commit
(`cache/<marketplace>/abcd/<sha>/`). Every update — manual or auto — is a
fresh clone into a fresh directory, which starts with no binary; a hand-copied
fix evaporates on the next update. A maintainer who enables marketplace
auto-update precisely so that merged fixes flow to their machines gets the
opposite: every merge re-breaks the hook surface. There is also a version-skew
corollary: the plugin surface (hooks, commands, agents) tracks the repository
tip, while the newest binary is the last tagged release — fixes that merge
without a release cut can leave the surface ahead of any binary a user can
download ([[itd-73-derived-versioning]], [[itd-69-plugin-metadata-lockstep-update]]).

## Decisions (grilled 2026-07-29)

The maintainer resolved the shape questions in a grill session; these are
commitments, not options:

- **Shape: bootstrap-on-missing.** A committed POSIX-sh script under `hooks/`
  (it must need no binary to run — the binary is exactly what is missing) is
  wired as the first `SessionStart` hook. Hook commands keep calling
  `$CLAUDE_PLUGIN_ROOT/abcd` unchanged. No PATH fallback in the hook commands.
- **Version: latest release, always.** The bootstrap fetches
  `releases/latest`. No user should ever need the repo — the binary is all
  they need. Surface-newer-than-binary skew is surfaced, not prevented
  (prevention belongs to [[itd-73-derived-versioning]]).
- **Trust bar: same-origin checksums, build-from-source as full trust.**
  SHA-256 against the release `checksums.txt`, refuse on mismatch or a
  missing manifest entry — identical to the README one-liner's bar. Users
  wanting a stronger root build from source (`go build ./cmd/abcd`) and place
  the binary themselves; the docs say so explicitly.
- **Guard window: fail open, loudly.** While no binary exists (offline first
  session, checksum refusal, unsupported platform) the Bash guard keeps its
  current fail-open wrapper: shell commands run, each carrying the UNGUARDED
  warning. No blocking, no escape-hatch machinery.
- **Failure UX: every hook reports, for now.** During the no-binary window
  each hook failure prints the actionable plain-language message. This is a
  deliberate testing posture; the recorded aspiration is one notice per
  session with the other hooks silent (tracked as a follow-up, alongside the
  status-line visibility idea in iss-168).
- **Platforms: darwin/linux × amd64/arm64 only.** On any other platform the
  bootstrap says so in plain language and stops; Windows support is its own
  future intent, not this one.
- **PATH: deferred to ahoy.** The bootstrap provisions the plugin root only
  (no sudo in hooks). Terminal CLI availability keeps its existing owner: the
  session-start surface suggests `ahoy install` once, whose existing symlink
  logic manages `/usr/local/bin/abcd`.

## What's In Scope

- The committed `hooks/` bootstrap script: OS/arch detection, download of the
  latest release binary + `checksums.txt`, SHA-256 verification (refusing on
  mismatch or an absent manifest entry), atomic install into
  `$CLAUDE_PLUGIN_ROOT/abcd`, and a no-op fast path when the binary is
  already present and executable.
- `hooks/hooks.json` wiring that runs the bootstrap before the binary-backed
  `SessionStart` hooks.
- Update survival: the same bootstrap heals a fresh commit-stamped cache
  directory after every plugin update with no manual step.
- Failure honesty: while the binary is genuinely unavailable, hooks fail with
  the actionable plain-language message (never a raw shell "No such file or
  directory"), and `abcd ahoy` guard health reports the gap.
- Version-skew visibility: when the installed surface (plugin commit) is newer
  than the newest published binary, the session-start surface says so rather
  than silently running old-binary/new-surface.
- README plugin section documents the contract: the plugin self-provisions its
  binary; the PATH install remains for terminal CLI use; build-from-source is
  the full-trust path.

## What's Out of Scope

- The release/versioning pipeline itself — how versions are derived, bumped,
  tagged, and published stays with [[itd-73-derived-versioning]] and
  [[itd-69-plugin-metadata-lockstep-update]]; this intent only consumes
  published release artefacts.
- Marketplace auto-update policy — whether a user enables the harness's
  auto-update is their choice; this intent makes either answer safe, it does
  not toggle harness settings.
- Signature infrastructure (minisign/cosign) — the accepted trust bar is
  same-origin checksums plus the documented build-from-source escape hatch.
- Windows binaries and Windows hook semantics — a future intent.
- The one-notice-per-session failure UX and harness status-line visibility —
  recorded aspirations (iss-168), follow-up work once the loud posture has
  been tested.
- Any change to the transport-agnostic core's behaviour — this is a
  distribution/wiring concern at the plugin surface; the bootstrap is shell,
  not core.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a machine that has never installed abcd, **when** a user adds the
  marketplace, installs the plugin, and starts a session in any repo with
  network available, **then** the bootstrap installs the checksum-verified
  latest release binary into the plugin root and every hook (`SessionStart`,
  `UserPromptSubmit`, `PreToolUse` guard, `PreCompact`, `SessionEnd`)
  executes successfully — no "No such file or directory", no unguarded-shell
  warning.
- **Given** a working install, **when** the plugin updates to a new commit
  (fresh cache directory), **then** the first session after the update has a
  working hook surface with no manual step.
- **Given** a downloaded binary whose SHA-256 does not match the release
  `checksums.txt` (or a manifest that lacks the platform entry), **then**
  nothing is installed, the message names the mismatch, and a corrupted or
  unpublished artefact never runs.
- **Given** the binary is genuinely unavailable (offline first session,
  refused checksum, unsupported platform), **when** hooks fire, **then** each
  failure prints the actionable plain-language message naming the recovery
  step, the Bash guard fails open with its UNGUARDED warning, and `abcd ahoy`
  reports the binary gap in guard health.
- **Given** an unsupported platform, **when** the bootstrap runs, **then** it
  states in plain language that the platform has no released binary and what
  the supported matrix is — and changes nothing.
- **Given** an installed surface newer than the newest published binary,
  **when** a session starts, **then** the skew is surfaced to the user rather
  than silently running a stale binary against newer surface expectations.
- **Given** a plugin root whose binary is already present and executable,
  **when** the bootstrap runs at session start, **then** it exits without
  network access — steady-state sessions pay no download and no latency
  beyond a file test.

## Open Questions

_None — resolved in the 2026-07-29 grill session; see Decisions. Follow-ups
are recorded as scope exclusions and ledger entries (iss-168), not open
questions._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-bba9a75a2d1d -->
Fidelity review — receipt rcp-bba9a75a2d1d (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:e93232730943fbe0c129bd039144c3d868df37501b9d015c76a6fc7c4662d728
Input attestations: diff:tree at 7c476185 (main lineage, itd-105 shipped)@-;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: hooks/bootstrap.sh resolves releases/latest over https-only curl, fetches the asset and checksums.txt from the same release, verifies SHA-256 and installs atomically; SessionStart runs it first and the UserPromptSubmit, PreToolUse and PreCompact hooks salvage a missing binary before calling it; TestBootstrapInstallsVerifiedBinary, TestBootstrapFreshInstallSelfCheck and TestBinaryHooksProvisionWhenTheBinaryIsAbsent are green at BASE
  evidence: hooks/bootstrap.sh:568 — "redirect=$(curl -q -fsS --proto '=https' --proto-redir '=https' --max-time 15 -o /dev/null -w '%{redirect_url}' "$releases_url/latest" 2>/dev/null) || redirect=''"
  evidence: hooks/bootstrap.sh:612 — "curl -q -fsSL --proto '=https' --proto-redir '=https' --max-time 30 -o "$auth_tmp/checksums.txt""
  evidence: hooks/hooks.json:13 — ""SessionStart": ["
  evidence: internal/surface/cli/bootstrap_test.go:408 — "func TestBootstrapInstallsVerifiedBinary(t *testing.T) {"
  evidence: internal/surface/cli/bootstrap_freshinstall_test.go:103 — "func TestBootstrapFreshInstallSelfCheck(t *testing.T) {"
  evidence: internal/surface/cli/hooks_selfprovision_test.go:183 — "func TestBinaryHooksProvisionWhenTheBinaryIsAbsent(t *testing.T) {"
- ac-2 — MET: a fresh commit-stamped plugin root is provisioned by the same bootstrap, now from the user-level cache without a download when the cached artefact authenticates; TestBootstrapSecondRootIsProvisionedFromCache and TestBootstrapProvisionsRootFromCacheWithoutDownload model the update and are green
  evidence: internal/surface/cli/bootstrap_cache_test.go:190 — "func TestBootstrapSecondRootIsProvisionedFromCache(t *testing.T) {"
  evidence: internal/surface/cli/bootstrap_cache_test.go:103 — "func TestBootstrapProvisionsRootFromCacheWithoutDownload(t *testing.T) {"
  evidence: internal/surface/cli/hooks_sessionstart_test.go:211 — "func TestSessionStartLeadsWithTheBootstrapSuccess(t *testing.T) {"
- ac-3 — MET: a mismatch and an absent manifest entry each refuse with a message naming the cause and install nothing; TestBootstrapRefusesChecksumMismatch, TestBootstrapRefusesAbsentManifestEntry and TestBootstrapFailsLoudlyAndLeavesNothingHalfInstalled are green
  evidence: hooks/bootstrap.sh:700 — "refuse "the downloaded $asset does not match its SHA-256 checksum in the release checksums.txt — the artefact is corrupted or is not the published one, so nothing was installed""
  evidence: hooks/bootstrap.sh:690 — "refuse "the release checksums.txt lists no entry for $asset, so the download cannot be verified and is not installed""
  evidence: internal/surface/cli/bootstrap_test.go:639 — "func TestBootstrapRefusesChecksumMismatch(t *testing.T) {"
  evidence: internal/surface/cli/bootstrap_test.go:656 — "func TestBootstrapRefusesAbsentManifestEntry(t *testing.T) {"
  evidence: internal/surface/cli/bootstrap_freshinstall_test.go:179 — "func TestBootstrapFailsLoudlyAndLeavesNothingHalfInstalled(t *testing.T) {"
- ac-4 — MET: every binary-backed hook in hooks.json prints a plain-language message naming the recovery when the binary is missing, the guard hook exits 1 with the UNGUARDED warning, the bootstrap's own refusal names three fixes, and ahoy's guard health reports the missing plugin-root binary; TestBinaryHooksSayWhatIsDegradedWhenUnprovisionable, TestSessionStartReportsAMissingBinaryOnce and TestGuardHealthBinaryUnreachable are green
  evidence: hooks/hooks.json:30 — "abcd guard: the plugin binary is missing, so shell commands run UNGUARDED until it is provisioned"
  evidence: hooks/bootstrap.sh:207 — "The abcd binary is not installed in the plugin root, so the abcd hooks cannot run and the shell-hazard guard is inactive"
  evidence: internal/core/ahoy/guard_health.go:89 — "the plugin-root abcd binary is missing or not executable, so the hook shim fails open on every command"
  evidence: internal/surface/cli/hooks_selfprovision_test.go:204 — "func TestBinaryHooksSayWhatIsDegradedWhenUnprovisionable(t *testing.T) {"
  evidence: internal/surface/cli/hooks_sessionstart_test.go:307 — "func TestSessionStartReportsAMissingBinaryOnce(t *testing.T) {"
  evidence: internal/core/ahoy/guard_health_test.go:95 — "func TestGuardHealthBinaryUnreachable(t *testing.T) {"
- ac-5 — MET: the platform gate maps uname to darwin/linux on amd64/arm64 and any other platform gets a notice naming the platform, the supported matrix and the build-from-source route, changing nothing; TestBootstrapUnsupportedPlatformChangesNothing is green
  evidence: hooks/bootstrap.sh:426 — "no abcd binary is released for this platform (%s %s). Released binaries cover darwin and linux on amd64 and arm64 only, so nothing was downloaded and nothing was changed."
  evidence: internal/surface/cli/bootstrap_test.go:710 — "func TestBootstrapUnsupportedPlatformChangesNothing(t *testing.T) {"
- ac-6 — MET_WITH_CONCERNS: hook session-start prints binarySkewNotice when the live plugin root's commit and the cached binary's release commit differ, so the skew is surfaced; the concern is that the notice is deliberately non-directional (it names both directions and claims neither) and stays silent when either commit did not resolve, so the criterion's 'surface newer than binary' is reported as 'at different commits'; TestHookSessionStartReportsBinarySkew and TestHookSessionStartSkewComparesTheLiveRoot are green
  evidence: internal/surface/cli/cli.go:1628 — "if n := binarySkewNotice(); n != "" {"
  evidence: internal/surface/cli/skew.go:71 — "the two are at different commits, so the surface may be running ahead of the last release, or the binary ahead of this plugin."
  evidence: internal/surface/cli/skew.go:56 — "if !resolvedSHA(pluginSHA) || !resolvedSHA(releaseSHA) || pluginSHA == releaseSHA {"
  evidence: internal/surface/cli/skew_test.go:53 — "func TestHookSessionStartReportsBinarySkew(t *testing.T) {"
- ac-7 — MET: the fast path is a regular-file-and-executable test before any network step, and the hooks skip the salvage entirely when the binary is present; TestBootstrapFastPathTouchesNoNetwork, TestBootstrapFastPathRejectsADirectory and TestBinaryHooksSteadyStateBypassesTheSalvage are green
  evidence: hooks/bootstrap.sh:302 — "if [ -f "$binary" ] && [ -x "$binary" ]; then"
  evidence: internal/surface/cli/bootstrap_test.go:381 — "func TestBootstrapFastPathTouchesNoNetwork(t *testing.T) {"
  evidence: internal/surface/cli/hooks_selfprovision_test.go:251 — "func TestBinaryHooksSteadyStateBypassesTheSalvage(t *testing.T) {"

Gap audit:
- honoured:
  - bootstrap-on-missing: a committed POSIX-sh script needing no binary, wired first in SessionStart, hook commands still calling $CLAUDE_PLUGIN_ROOT/abcd
    evidence: hooks/hooks.json:13 — ""SessionStart": ["
    evidence: internal/surface/cli/hooks_sessionstart_test.go:185 — "func TestSessionStartIsOneChainedCommand(t *testing.T) {"
  - same-origin checksums as the trust bar, curl pinned to https and its own config surface ignored
    evidence: hooks/bootstrap.sh:151 — "So: -q first on every call below"
    evidence: internal/surface/cli/bootstrap_test.go:973 — "func TestBootstrapIgnoresCurlsOwnConfigSurface(t *testing.T) {"
  - guard window fails open, loudly, with the UNGUARDED warning
    evidence: internal/surface/cli/hooks_selfprovision_test.go:296 — "func TestGuardHookKeepsItsExitCodeFence(t *testing.T) {"
  - build-from-source documented as the full-trust path
    evidence: docs/how-to/install.md:151 — "For a stronger root of trust than same-origin checksums, build from source —"
- diverged:
  - In Scope says the README plugin section documents the self-provisioning contract; the README's plugin section describes the marketplace install and the SHA-256-named plugin archive and defers to the install guide, which is where the bootstrap contract and the build-from-source path are documented
    evidence: README.md:109 — "The [install guide] (docs/how-to/install.md) covers building from source and what to do when `abcd` isn't found afterwards."
    evidence: docs/how-to/install.md:45 — "[`hooks/bootstrap.sh`] (https://github.com/intentdriven/abcd/blob/main/hooks/bootstrap.sh) runs"
  - the hook commands carry a PATH rung after the plugin root (a recorded-and-owned PATH binary), where the intent's decision says no PATH fallback in the hook commands; SessionStart alone keeps the no-PATH shape
    evidence: internal/surface/cli/hooks_selfprovision_test.go:475 — "func TestBinaryHooksRunAnOwnedPathBinary(t *testing.T) {"
    evidence: internal/surface/cli/hooks_selfprovision_test.go:540 — "func TestSessionStartHasNoPathRungAndFailsClosed(t *testing.T) {"
  - the skew notice is non-directional by design, where the intent promises to surface a surface-newer-than-binary skew
    evidence: internal/surface/cli/skew.go:66 — "Non-directional on purpose: comparing two commits establishes that they"
- missing: (none)
