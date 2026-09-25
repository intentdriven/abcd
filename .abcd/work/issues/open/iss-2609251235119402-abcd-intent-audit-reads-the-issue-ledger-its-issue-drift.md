---
schema_version: 1
id: "iss-2609251235119402"
slug: "abcd-intent-audit-reads-the-issue-ledger-its-issue-drift"
severity: "nitpick"
category: "ux"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
---

abcd intent audit reads the issue ledger (its issue-drift form checks the promote join) without naming which checkout's ledger and branch it read. Decomposed out of iss-2609202053570475, which made every capture verb and the record dispatcher name the ledger they addressed: the audit's output is the intent-auditor verdict envelope, owned by the intent surface, so adding the member there is that surface's change. The helpers to reuse are ledgerIdentityOf and renderLedger in internal/surface/cli.
