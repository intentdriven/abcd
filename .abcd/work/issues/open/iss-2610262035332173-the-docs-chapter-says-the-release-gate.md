---
schema_version: 1
id: "iss-2610262035332173"
slug: "the-docs-chapter-says-the-release-gate"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 580 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/10-docs.md:223"
remedy: "In 10-docs.md:223 reword 'The release gate runs both' to say the gate runs the currency lint and checks the receipt of the docs-currency-reviewer, which the operator runs (01-agents.md:45)."
finding_ref: cfk-36660cb300dfd052
---

The docs chapter says the release gate runs the docs-currency-reviewer; the agent catalogue says no verb starts that reviewer: the operator runs it and the gate only checks its receipt.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/10-docs.md:223` — "The release gate runs both."
- End B: `.abcd/development/brief/05-internals/01-agents.md:45` — "for the operator to run, and `launch receipts` and the release workflow check its receipt, but no verb starts it or records its verdict"

10-docs presents the docs-currency-reviewer as the semantic complement of the currency lint and says 'The release gate runs both.' 01-agents says `launch ship` names the reviewer in a checklist for the operator to run. `launch receipts` and the release workflow only check its receipt, and no verb starts it or records its verdict. A gate that runs the reviewer and a gate that only checks a receipt the operator produced are different designs.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 580.
