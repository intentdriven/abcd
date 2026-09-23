---
id: itd-101
shipped_in: v0.4.2
slug: every-citation-abcd-publishes-is-provably-alive-and-honestly
spec_id: spc-17
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# **Every citation abcd publishes is provably alive and honestly labelled — and the gate that enforces it never flakes.** A cited reference page is only as trustworthy as its links, and links rot silently: pages retitle, URLs redirect, whole platforms announce their own shutdown. abcd splits citation validation across a deterministic gate and an explicit refresh. An offline lint family checks structure and source policy on every commit — footnote markers and definitions in bijection, URLs and DOIs well-formed, aggregator domains refused — with zero network in the gate. `abcd docs cite refresh` does the live fetching on demand and writes a committed baseline: per URL, the final resolved address, when it was last checked, and whether verification was automatic or manual — never how. Sources that block automated fetchers join a manual queue the maintainer clears link by link: printed as a checklist first, later as a generated, disposable checklist page that hands back a receipt file the verb ingests. The lint then enforces the baseline offline — no broken entries, no stale entries, no cited URL without a receipt, no citation whose recorded final address has drifted. The engine is native and dependency-free first; a specialist link checker can slot in later behind the same seam. "My release gate stays deterministic and my citations stay alive," said Kira, an open-source maintainer. "When a source needs human eyes, abcd hands me the exact list to click through and records that I did — not how I did it."

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

**Every citation abcd publishes is provably alive and honestly labelled — and the gate that enforces it never flakes.** A cited reference page is only as trustworthy as its links, and links rot silently: pages retitle, URLs redirect, whole platforms announce their own shutdown. abcd splits citation validation across a deterministic gate and an explicit refresh. An offline lint family checks structure and source policy on every commit — footnote markers and definitions in bijection, URLs and DOIs well-formed, aggregator domains refused — with zero network in the gate. `abcd docs cite refresh` does the live fetching on demand and writes a committed baseline: per URL, the final resolved address, when it was last checked, and whether verification was automatic or manual — never how. Sources that block automated fetchers join a manual queue the maintainer clears link by link: printed as a checklist first, later as a generated, disposable checklist page that hands back a receipt file the verb ingests. The lint then enforces the baseline offline — no broken entries, no stale entries, no cited URL without a receipt, no citation whose recorded final address has drifted. The engine is native and dependency-free first; a specialist link checker can slot in later behind the same seam. "My release gate stays deterministic and my citations stay alive," said Kira, an open-source maintainer. "When a source needs human eyes, abcd hands me the exact list to click through and records that I did — not how I did it."

## Scope Conditions

None stated.

## Acceptance Criteria

- Given `abcd docs lint` runs in the commit gate, when the citations family evaluates, then it uses zero network: footnote structure (marker-definition bijection, and every crosswalk table row carrying at least one footnote), URL/DOI syntax, and source-domain policy come from committed config and the committed baseline.
- Given `abcd docs cite refresh` runs, when it writes the baseline, then each URL entry records the final resolved address, when it was checked, and whether verification was automatic or manual (with its date) — never how.
- Given a baseline entry older than 180 days, when docs lint runs, then it warns; given one older than 365 days, when the release gate runs, then it blocks; human-verified entries age on the same clock and re-enter the manual queue when stale.
- Given a source that blocks automated fetchers, when the maintainer clears the manual queue, then a printed checklist plus a confirm verb writes the dated receipt; the later generated checklist page hands back a receipt file the same verb ingests — one receipt schema for both rungs.
- Given a specialist link checker is adopted later, then it slots behind the refresh seam as an adapter without changing gate semantics — internal basic first, SOTA dependency later.

## Open Questions

- The generated checklist page's receipt-file format details (settled only as: same schema as the confirm verb writes).
- Where the scheduled-CI refresh wrapper lands when it earns its own sign-off (explicitly out of this intent's scope, per the 2026-07-27 grill).

## Grill Settlements (2026-07-27)

- Refresh is manual now, surfaced by ahoy status and release-preflight nagging; a scheduled-CI wrapper is a later, separately signed-off change.
- Staleness policy: warn at 180 days in docs lint; blocker at 365 days at the release gate only — commits are never calendar-blocked.
- The row-has-footnote structural rule deferred from spc-15 is homed here (DECISIONS 2026-07-27): implemented in this intent or not at all.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-c9215c58f907 -->
Fidelity review — receipt rcp-c9215c58f907 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:52b26b83a679523966b7b9a13546b3b07444305929d5fce0529803864c76f38e
Input attestations: diff:tree at de3ba5fa (spc-17 delivered; main after PR #661)@sha256:4e7430d38ef6b7b6bc533fdf0b8b32a6e08a3e2449d0566027d43316036b8178;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: the citations lint family imports net/url only (no HTTP client): footnote bijection, crosswalk-row footnotes, URL/DOI shape and the configured refused-domain policy all read committed config and the committed baseline; the concern is where 'the commit gate' is — docs lint runs in make preflight and the pre-push hook, not in the pre-commit hook
  evidence: internal/core/lint/citations.go:20 — ""net/url""
  evidence: internal/core/lint/citations.go:347 — "checkCitationFootnotes enforces the marker/definition bijection."
  evidence: internal/core/lint/citations.go:398 — "func checkCitationCrosswalkRows(rel string, lines []string, mask []bool, cfg RuleConfig) ([]Finding, error) {"
  evidence: internal/core/lint/citations.go:217 — "doiShapeRe = regexp.MustCompile(`^10\.\d{4,9}/\S+$`)"
  evidence: internal/core/lint/config.go:210 — "RefusedDomains is the citation_source_policy list of aggregator domains"
  evidence: Makefile:229 — "@go run ./cmd/abcd docs lint"
  evidence: .githooks/pre-push:64 — "make preflight"
- ac-2 — MET: each baseline entry carries final_url, last_checked, verification (automatic|manual) and verified_on; a method/how/transcript key is refused by the loader and the receipt type is proved by reflection to carry no such field
  evidence: internal/core/lint/baseline.go:98 — "FinalURL string `json:"final_url"`"
  evidence: internal/core/lint/baseline.go:102 — "LastChecked string `json:"last_checked"`"
  evidence: internal/core/lint/baseline.go:107 — "Verification string `json:"verification"`"
  evidence: internal/core/lint/baseline.go:109 — "VerifiedOn string `json:"verified_on"`"
  evidence: internal/core/lint/baseline.go:141 — "a smuggled method/how/transcript key is a refusal"
  evidence: internal/core/cite/confirm_test.go:179 — "func TestReceiptHasNoMethodFieldByConstruction"
- ac-3 — MET: the thresholds are 180 (warn) and 365 (block) days; the launch gate measures the baseline through the CLI and refuses on any overdue entry, docs lint --release-gate promotes the finding to a blocker, and a stale manual receipt is re-checked by refresh on the same clock
  evidence: internal/core/lint/citations.go:44 — "defaultCitationWarnDays = 180"
  evidence: internal/core/lint/citations.go:45 — "defaultCitationBlockDays = 365"
  evidence: internal/core/launch/citations.go:107 — "if pre.Overdue > 0 {"
  evidence: internal/surface/cli/cli.go:295 — "Citations: citationPreflight(cwd),"
  evidence: internal/surface/cli/cli.go:492 — "cfg = lint.ArmCitationOverdue(cfg)"
  evidence: internal/core/cite/refresh_test.go:212 — "func TestRefreshRechecksAStaleManualReceipt"
- ac-4 — MET_WITH_CONCERNS: refresh prints the blocked URLs as a manual checklist and `docs cite confirm` writes a dated manual receipt from URLs on the command line or from a --receipt file in the one Receipt schema; the concern is that the generated checklist page itself is not delivered — spc-17 defers it to a later rung and only its receipt schema is fixed
  evidence: internal/core/cite/refresh.go:95 — "Queue is the manual checklist: everything a human must clear."
  evidence: internal/surface/cli/cite.go:114 — "Use: "confirm [url...]","
  evidence: internal/surface/cli/cite.go:164 — "cmd.Flags().StringVar(&receiptPath, "receipt", "","
  evidence: internal/core/cite/confirm.go:55 — "Receipt is what a human — or the generated checklist page — hands back."
  evidence: internal/core/cite/confirm_test.go:15 — "func TestConfirmWritesADatedManualReceipt"
  evidence: .abcd/development/specs/closed/spc-17-every-citation-abcd-publishes-is-provably-alive-and-honestly.md:105 — "The generated checklist page itself may land as the later rung"
- ac-5 — MET: the refresh takes a Checker interface with one method; HTTPChecker is the native implementation and the tests drive the refresh through a stub checker, so a specialist checker is an adapter and the gate reads the baseline only
  evidence: internal/core/cite/fetch.go:105 — "type Checker interface {"
  evidence: internal/core/cite/fetch.go:109 — "HTTPChecker is the native, dependency-free implementation of Checker."
  evidence: internal/core/cite/refresh_test.go:220 — "Refresh(RefreshRequest{RepoRoot: root, Config: testConfig(), Checker: checker, Now: now})"

Gap audit:
- honoured:
  - an offline citations lint family with zero network in the gate
    evidence: internal/core/lint/citations.go:20 — ""net/url""
  - a committed baseline recording final address, when checked, and automatic-vs-manual with its date — never how
    evidence: internal/core/lint/baseline.go:141 — "a smuggled method/how/transcript key is a refusal"
  - 180-day warn in docs lint, 365-day blocker at the release gate, manual receipts ageing on the same clock
    evidence: internal/core/lint/citesummary.go:118 — "case age >= blockDays:"
    evidence: internal/core/launch/citations.go:107 — "if pre.Overdue > 0 {"
  - a manual queue cleared by a confirm verb writing dated receipts, one schema
    evidence: internal/core/cite/confirm.go:55 — "Receipt is what a human — or the generated checklist page — hands back."
  - a native engine behind a seam a specialist checker can slot into
    evidence: internal/core/cite/fetch.go:105 — "type Checker interface {"
- diverged:
  - docs lint runs in the commit gate — delivered in the preflight/pre-push gate, not the pre-commit hook
    evidence: .githooks/pre-push:64 — "make preflight"
    evidence: Makefile:229 — "@go run ./cmd/abcd docs lint"
- missing:
  - the generated, disposable checklist page that hands back a receipt file
    evidence: .abcd/development/specs/closed/spc-17-every-citation-abcd-publishes-is-provably-alive-and-honestly.md:105 — "The generated checklist page itself may land as the later rung"
    evidence: internal/surface/cli/cite.go:165 — "the format the generated checklist page emits"