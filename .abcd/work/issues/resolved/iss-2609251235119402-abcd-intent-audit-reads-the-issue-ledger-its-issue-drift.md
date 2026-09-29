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
resolution: "runIssueDrift names the checkout and branch whose ledger it read, with ledgerIdentityOf and renderLedger: a stderr line in the text render and a ledger member on the --json envelope, as every capture verb does."
impact: fix
resolved_by:
  commit: "62140a148"
---

abcd intent audit reads the issue ledger (its issue-drift form checks the promote join) without naming which checkout's ledger and branch it read. Decomposed out of iss-2609202053570475, which made every capture verb and the record dispatcher name the ledger they addressed: the audit's output is the intent-auditor verdict envelope, owned by the intent surface, so adding the member there is that surface's change. The helpers to reuse are ledgerIdentityOf and renderLedger in internal/surface/cli.

## Grounds

- pursued: we expect intent audit --issue-drift to name the ledger's checkout and branch in both renders; a run that reads the ledger without naming it would show it wrong
