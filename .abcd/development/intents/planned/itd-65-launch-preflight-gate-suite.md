---
id: itd-65
slug: launch-preflight-gate-suite
spec_id: null
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
---

# abcd Refuses To Publish A Payload Until The Full Ship-Time Pre-Flight Gate Suite Passes, Not Just The Secret Scan

## Press Release

> **`/abcd:launch ship` gains the complete Phase-5 pre-flight gate suite the brief specifies: on top of the spc-64 secret + PII scan, it adds the custom-regex identity layer (home-dir paths, real emails, GitHub usernames), marker-block sanity, `plugin.json` + `marketplace.json` validation, dirty-tree refusal, and the warn-fail documentation and hook-compliance checks — each hard-failing (or warn-failing) exactly as the brief's § 1 pins.** Today `launch` is a dry-run/render-only stub: it runs the spc-64 gate for real but renders every other gate as "(not yet implemented)". That means a real promotion would ship with those gates inert — precisely the invisible risks abcd exists to catch. This intent graduates the dry-run's "not yet implemented" lines into a real, runnable, fail-closed gate suite so a publish is blocked on a finding, not merely previewed.

> "The dry-run already tells me a home-directory path or a broken plugin.json *would* be a problem," said a maintainer. "But 'would' isn't 'does' — ship has to actually hard-fail on it. I don't want to hand-audit the payload before every snapshot; the gate suite should."

## Why This Matters

abcd's whole thesis is routing the risks a non-expert cannot see to a fail-closed gate ([[itd-62-pluggable-safety-gate]]). Its OWN publish path is the highest-stakes instance of that: a launch cuts a curated release from the single repo — packaging that excludes `.abcd/**` ([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)) — and publishes it, where a leaked home-dir path, real email, or committed secret is irreversible. The canonical launch brief (`04-surfaces/04-launch.md` § 1) already specifies the full gate suite; spc-64 built the secret/PII floor; but the custom-regex identity layer, marker-block sanity, plugin/marketplace validation, and dirty-tree refusal are still stubs. Until they are real, `launch ship` cannot honestly claim to gate a promotion — and the project standards (no home-dir paths, no real emails, no usernames in file content) have no enforcement at the one moment they matter most. This closes that honesty gap the same way spc-74 closed the doc-fidelity one: make the built reality match what the surface implies.

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
- Gate composition + tiering (grill Q2/Q4/Q6): the custom-regex identity layer is a SIBLING gate the orchestrator composes (spc-64 keeps its pinned gitleaks+pii.py engines); the identity layer flags only leaks of the LOCAL git identity (user.name/user.email, dev-repo remote URLs) with the public org handle allowlisted, never arbitrary handles. The suite is built as a callable unit so CI and pre-commit can invoke the SAME checks earlier (advisory/blocking per tier), while `launch ship` holds the authoritative hard-fail.
- Doc-history detector (grill Q3/Q5): a LAYERED detector — deterministic narrow patterns (past→present transitions: "used to X", "changed from X to Y", "no longer X", "migrated from", "renamed X to Y") run always, local-first, never hard-failing on bare present-tense "now"/"previously"; an OPTIONAL oracle/LLM pass (reusing the spc-27 oracle + spc-64 fail-closed-on-unavailable precedent) adjudicates ambiguous hits when a backend is available, falling back to surface-for-confirmation when not. Auto-reroute stages its own changelog + doc edits as one fix transaction; the dirty-tree gate runs AFTER reroute resolution so a clean staged fix is not mistaken for unexpected dirt.
- Reuse of the unmodified upstream scanners (gitleaks ≥ 8.18.0 pinned per spc-64; wrap, never fork) per the wrap-only rule.

## What's Out of Scope

- The payload render / mirror-mode / versioning machinery (that is the sibling launch work [[itd-66-launch-payload-render-parity]] — this intent is the GATES only).
- Replacing the spc-64 secret/PII engine or adopting Presidio as the wired engine (a separate recorded decision per brief § 1 / spc-64 C1a).
- Forking or reimplementing any scanner — configure and wrap the trusted ones.
- Publishing anything: this intent decides go/no-go; it never pushes.

## Scope Conditions

None stated.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a resolved payload containing a home-dir path, real email, or GitHub username, **when** the ship gate suite runs, **then** it HARD-FAILS (non-zero) and names the finding with its file and match, never a silent pass.
- **Given** a malformed marker block or an unparseable `plugin.json` / dangling `marketplace.json` reference in the payload, **when** the suite runs, **then** it hard-fails with the specific defect.
- **Given** a shipped doc body containing change-history or rationale-for-change narration ("previously X, now Y", migration notes), **when** the doc-history gate runs during ship, **then** it HARD-FAILS and offers to auto-append the flagged passage to the [[itd-67-installable-versioned-plugin]] changelog; the ship proceeds only once the doc describes present state and the change is recorded in the changelog.
- **Given** a dirty working tree, **when** ship runs without `--allow-dirty`, **then** it refuses; **with** `--allow-dirty` it proceeds and records the override.
- **Given** a doc-auditor or hook-compliance concern, **when** the suite runs, **then** it WARN-fails (surfaced, non-blocking unless configured strict).
- **Given** gitleaks is absent or older than the pinned floor, **when** the suite runs, **then** it fails closed (never a regex fallback), consistent with spc-64.
- **Given** a fully clean payload, **when** the suite runs, **then** it exits 0 and writes the pre-flight report.
- **Given** a payload with multiple independent findings across different gates, **when** the suite runs, **then** it reports ALL of them in one pass (run-all-collect-all), not just the first hard-fail.
- **Given** the identity gate, **when** it scans, **then** it flags a leaked LOCAL git identity (maintainer's personal name/email/handle) but NOT the allowlisted public org handle appearing in install docs.
- **Given** a doc using bare present-tense "now"/"previously" without a change construct, **when** the doc-history gate runs, **then** it does NOT hard-fail; a genuine "changed from X to Y" / "no longer" construct DOES.
- **Given** the doc-history auto-reroute writes the changelog and stages it, **when** the dirty-tree gate then runs, **then** it treats the reroute's staged fix as expected (not dirt) because reroute is ordered before it.

## Open Questions

- Should the custom-regex identity layer live inside the spc-64 gate module (extending its config) or as a sibling gate the orchestrator composes? (Reuse vs separation.)
- What is the exact GitHub-username source — git config `user.name`/`user.email`, remote URLs, or a maintained denylist — and how are legitimate org handles in docs distinguished from leaked personal ones?
- Does the documentation auditor reuse the existing doc-scout machinery, or is it a launch-specific pass?
- Where does `--allow-doc-warnings` sit relative to a strict CI invocation of the same suite?
