---
id: itd-65
slug: launch-preflight-gate-suite
spec_id: spc-2609201955279019
kind: standalone
suggested_kind: standalone
reclassification_history: []
related_adrs: [adr-28]
prd_path: null
grill_session_id: 65d0f1de-0065-4a65-9c0d-000000000065
grilled_at: 2026-07-01
grilled_intent_hash: e11007a63a6727350b011965544b548683256e4cb249d67e5b1ab7894d13809f
glossary_terms_used:
- distribution/release
- distribution/end-user
- core/brief
- core/intent
- core/oracle
- core/phase
- interview/session
warrants_assumed:
- "An oracle/LLM backend may or may not be available on the ship host; the doc-history gate degrades to deterministic patterns without failing open."
- "The public org handle is a known allowlistable constant distinct from the maintainer's personal git identity."
blocked_by: [itd-66]
builds_on: [itd-67]
severity: critical
impact: additive
---

# abcd Refuses To Publish A Payload Until The Full Ship-Time Pre-Flight Gate Suite Passes, Not Just The Secret Scan

## Press Release

> **`/abcd:launch ship` gains the complete Phase-5 pre-flight gate suite the brief specifies: on top of the spc-64 (predecessor store) secret + PII scan, it adds the custom-regex identity layer (home-dir paths, real emails, GitHub usernames), marker-block sanity, `plugin.json` + `marketplace.json` validation, dirty-tree refusal, and the warn-fail documentation and hook-compliance checks — each hard-failing (or warn-failing) exactly as the brief's § 1 pins.** Today `launch` is a dry-run/render-only stub: it runs the spc-64 (predecessor store) gate for real but renders every other gate as "(not yet implemented)". That means a real promotion would ship with those gates inert — precisely the invisible risks abcd exists to catch. This intent graduates the dry-run's "not yet implemented" lines into a real, runnable, fail-closed gate suite so a publish is blocked on a finding, not merely previewed.

> "The dry-run already tells me a home-directory path or a broken plugin.json *would* be a problem," said a maintainer. "But 'would' isn't 'does' — ship has to actually hard-fail on it. I don't want to hand-audit the payload before every snapshot; the gate suite should."

## Why This Matters

abcd's whole thesis is routing the risks a non-expert cannot see to a fail-closed gate ([[itd-62-pluggable-safety-gate]]). Its OWN publish path is the highest-stakes instance of that: a launch cuts a curated release from the single repo — packaging that excludes `.abcd/**` ([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)) — and publishes it, where a leaked home-dir path, real email, or committed secret is irreversible. The canonical launch brief (`04-surfaces/04-launch.md` § 1) already specifies the full gate suite; spc-64 (predecessor store) built the secret/PII floor; but the custom-regex identity layer, marker-block sanity, plugin/marketplace validation, and dirty-tree refusal are still stubs. Until they are real, `launch ship` cannot honestly claim to gate a promotion — and the project standards (no home-dir paths, no real emails, no usernames in file content) have no enforcement at the one moment they matter most. This closes that honesty gap the same way spc-74 closed the doc-fidelity one: make the built reality match what the surface implies.

## What's In Scope

- The custom-regex identity layer over the resolved payload: home-dir paths (`/Users/...`, `/home/...`), real emails, GitHub usernames from git config — hard-fail (per brief § 1). <!-- abcd-audit:allow -->
- Marker-block sanity over shipped Markdown — hard-fail on malformed.
- `plugin.json` parse + `marketplace.json` reference cross-check — hard-fail.
- **Doc-history gate** — hard-fail on change-history / rationale-for-change
  narration in shipped doc bodies (the `abcd-cli/CLAUDE.md` "docs describe
  present state, never change history" rule, enforced at ship). On a finding
  the gate extracts the flagged passage and OFFERS to auto-append it to the
  changelog owned by [[itd-67-installable-versioned-plugin]] — so the fix is
  one confirmation, not a manual rewrite. Reuses the spc-74 `doc_fidelity`
  surface as the detector.
- Dirty-tree refusal unless `--allow-dirty`; git-inferable-metadata scan (dates/authors/versions in file content, per project standards).
- Warn-fail gates: hook-compliance, documentation auditor over `docs/`.
- A single fail-closed orchestrator that runs the suite against the § 2 payload include-manifest and writes the pre-flight report (`.abcd/logbook/launch/<ts>/preflight.{json,md}`), returning non-zero on any hard-fail — the Phase-5 `ship` behaviour, distinct from `dry-run`'s always-exit-0 preview. The orchestrator RUNS ALL gates and collects ALL findings before returning a verdict (report-everything, one fix pass), with a single ordering constraint: doc-history reroute runs before the dirty-tree gate (see below).
- Gate composition + tiering (grill Q2/Q4/Q6): the custom-regex identity layer is a SIBLING gate the orchestrator composes (spc-64, predecessor store, keeps its pinned gitleaks+pii.py engines); the identity layer flags only leaks of the LOCAL git identity (user.name/user.email, dev-repo remote URLs) with the public org handle allowlisted, never arbitrary handles. The suite is built as a callable unit so CI and pre-commit can invoke the SAME checks earlier (advisory/blocking per tier), while `launch ship` holds the authoritative hard-fail.
- Doc-history detector (grill Q3/Q5): a LAYERED detector — deterministic narrow patterns (past→present transitions: "used to X", "changed from X to Y", "no longer X", "migrated from", "renamed X to Y") run always, local-first, never hard-failing on bare present-tense "now"/"previously"; an OPTIONAL oracle/LLM pass (reusing the spc-27 oracle + spc-64 fail-closed-on-unavailable precedent, both predecessor store) adjudicates ambiguous hits when a backend is available, falling back to surface-for-confirmation when not. Auto-reroute stages its own changelog + doc edits as one fix transaction; the dirty-tree gate runs AFTER reroute resolution so a clean staged fix is not mistaken for unexpected dirt.
- Reuse of the unmodified upstream scanners (gitleaks ≥ 8.18.0 pinned per spc-64 (predecessor store); wrap, never fork) per the wrap-only rule.

## What's Out of Scope

- The payload render / mirror-mode / versioning machinery (that is the sibling launch work [[itd-66-launch-payload-render-parity]] — this intent is the GATES only).
- Replacing the spc-64 (predecessor store) secret/PII engine or adopting Presidio as the wired engine (a separate recorded decision per brief § 1 / spc-64 (predecessor store) C1a).
- Forking or reimplementing any scanner — configure and wrap the trusted ones.
- Publishing anything: this intent decides go/no-go; it never pushes.

## Scope Conditions

None stated.

## Delivery Status

Reviewed against the v0.9.0 tree on 2026-09-20. Delivered and cited: the
identity hard-fails on a home path, a real email and a username, naming the
file (`internal/adapter/scanner/identity.go`, `TestRenderPayloadSecretRefuses`,
`TestShipBlocksOnSecret`); the scanner failing closed when unavailable
(`TestZeroCoverageRefuses`); a local identity flagged while an org handle
passes (`TestOtherIdentitiesArmMatchersAndHandleStaysPublic`); a clean payload
exiting 0 (`TestShipCleanWouldPublish`); the manifest half of the marker
criterion (`smoke.go`). Moot: the auto-append of narration into the changelog,
because the changelog is derived (adr-37); the reroute-not-dirt criterion,
because no reroute exists. Scoped by spc-2609201955279019 and delivered with
it on 2026-09-25: the marker-block check, the change-narration detector, the
dirty-tree refusal, the two warn-fail rows, the report file, the multi-gate
test (the evidence for each is under `## Decisions`).

## Decisions

- **2026-09-25 — two criteria are superseded in part by the derived
  changelog, and the rest of each stands.** The changelog is derived from the
  records: [adr-37](../../decisions/adrs/0037-changelog-driven-releases.md)
  makes the dated CHANGELOG heading the release instrument, and its
  2026-08-26 amendment records that the ingest composes that section from the
  records that shipped, behind the completeness bijection (first cut this way:
  v0.5.1, `.abcd/work/DECISIONS.md`, 2026-08-16). A gate cannot append prose to
  a changelog nothing hand-writes, so:
  - *The doc-history criterion (third).* Superseded: the offer to auto-append
    the flagged passage, and "the change is recorded in the changelog" as a
    condition the gate checks — the change reaches the changelog through its
    record, not through the gate. Stands and is met: the gate HARD-FAILS on a
    change-narration sentence in a shipped doc body and names it
    (`change-narration` row; `narrationFindings` in
    `internal/core/launch/gates.go`; `TestNarrationGateHardFailsOnAChangeConstruct`),
    and the ship proceeds only once the doc describes present state.
  - *The reroute criterion (last).* Superseded whole: with no auto-append there
    is no reroute and nothing it stages. The ordering it protected stands in
    another form and is met: the cut runs the dirty-tree gate before it writes
    anything, and the render after its own writes skips it, so the cut's own
    output is never read as dirt (`DirtySkip` in `gates.go`;
    `TestLaunchShipRefusesADirtyTreeUnlessAllowed`).
- **2026-09-25 — how the remaining criteria are met** (spc-2609201955279019).
  Marker blocks: `marker-block` row, `TestMarkerBlockGateRefusesAMalformedBlock`.
  Dirty tree: `dirty-tree` row and `abcd launch ship --allow-dirty`, the
  override recorded in the pre-flight report with every path it carried
  (`TestDirtyTreeGateRefusesUnlessAllowed`,
  `TestLaunchShipRefusesADirtyTreeUnlessAllowed`). Warn tier: the
  `documentation-auditor` row runs the docs-lint engine the front door
  measures, the `hook-compliance` row checks hook executability, handler
  commands and timeouts, and `"strict_warnings": true` in
  `.abcd/config/launch-payload.json` is the configured strict tier
  (`TestWarnRowsSurfaceWithoutBlocking`). Report: every preview and every cut
  that renders a payload writes `preflight.{json,md}` under
  `.abcd/.work.local/logs/launch/<ts>/`, the tier iss-73 settled
  (`TestWritePreflightReportLandsInTheLocalTier`,
  `TestLaunchDryRunWritesThePreflightReport`). Run-all-collect-all:
  `TestSuiteReportsEveryGateInOnePass` plants findings in four gates and finds
  all four in the preview, the render refusal and the written report. Bare
  present-tense "now"/"previously": `TestNarrationGatePassesPresentTense`.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a resolved payload containing a home-dir path, real email, or GitHub username, **when** the ship gate suite runs, **then** it HARD-FAILS (non-zero) and names the finding with its file and match, never a silent pass.
- **Given** a malformed marker block or an unparseable `plugin.json` / dangling `marketplace.json` reference in the payload, **when** the suite runs, **then** it hard-fails with the specific defect.
- **Given** a shipped doc body containing change-history or rationale-for-change narration ("previously X, now Y", migration notes), **when** the doc-history gate runs during ship, **then** it HARD-FAILS and offers to auto-append the flagged passage to the [[itd-67-installable-versioned-plugin]] changelog; the ship proceeds only once the doc describes present state and the change is recorded in the changelog.
- **Given** a dirty working tree, **when** ship runs without `--allow-dirty`, **then** it refuses; **with** `--allow-dirty` it proceeds and records the override.
- **Given** a doc-auditor or hook-compliance concern, **when** the suite runs, **then** it WARN-fails (surfaced, non-blocking unless configured strict).
- **Given** gitleaks is absent or older than the pinned floor, **when** the suite runs, **then** it fails closed (never a regex fallback), consistent with spc-64 (predecessor store).
- **Given** a fully clean payload, **when** the suite runs, **then** it exits 0 and writes the pre-flight report.
- **Given** a payload with multiple independent findings across different gates, **when** the suite runs, **then** it reports ALL of them in one pass (run-all-collect-all), not just the first hard-fail.
- **Given** the identity gate, **when** it scans, **then** it flags a leaked LOCAL git identity (maintainer's personal name/email/handle) but NOT the allowlisted public org handle appearing in install docs.
- **Given** a doc using bare present-tense "now"/"previously" without a change construct, **when** the doc-history gate runs, **then** it does NOT hard-fail; a genuine "changed from X to Y" / "no longer" construct DOES.
- **Given** the doc-history auto-reroute writes the changelog and stages it, **when** the dirty-tree gate then runs, **then** it treats the reroute's staged fix as expected (not dirt) because reroute is ordered before it.

## Open Questions

- ~~Should the custom-regex identity layer live inside the spc-64 (predecessor store) gate module (extending its config) or as a sibling gate the orchestrator composes? (Reuse vs separation.)~~ Answered by this intent's own scope (grill Q2/Q4/Q6, "Gate composition + tiering"): a sibling gate the orchestrator composes, with spc-64 (predecessor store) keeping its pinned engines; delivered as `internal/adapter/scanner/identity.go` (Delivery Status).
- ~~What is the exact GitHub-username source — git config `user.name`/`user.email`, remote URLs, or a maintained denylist — and how are legitimate org handles in docs distinguished from leaked personal ones?~~ Answered by the same scope bullet: the LOCAL git identity (`user.name`/`user.email`, the dev-repo remote URL), never a denylist of arbitrary handles, with the public org handle allowlisted; delivered and pinned by `TestOtherIdentitiesArmMatchersAndHandleStaysPublic`.
- Does the documentation auditor reuse the existing doc-scout machinery, or is it a launch-specific pass? Open. The `documentation-auditor` row runs the deterministic docs-lint engine today; whether the host-delegated `documentation-auditor` agent (`brief/05-internals/01-agents.md`) joins it is not ruled.
- Where does `--allow-doc-warnings` sit relative to a strict CI invocation of the same suite? Open. No such flag ships: a warning refuses nothing unless the repository sets `"strict_warnings": true`, and whether a per-run override of that setting is wanted is not ruled.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-28b4f73968a9 -->
Fidelity review — receipt rcp-28b4f73968a9 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:d1dc999aa4e96dcd17f1bf6b2596d9c3612dbd635f617f29130bd14cbb0b4c1c
Input attestations: diff:44eaaf64^..811fba17 -- internal/core/launch internal/surface/cli/launch.go internal/surface/cli/launch_gates_test.go (the gate suite from its feat commit to main)@sha256:fc86184375dafd5ef674ae77e5910e15c2193c9d82a928f9600c68d0bef89811;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 4 · NOT_MET 1 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the identity scanner classifies the caller's home path, the other-home shape, the git identity and the local username as findings, the render precheck refuses on a finding in an included file before any write, and the ship path blocks on it naming the file
  evidence: internal/adapter/scanner/identity.go:44 — "kindHomeSelf = "home_path_self""
  evidence: internal/adapter/scanner/identity.go:72 — "func ProbeIdentity(repoRoot string) Identity {"
  evidence: internal/core/launch/render_scan_test.go:18 — "func TestRenderPayloadSecretRefuses"
  evidence: internal/core/launch/dryrun_test.go:52 — "func TestShipBlocksOnSecret"
- ac-2 — MET: the marker-block row hard-fails a malformed block naming it, and the lockstep check refuses a marketplace entry with no source, an unparseable sourced manifest, or a name the sourced manifest does not carry
  evidence: internal/core/launch/gates.go:306 — "func markerBlockFindings(bundle Bundle) []GateFinding {"
  evidence: internal/core/launch/gates_test.go:61 — "func TestMarkerBlockGateRefusesAMalformedBlock"
  evidence: internal/core/launch/lockstep.go:118 — "marketplace lists %q but the sourced manifest is named %q — the install id would not resolve"
  evidence: internal/core/launch/render_test.go:222 — "func TestRenderPayloadRefusesAnUnstampableMarketplace"
- ac-3 — MET_WITH_CONCERNS: the change-narration row hard-fails a change construct in a shipped doc body and names it; the offer to auto-append the passage to the changelog and the changelog-recorded condition are not delivered, superseded by the derived changelog (adr-37) as the intent's 2026-09-25 decision records, and the hard tier's reach is the subject of the open finding iss-2609251827286563
  evidence: internal/core/launch/gates.go:749 — "func narrationFindings(bundle Bundle) []GateFinding {"
  evidence: internal/core/launch/gates_test.go:184 — "func TestNarrationGateHardFailsOnAChangeConstruct"
  evidence: .abcd/development/intents/shipped/itd-65-launch-preflight-gate-suite.md:98 — "Superseded: the offer to auto-append"
  evidence: .abcd/work/issues/open/iss-2609251827286563-the-launch-gate-suite-s-change-narration-gate-hard-fail-tier.md:1 — "the-launch-gate-suite-s-change-narration-gate-hard-fail-tier"
- ac-4 — MET_WITH_CONCERNS: the dirty-tree row refuses a dirty tree and --allow-dirty proceeds with the override and its paths in the report, on the core and through the CLI; the concern is the open finding that the render path hardcodes the gate to DirtySkip (iss-2609251827294854), so one entry point states a policy it does not apply
  evidence: internal/core/launch/gates.go:941 — "func dirtyTreeGate(repoRoot string, policy DirtyPolicy) (GateSummary, []string, string) {"
  evidence: internal/core/launch/gates_test.go:265 — "func TestDirtyTreeGateRefusesUnlessAllowed"
  evidence: internal/surface/cli/launch_gates_test.go:73 — "func TestLaunchShipRefusesADirtyTreeUnlessAllowed"
  evidence: .abcd/work/issues/open/iss-2609251827294854-renderpayload-hardcodes-the-dirty-tree-gate-to-dirtyskip-for.md:1 — "renderpayload-hardcodes-the-dirty-tree-gate-to-dirtyskip-for"
- ac-5 — MET_WITH_CONCERNS: the documentation-auditor and hook-compliance rows are warn tier, surfaced as warnings and refusing only under strict_warnings in the include config; the concern is the open finding that the hook-compliance row fails open on one shape (iss-2609251827104081)
  evidence: internal/core/launch/gates.go:181 — "if req.Policy.StrictWarnings {"
  evidence: internal/core/launch/gates.go:1003 — "func hookComplianceGate(bundle Bundle) GateSummary {"
  evidence: internal/core/launch/gates_test.go:330 — "func TestWarnRowsSurfaceWithoutBlocking"
  evidence: .abcd/work/issues/open/iss-2609251827104081-the-launch-gate-suite-s-hook-compliance-row-fails-open-on-a.md:1 — "the-launch-gate-suite-s-hook-compliance-row-fails-open-on-a"
- ac-6 — MET_WITH_CONCERNS: the suite fails closed when the scanner is unavailable or leaves an included file uncovered, never publishing; the premise of the criterion does not hold as written, because gitleaks is not the engine: the 2026-07-24 ruling made the native pattern scanner the default engine with no shell-out and gitleaks an optional stronger one, so there is no pinned-floor check and the fail-closed property attaches to scanner availability and coverage instead
  evidence: internal/core/launch/dryrun.go:241 — "if scan.Unavailable {"
  evidence: internal/core/launch/dryrun.go:247 — "// Fail closed on the coverage gap: any include-selected file the scanner"
  evidence: internal/core/launch/dryrun_test.go:109 — "func TestZeroCoverageRefuses"
  evidence: internal/adapter/scanner/finding.go:6 — "gitleaks/trufflehog shell-out) — so it is fully testable and reusable across"
  evidence: .abcd/work/DECISIONS.md:744 — "native patterns are the default engine, gitleaks is the stronger"
- ac-7 — MET: a clean payload would publish with exit 0 and every preview or cut that renders a payload writes preflight.json and preflight.md under the local tier
  evidence: internal/core/launch/dryrun_test.go:78 — "func TestShipCleanWouldPublish"
  evidence: internal/core/launch/gates_test.go:464 — "func TestWritePreflightReportLandsInTheLocalTier"
  evidence: internal/surface/cli/launch_gates_test.go:34 — "func TestLaunchDryRunWritesThePreflightReport"
- ac-8 — MET: the suite runs every gate and collects every finding before returning, and the test plants findings in four gates and finds all four in the preview, the render refusal and the written report
  evidence: internal/core/launch/gates.go:163 — "func runGateSuite(req suiteRequest) suiteResult {"
  evidence: internal/core/launch/gates_test.go:402 — "func TestSuiteReportsEveryGateInOnePass"
- ac-9 — MET: the identity probe arms matchers from the local git identity and its other scopes while the public org handle passes as a public handle
  evidence: internal/adapter/scanner/identity.go:230 — "func isPublicHandle(name string, id Identity) bool {"
  evidence: internal/adapter/scanner/identity_scopes_test.go:111 — "func TestOtherIdentitiesArmMatchersAndHandleStaysPublic"
- ac-10 — MET: bare present-tense now/previously passes and a changed-from / no-longer construct hard-fails, each pinned by its own test
  evidence: internal/core/launch/gates_test.go:221 — "func TestNarrationGatePassesPresentTense"
  evidence: internal/core/launch/gates_test.go:184 — "func TestNarrationGateHardFailsOnAChangeConstruct"
  evidence: internal/core/launch/gates.go:552 — "func noLongerNarrates(sentence string, at []int) bool {"
- ac-11 — NOT_MET: promised: a doc-history auto-reroute that stages a changelog fix, ordered before the dirty-tree gate; delivered: no reroute exists, so nothing is staged and nothing is ordered before the gate; the intent's 2026-09-25 decision records the criterion as superseded whole by the derived changelog, and the cut's own writes are kept out of the dirty read by DirtySkip instead
  evidence: .abcd/development/intents/shipped/itd-65-launch-preflight-gate-suite.md:106 — "Superseded whole: with no auto-append there"
  evidence: internal/core/launch/gates.go:89 — "// DirtySkip leaves the gate out. It exists for the render a cut runs AFTER"

Gap audit:
- honoured:
  - a publish is blocked on a finding, not merely previewed: the render refuses before any write on the same verdict the preview reports
    evidence: internal/core/launch/dryrun.go:236 — "(PrecheckPayload) fails closed on exactly the same scan verdict the dry-run and"
  - run-all-collect-all with the pre-flight report written to the local tier
    evidence: internal/core/launch/gates_test.go:402 — "TestSuiteReportsEveryGateInOnePass"
    evidence: internal/core/launch/gates_test.go:464 — "TestWritePreflightReportLandsInTheLocalTier"
  - the identity layer is a sibling gate flagging the local identity only, with the org handle allowlisted
    evidence: internal/adapter/scanner/identity_scopes_test.go:111 — "TestOtherIdentitiesArmMatchersAndHandleStaysPublic"
- diverged:
  - the doc-history gate offers to auto-append the flagged passage to the changelog; delivered as a hard-fail only, the append superseded by the derived changelog (adr-37) and recorded on the intent
    evidence: .abcd/development/intents/shipped/itd-65-launch-preflight-gate-suite.md:98 — "Superseded: the offer to auto-append"
  - gitleaks at a pinned floor as the secret engine, failing closed when absent; delivered as a native pattern scanner with no shell-out, failing closed on unavailability and coverage, per the 2026-07-24 ruling that the intent record does not itself name
    evidence: .abcd/work/DECISIONS.md:743 — "2026-07-24 — itd-28 gitleaks sign-off: Stage 2 runs through the scanner"
    evidence: internal/adapter/scanner/finding.go:6 — "gitleaks/trufflehog shell-out)"
  - the optional oracle/LLM adjudication of ambiguous narration hits; delivered as a deterministic layered detector only, the oracle pass absent and the auditor-agent question left open on the intent
    evidence: internal/core/launch/gates.go:807 — "func narrationConstruct(sentence string) string {"
    evidence: .abcd/development/intents/shipped/itd-65-launch-preflight-gate-suite.md:151 — "Does the documentation auditor reuse the existing doc-scout machinery"
- missing:
  - the doc-history auto-reroute ordered before the dirty-tree gate; nothing reroutes, recorded as superseded whole
    evidence: .abcd/development/intents/shipped/itd-65-launch-preflight-gate-suite.md:106 — "Superseded whole: with no auto-append there"
  - the review's five gate-suite findings captured on 2026-09-25 remain open, two of them on gates this audit marks with concerns (hook-compliance fails open; the render path's DirtySkip)
    evidence: .abcd/work/issues/open/iss-2609251827104081-the-launch-gate-suite-s-hook-compliance-row-fails-open-on-a.md:1 — "the-launch-gate-suite-s-hook-compliance-row-fails-open-on-a"
    evidence: .abcd/work/issues/open/iss-2609251827294854-renderpayload-hardcodes-the-dirty-tree-gate-to-dirtyskip-for.md:1 — "renderpayload-hardcodes-the-dirty-tree-gate-to-dirtyskip-for"
