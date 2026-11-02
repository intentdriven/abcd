---
id: itd-176
slug: whatever-ships-says-how-hard-anyone
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# Whatever ships says how hard anyone looked at it

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

Whatever ships says how hard anyone looked at it

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [verification-assurance SOTA note](../../research/notes/2026-10-23-verification-assurance-sota.md).

- **A per-section label with graded sub-levels is the established documentation shape.** Node marks every API section 0 Deprecated, 1 Experimental (1.0 early development, 1.1 active development, 1.2 release candidate), 2 Stable or 3 Legacy, each with what it promises and what it does not (Node.js, About this documentation).
- **A review claim names its scope and disclaims the rest.** The kernel's Reviewed-by carries a statement of oversight: a technical review for "appropriateness and readiness", concerns communicated back, "free of known issues which would argue against its inclusion", and no warranty "that it will achieve its stated purpose"; Tested-by says only "(in some environment)"; Acked-by is "not as formal" (kernel, Submitting patches).
- **A level is stated with what it defends against and what it does not.** SLSA Build L1 "Can be used to prevent mistakes but is trivial to bypass or forge"; L2's forgery needs "an explicit 'attack', though this may be easy"; L3 defends against tampering during the build (SLSA v1.0 levels). The honest negative is part of the definition.
- **Self-reported assurance is a claim with evidence and a date.** Security Insights v2.2.0, the current specification, records an assessment as `name`, `date`, `evidence` URL and a required `comment`, and each security tool with an `integration` of `ci`, `release` and `adhoc` booleans and where each set of results lives (Security Insights specification v2.2.0, `docs/schema.md`); the 1.0.0 shape of auditor name, URL and report year is superseded.
- **Self-certified and machine-verified are reported apart.** Scorecard says the Best Practices badge "can report on claims and claim justifications from people", which it cannot verify; the badge annotates each criterion with a justification and a URL (Scorecard checks; Best Practices criteria).
- **Locally**, the material exists as receipts: the preflight receipt per commit, the docs-fidelity review labelled with the commit it read, the intent-fidelity verdict carrying verifier id and version with rubric and prompt hashes, and a changelog composed only from records. Missing is one derived line per shipped thing that reads them.

**Path 2: a derived disclosure per shipped intent and per release, shaped to the kernel's statement of oversight and Security Insights' claim-plus-evidence record.** The binary composes a "looked at" block from the receipts it holds: per track, the rung reached (itd-173), the verifier (id and version, or a role), the commit the check read, and the standing disclaimer; a rung with no receipt renders as "not looked at" rather than being omitted. The seam is one record shape per track that an outside audit writes into as one more row, shaped to the v2.2.0 assessment fields (name, date, evidence, comment) and tool integration flags (ci, release, adhoc). Never a hand-written line, since the changelog is derived and a self-asserted level is the claim Scorecard warns it cannot check. No dependency. The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) Where does the block render: the shipped intent's audit notes, the release page, the changelog section, or all three from one source?
- (SOTA 2026-10-23) Does "not looked at" on a track block anything, or is it disclosure only? (The itd-175 ruling names an absence without blocking; the same stance is the obvious default.)
- (SOTA 2026-10-23) How is a person's review recorded so it can be disclosed by role without naming them, given the record rules on names?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
