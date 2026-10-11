---
id: itd-149
slug: abcd-handles-inbound-security-advisories
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# abcd handles inbound security advisories and issues, and cuts the release, for every managed repo. The 2026-08-27 security-advisory pilot proved the loop by hand across two independent operators: advisory and issue intake, recall-dedup triage against the ledger, record-first fixes in isolated worktrees, one integration branch, an independent adversarial gate over the assembled diff, auto-merge armed only after APPROVE, and a derived release cut through the semantic-receipt gates. This intent automates that loop as abcd capability for any managed repo, with findings F-A through F-W of the pilot note as the acceptance-criteria source and F-U (re-run every required pass against the final content commit; never transfer a verdict) and F-Q (the two release gates are human-by-design) load-bearing.

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

abcd handles inbound security advisories and issues, and cuts the release, for every managed repo. The 2026-08-27 security-advisory pilot proved the loop by hand across two independent operators: advisory and issue intake, recall-dedup triage against the ledger, record-first fixes in isolated worktrees, one integration branch, an independent adversarial gate over the assembled diff, auto-merge armed only after APPROVE, and a derived release cut through the semantic-receipt gates. This intent automates that loop as abcd capability for any managed repo, with findings F-A through F-W of the pilot note as the acceptance-criteria source and F-U (re-run every required pass against the final content commit; never transfer a verdict) and F-Q (the two release gates are human-by-design) load-bearing.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
