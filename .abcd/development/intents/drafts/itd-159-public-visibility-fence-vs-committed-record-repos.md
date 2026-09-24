---
id: itd-159
slug: public-visibility-fence-vs-committed-record-repos
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-223]
---

# the managed .gitignore visibility table has no mode for a public repo that commits its record: on abcd-cli (public, single-repo-curated-release, .abcd/** deliberately in-tree) ahoy install applied the public policy and fenced /.abcd/ and /memory/, contradicting the repo's own boundary that the record is present in every checkout. The visibility table needs a committed-record declaration (config or marker) that suppresses the fence

## Press Release

> _Seeded by promotion from iss-223. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-223`: the managed .gitignore visibility table has no mode for a public repo that commits its record: on abcd-cli (public, single-repo-curated-release, .abcd/** deliberately in-tree) ahoy install applied the public policy and fenced /.abcd/ and /memory/, contradicting the repo's own boundary that the record is present in every checkout. The visibility table needs a committed-record declaration (config or marker) that suppresses the fence. Read that issue record for the source observation.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
