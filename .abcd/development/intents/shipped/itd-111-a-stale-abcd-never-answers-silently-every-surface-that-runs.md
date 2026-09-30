---
id: itd-111
slug: a-stale-abcd-never-answers-silently-every-surface-that-runs
spec_id: spc-22
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-108]
severity: minor
impact: additive
shipped_in: v0.5.0
---

# A Stale abcd Never Answers Silently

## Press Release

> **Every abcd surface now knows its own vintage and says so.** The binary
> carries its build revision; at session start the plugin compares it against
> what should be running — the source tip in a dogfood checkout, the
> plugin manifest's pinned version everywhere else — and a mismatch is
> announced with the one-command fix, never discovered a session later
> through misbehaviour. `abcd version` and `abcd ahoy` always show install
> mode and vintage. And the one verb where stale logic writes state refuses
> outright: `ahoy install` run through a binary older than its own source
> declines to touch the machine until the binary is rebuilt. abcd never asks
> the network "what's new?" on its own: implicit checks read only what is on
> disk, the network answers only an explicit `--check`, and when a plugin
> update names a new binary version, provisioning fetches exactly that
> pinned, checksum-verified artifact — completing an update the user already
> chose, not phoning home.
>
> "A month-stale binary spent a morning confidently applying month-stale
> install logic at my machine," said Alice, a maintainer. "Now the session
> opens by telling me the binary is behind the tip, and the install verb
> won't even run through it."

## Why This Matters

The 2026-08-15 install session is the evidence (iss-228): the repo-root
plugin binary predated the no-sudo install work, so it targeted root-owned
`/usr/local/bin`, rejected the `--bin-dir` flag its own skill documentation
described (iss-225/226 — the docs were current, the binary was not), and sent
a whole session into detective work before `make build` revealed the trivial
root cause. Nothing in the system knew — or could say — that the binary was a
month behind the surface fronting it. The failure class is invisible by
construction: a stale binary behaves plausibly, just wrongly, and the newer
the documentation the more misleading the combination. iss-227 (error paths
that report partial with no note) compounds it: silence on top of silence.

Detection is cheap and already half-present — Go stamps the VCS revision into
the binary, the checkout knows its tip, the plugin cache knows its pinned
version. What is missing is only the comparison and the refusal to stay quiet.

## Design Decisions (grilled 2026-08-15)

1. **Separate intent, detection-only.** itd-108 ships a distribution channel;
   this ships a standing invariant across both channels. itd-111 never
   writes and never heals — rebuild is the dev shim's job, re-download is
   itd-105/108's provisioning job. It detects and refuses to be silent.
2. **The network trichotomy is a system-wide trust rule, extracted.** The
   itd-84 decomposition at planning (2026-08-15) routed it out of this intent:
   the rule now lives as
   [adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md) and
   brief invariant 7, which this intent cites and implements — implicit
   checks disk-only; the network answers only an explicit ask; provisioning
   fetches only the manifest-named, checksum-verified version.
3. **Warning surfaces.** SessionStart notice (the existing gap-notice
   channel) names the stale binary, its vintage, and the one-command fix.
   `abcd version` and `abcd ahoy` always print install mode + vintage.
   Ordinary verbs stay silent — with one deliberate exception: `ahoy
   install` run through a stale-against-tip binary refuses with the mismatch
   named and writes nothing, because stale install logic mutating the
   machine is the trap the evidence session actually fell into.
4. **Doc/binary version-match is a corollary, not a mechanism.** The plugin
   ships docs and binary as one unit, so drift exists only where vintage
   drift exists (dev checkouts, failed provisioning); the vintage warning is
   the doc-drift warning. No separate doc-version machinery.
5. **Platform parity is verified, not assumed.** macOS and Linux share the
   same paths and semantics by design; the claim becomes a machine-class
   criterion in the itd-109 calibration set (fresh Linux box on the (b)
   page), not an assumption in prose.
6. **The explicit check lives at `abcd version --check`** (planning ruling,
   2026-08-15). Vintage is `version`'s domain; `ahoy` stays disk-only.
7. **Unknown vintage fails closed at the refusal gate** (planning ruling,
   2026-08-15, from the fit-challenge's stamping-gaps finding). A binary
   whose vintage cannot be determined — unstamped build, `-buildvcs=false`,
   or a dirty (`vcs.modified`) rebuild — is reported as **unknown**, never
   silently treated as fresh; `ahoy install` refuses through it, naming a
   rebuild (or an explicit override flag, specced there) as the out.
8. **Decomposition record (itd-84 hand-run, 2026-08-15).** Verdict SPLIT:
   the network posture → adr-38 + brief invariant 7 (above); the anti-
   wallpaper micro-prompt seed → iss-230. Typed links: `refines` itd-105
   (this intent reports the transition provisioning performs), `refines`
   itd-109 (parity as a machine-class calibration criterion), `refines` the
   iss-206 skew-notice retirement — a scoped replacement, not a reversal:
   steady-state skew machinery stays retired; this intent covers the
   non-steady states (dev checkouts, failed provisioning) pinned
   provisioning cannot reach.

## SOTA

Anchors: the `update-notifier` pattern (npm ecosystem), Homebrew's
auto-update-on-use, and Go's embedded build info (`runtime/debug.BuildInfo`
VCS stamping). **Declared path: 2 — native floor.** The anchors' implicit
network checks fail the fit-challenge outright (a privacy-positioned tool
making background HTTP calls is the posture this record scores against
elsewhere); what survives is their UX grammar — cached comparison, gentle
nudge, one-command fix — implemented over disk-only sources.

**Fit-challenge (independent, 2026-08-15): UPHELD**, with three recorded
caveats. (a) The seam is a **version-source provider interface feeding one
comparator** — `compare(current, expected)` with `expected` drawn from a
provider (disk providers now: embedded revision, checkout tip, manifest pin;
a network/selfupdate provider is a drop-in later, behind the new-dependency
gate). A fixed-arity function over three disk sources would be reuse, not
swappability. (b) The closest disk-only precedent joins the anchors: git's
own "behind upstream" notice — comparison against locally cached refs,
refreshed only by explicit fetch — is exactly the grammar built here.
(c) Go's VCS stamping has known holes (`go run`/`go test` binaries,
`-buildvcs=false`, `.git`-less builds, `go install module@version` history,
`vcs.modified` dirty rebuilds), so the comparator has an explicit
**unknown** outcome — see design decision 7. The challenge also confirmed
no adoptable path-1 candidate exists (every checker in the class is
network-based) and that the pattern's ecosystems offer opt-out, not a
disk-only mode.

## Scope Conditions

None stated.

## Acceptance Criteria

- Given a dogfood checkout whose plugin-root binary predates the source tip,
  when a session starts, then the SessionStart notice names the binary, its
  embedded revision, the tip it trails, and the one-command rebuild fix.
- Given a binary stale against its checkout tip, when `abcd ahoy install`
  runs through it, then the install refuses before any write, naming the
  vintage mismatch; a fresh binary proceeds normally.
- Given any install mode, when `abcd version` or `abcd ahoy` runs, then the
  output includes install mode and vintage (embedded revision or pinned
  version) and staleness relative to the disk references.
- Given any verb without an explicit check flag, when it runs, then no
  network request for version discovery is made — verifiable in the
  zero-network test harness the citation gate already uses.
- Given the explicit check (`abcd version --check`), when the user invokes
  it, then the latest release is fetched once, compared, and reported with
  its source named.
- Given provisioning has fetched a new pinned binary (itd-105/108's job),
  when the next session starts, then the session reports the version
  transition performed.
  _Amended 2026-09-30 by the product thinker's rulings CJ1 and CJ1b of
  2026-09-29 (recorded in `.abcd/work/DECISIONS.md` under 2026-09-30): the
  transition is reported once, by the process that performed the swap, when
  it completes: the bootstrap's success notice and `abcd update`'s receipt
  open with `abcd updated from X to Y`, and the session start shows it only
  for a swap made by a hook that discards its output, once. The per-repo
  setup_version comparison that first met this criterion is removed
  (iss-2609291942520919)._
- Given a binary whose vintage cannot be determined (unstamped build, dirty
  `vcs.modified` rebuild), when staleness is evaluated, then the state is
  reported as unknown — never as fresh — and `abcd ahoy install` run through
  it refuses before any write, naming the rebuild fix.
- Given the same staleness scenarios on macOS and on Linux, when they run on
  each platform, then observed behaviour matches — verified by an agent's
  check on macOS plus the Linux leg of CI, with the record naming both runs.
  _Amended 2026-09-29 by the product thinker's ruling (recorded in
  `.abcd/work/DECISIONS.md` under that date): the agent's check on macOS and
  the Linux CI run stand in for the per-platform check by a person that this
  criterion first asked for ("recorded as a machine-class criterion,
  human-verified per platform", with the itd-109 calibration as the harness).
  itd-109 is still a draft, so no calibration harness exists to run; the
  ruling refines this criterion and no record link type names a ruling, so
  the amendment is stated here. The two runs are recorded in
  [spc-2609230613208843](../../specs/closed/spc-2609230613208843-per-platform-staleness-calibration.md)._

## Open Questions

_All resolved or explicitly deferred at planning (2026-08-15):_

- **Sampled re-surfacing / one-tap micro-prompt** — graduated to its own
  capture, iss-230 (facilitator-experience plan); struck from this intent.
- **Explicit check naming** — resolved: `abcd version --check` (design
  decision 6).
- **Harness portability of the SessionStart channel** — **explicit
  deferral** to the itd-22 lineage: the comparison provider is host-agnostic
  core; each harness adapter wires its own session-start channel when that
  harness lands. Not a criterion of this intent.
- **Refusal breadth** — resolved: deliberately narrow, `ahoy install` only
  (design decision 3); extending to other state-writing verbs is a later
  intent if evidence arrives.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-69424cae8106 -->
Fidelity review — receipt rcp-69424cae8106 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:826d0d983d9ba33c40ce27386862f78a1b69f70dcfa0af491ab32d1abda59ee8
Input attestations: diff:ce8e1c9ae..8ffb6d069 (feat/itd-111-close; spc-22 behaviour judged in the tree at 8ffb6d069)@sha256:95e8eba0442aba7b2e180d12914484e2715c4a0b68d5f2808bea280a0176e2ed;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 5 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the session-start hook runs the plugin-root binary, which renders a notice naming the binary path, its embedded revision, the checkout tip and `make build` when the dogfood comparison is stale; TestFormatStalenessNotice asserts all four tokens
  evidence: internal/surface/cli/cli.go:1735 — "if exe, err := os.Executable(); err == nil { if n := stalenessNotice(cwd, exe)"
  evidence: internal/surface/cli/staleness.go:32 — ""abcd: %s was built from commit %s but this checkout is at %s — the binary is behind its own source. Rebuild it with `make build`.""
  evidence: internal/surface/cli/staleness_test.go:18 — "for _, want := range []string{exe, rev[:12], tip[:12], "make build"}"
  evidence: hooks/hooks.json:19 — ""$CLAUDE_PLUGIN_ROOT/abcd" hook session-start"
- ac-2 — MET_WITH_CONCERNS: the refusal sits before the first apply step and names both revisions; TestInstallRefusesStaleAgainstTip proves no repo write and TestInstallProceedsThroughFreshBinary proves the fresh path; the concern is that under an explicit --bin-dir the writability probe at apply.go:91 (dirWritable, store.go:431) creates and removes a temp file BEFORE the refusal fires, while its comment at apply.go:267 claims it creates nothing
  evidence: internal/core/ahoy/apply.go:157 — "if !opts.AllowStaleBinary { if reason := staleBinaryRefusal(currentVintage(), abs); reason != """
  evidence: internal/core/ahoy/vintage.go:158 — ""the running abcd binary was built from commit %s but this checkout is at %s — it is behind its own source"
  evidence: internal/core/ahoy/refusal_test.go:49 — "func TestInstallRefusesStaleAgainstTip"
  evidence: internal/core/ahoy/refusal_test.go:97 — "func TestInstallProceedsThroughFreshBinary"
  evidence: internal/core/ahoy/store.go:431 — "f, err := os.CreateTemp(dir, ".abcd-write-probe-*")"
  evidence: internal/core/ahoy/apply.go:267 — "Probe writability without creating anything"
- ac-3 — MET_WITH_CONCERNS: `abcd --version` and `abcd ahoy` both render install mode, vintage and staleness from the one ahoy.Vintage comparator, in text and JSON; the concern is wording drift: the literal `abcd version` the criterion names is a moved stub that refuses and points at `abcd --version` (itd-2609212130136102)
  evidence: internal/surface/cli/version.go:105 — "fmt.Fprintf(w, " install: %s\n", out.InstallMode) ... vintage ... staleness"
  evidence: internal/surface/cli/cli.go:252 — "return runVersion(cmd, asJSON, false)"
  evidence: internal/surface/cli/cli.go:3298 — "vin := ahoy.Vintage(cwd)"
  evidence: internal/surface/cli/cli.go:3312 — "fmt.Fprintf(w, " vintage: %s\n", out.Vintage)"
  evidence: internal/surface/cli/version.go:77 — "return movedRefusal("abcd version", "abcd --version")"
  evidence: internal/core/ahoy/vintage_test.go:77 — "func TestVintageDisplayAndStaleness"
- ac-4 — MET: the release fetcher is constructed only behind the explicit check; TestOnlyUpdateCheckTouchesTheNetwork counts zero fetcher constructions across --version, ahoy and session-start, and TestModeAndStatuslineTouchNoNetwork counts http.DefaultTransport hits at zero for mode, statusline and the bare verb
  evidence: internal/surface/cli/version_check_test.go:44 — "if got := atomic.LoadInt32(&calls); got != 0 { t.Fatalf("an implicit path fetched"
  evidence: internal/surface/cli/version.go:102 — "if check { out.Check = runReleaseCheck(v.Version) }"
  evidence: internal/core/vintage/release.go:23 — "ReleaseFetcher ... reached only by an explicit `abcd update --check`. A disk path never constructs one."
  evidence: internal/surface/cli/statusline_test.go:252 — "http.DefaultTransport = countingTransport{&hits}"
- ac-5 — MET_WITH_CONCERNS: the explicit check fetches the latest tag exactly once, compares through vintage.Compare and names its source; the concern is wording drift: the criterion names `abcd version --check`, which is a moved stub refusing towards `abcd update --check` (itd-2609212130136102), where the behaviour lives
  evidence: internal/surface/cli/update.go:49 — "if check {"
  evidence: internal/surface/cli/update.go:53 — "return runVersion(cmd, *asJSON, true)"
  evidence: internal/surface/cli/version.go:129 — "exp := vintage.ReleaseProvider(newReleaseFetcher()).Expected()"
  evidence: internal/surface/cli/version.go:30 — "const checkSource = "github.com/intentdriven/abcd releases""
  evidence: internal/surface/cli/version_check_test.go:50 — "if got := atomic.LoadInt32(&calls); got != 1 { t.Fatalf("update --check fetched %d time(s), want exactly 1""
  evidence: internal/surface/cli/version.go:75 — "return movedRefusal("abcd version --check", "abcd update --check")"
- ac-6 — MET_WITH_CONCERNS: the session-start hook reports a running version that differs from the repo's recorded meta.setup_version, proven by TestSessionStartReportsVersionTransition; the concern is the narrower mechanism: the reference is the per-repo setup_version written by `ahoy install` (not a record beside the plugin-cache metadata as spc-22 stated), so a repo never set up reports no transition, and a dev build on either side is never a transition
  evidence: internal/surface/cli/cli.go:1744 — "if from, to, changed := ahoy.VersionTransition(cwd); changed {"
  evidence: internal/core/ahoy/vintage.go:221 — "return versionTransitionFrom(recordedSetupVersion(cwd), core.Version)"
  evidence: internal/core/ahoy/vintage.go:230 — "if isDevOrUnknown(running) || isDevOrUnknown(recorded) { return recorded, running, false }"
  evidence: internal/surface/cli/transition_test.go:15 — "func TestSessionStartReportsVersionTransition"
- ac-7 — MET: an unstamped or vcs.modified build yields Current{Known:false}, Compare returns Unknown before consulting any provider, vintageFrom reports it terminally, and staleBinaryRefusal refuses naming `make build`; TestInstallRefusesUnknownVintage proves no write
  evidence: internal/core/vintage/vintage.go:80 — "if !cur.Known { return Report{Outcome: Unknown, Current: cur.Revision} }"
  evidence: internal/core/vintage/vintage.go:138 — "if haveModified && modified { return Current{Revision: rev, Known: false} }"
  evidence: internal/core/ahoy/vintage.go:155 — ""the running abcd binary's vintage cannot be determined (an unstamped or modified/dirty build) ... Rebuild it with `make build`"
  evidence: internal/core/ahoy/refusal_test.go:23 — "func TestInstallRefusesUnknownVintage"
  evidence: internal/core/ahoy/vintage_test.go:41 — "func TestVintageFromUnknownCurrentIsTerminal"
- ac-8 — MET_WITH_CONCERNS: the closed spec names both runs; the Linux run (36608836118 at de42f275a, job check (ubuntu-latest) 109544802516) reads success on the forge, and the auditor re-ran the 25 named tests on darwin/arm64 (25 pass, 0 skipped); the concerns are that the amendment cites a 2026-09-29 ruling 'recorded in .abcd/work/DECISIONS.md' which no line of that file carries at 8ffb6d069 or on main, and that the end-to-end scratch-copy check is narrative only, with no script or log in the tree to re-run
  evidence: .abcd/development/specs/closed/spc-2609230613208843-per-platform-staleness-calibration.md:36 — "CI workflow run 36608836118, job `check (ubuntu-latest)` (job 109544802516)"
  evidence: .abcd/development/specs/closed/spc-2609230613208843-per-platform-staleness-calibration.md:42 — "**macOS: the agent's check, 2026-09-29, at the same commit (darwin/arm64).**"
  evidence: .abcd/development/intents/shipped/itd-111-a-stale-abcd-never-answers-silently-every-surface-that-runs.md:160 — "`.abcd/work/DECISIONS.md` under that date"
  evidence: .abcd/work/DECISIONS.md:2587 — "2026-09-29 — An autonomous run keeps at most five sub-agents alive at once (the last 2026-09-29 line; none names itd-111, platform parity or a stand-in check)"

Gap audit:
- honoured:
  - one comparator with a first-class unknown outcome, fed by a provider interface (fit-challenge seam)
    evidence: internal/core/vintage/vintage.go:64 — "type Provider interface { Expected() Expected }"
    evidence: internal/core/vintage/vintage.go:79 — "func Compare(cur Current, p Provider) Report"
  - `ahoy install` refuses through a stale or unknown-vintage binary before the first apply step, with a documented override
    evidence: internal/core/ahoy/apply.go:157 — "if !opts.AllowStaleBinary {"
    evidence: internal/core/ahoy/refusal_test.go:73 — "func TestInstallOverrideProceedsThroughStaleBinary"
  - implicit checks are disk-only; the network answers only the explicit check (adr-38)
    evidence: internal/surface/cli/version_check_test.go:38 — "// Every implicit path: none may fetch."
  - the session-start notice names binary, revision, tip and the one-command rebuild
    evidence: internal/surface/cli/staleness.go:32 — "Rebuild it with `make build`."
  - both platforms exercised the staleness tests: Linux CI success and a macOS re-run by the auditor
    evidence: .abcd/development/specs/closed/spc-2609230613208843-per-platform-staleness-calibration.md:36 — "CI workflow run 36608836118"
- diverged:
  - the explicit check is at `abcd update --check`, not the `abcd version --check` AC5 and design decision 6 name (wording drift from itd-2609212130136102, judged on behaviour)
    evidence: internal/surface/cli/version.go:75 — "return movedRefusal("abcd version --check", "abcd update --check")"
    evidence: internal/surface/cli/update.go:53 — "return runVersion(cmd, *asJSON, true)"
  - the vintage report AC3 places on `abcd version` is at `abcd --version`; `abcd version` refuses with a moved notice (same consolidation)
    evidence: internal/surface/cli/version.go:72 — "cmd.Deprecated = "its report moved to `abcd --version` and its check to `abcd update --check`""
  - the amended AC8 cites a 2026-09-29 product-thinker ruling recorded in .abcd/work/DECISIONS.md; no such line exists at 8ffb6d069 or on main (the PR body says a separate rulings lane appends it)
    evidence: .abcd/development/intents/shipped/itd-111-a-stale-abcd-never-answers-silently-every-surface-that-runs.md:160 — "`.abcd/work/DECISIONS.md` under that date"
    evidence: .abcd/work/DECISIONS.md:2587 — "2026-09-29 — An autonomous run keeps at most five sub-agents alive"
  - under an explicit --bin-dir a writability probe creates and removes a temp file before the stale refusal, and its comment says it creates nothing
    evidence: internal/core/ahoy/apply.go:91 — "binTargetPath, err := resolveInstallTarget(opts, det.pluginRoot)"
    evidence: internal/core/ahoy/store.go:431 — "f, err := os.CreateTemp(dir, ".abcd-write-probe-*")"
    evidence: internal/core/ahoy/apply.go:267 — "Probe writability without creating anything"
  - the transition report keys on the repo's meta.setup_version (per repo, silent for dev builds and for a repo never set up), not on a record beside the plugin-cache metadata as spc-22 stated
    evidence: internal/core/ahoy/vintage.go:242 — "func recordedSetupVersion(cwd string) string"
    evidence: .abcd/development/specs/closed/spc-22-a-stale-abcd-never-answers-silently-every-surface-that-runs.md:84 — "the last-reported one recorded beside the plugin-cache"
- missing: (none)
<!-- abcd-review-end receipt=rcp-69424cae8106 -->

Surfaces named, recorded on 2026-09-29 as iss-2609292057441479 after the fidelity review above: criterion 3's `abcd version` report is met by `abcd --version` (and by `abcd ahoy`), and criterion 5's `abcd version --check` is met by `abcd update --check`, the spellings itd-2609212130136102 consolidated them into; both old spellings refuse with a notice naming the new one. Design decision 6 and the resolved open question on check naming read the same way. The criterion text above stands as shipped, and adr-38's rule is unchanged, so it is not superseded.

Probe ordering paid, recorded on 2026-09-30: the ac-2 concern above, that under an explicit `--bin-dir` the writability probe created and removed a temp file before the stale-binary refusal fired, is resolved by iss-2609291942529461. The refusal runs before the adoption question and before the install target is resolved, so a stale or unknown-vintage binary refuses before it touches the filesystem, and TestStaleRefusalPrecedesTheBinDirProbe holds that order. The ac-2 verdict above stands as the audit recorded it.
