---
schema_version: 1
id: "iss-317"
slug: "development-readme-conventions-sentence"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: ".abcd/development/README.md"
resolution: "development/README.md conventions line rewritten to match adr-32 (ledger is working-tier data)."
impact: internal
---

development README conventions sentence says issues graduate rather than a ledger contradicting accepted adr-32 and the live work-tier ledger
## Evidence
`.abcd/development/README.md:23-24` (Conventions): "Issues graduate into intents/ or principles/ rather than a ledger." Contradicts accepted ADR-32 (`decisions/adrs/0032-issue-ledger-is-working-tier-data.md`: "The issue ledger lives in the work tier — .abcd/work/issues/"), the ~290-entry ledger, and record-lint rules (issue_id_unique/issue_impact_valid/record_schema pointing at `.abcd/work/issues`).

## Adversarial verdict: CONFIRMED (minor)
The paragraph names all three tiers, so it is not scoped to the development/ dir; its pre-ADR-32 ancestor (`plans/2026-07-06-go-rebuild.md:142`) carried the scoping clause "Issues are not a record folder" and the README compressed it into a mechanism denial that is now false. Observed 2026-07-08 (research note :119), routed to iss-36/iss-38, both resolved WITHOUT fixing this line — resolved-but-unfixed, no open item covers it. Fix: replace with a sentence consistent with adr-32 (the ledger is working-tier data at ../work/issues/, and a design-significant issue graduates from it into intents/ or principles/).
