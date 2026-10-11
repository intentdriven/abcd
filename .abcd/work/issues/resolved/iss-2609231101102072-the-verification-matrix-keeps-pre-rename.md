---
schema_version: 1
id: "iss-2609231101102072"
slug: "the-verification-matrix-keeps-pre-rename"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
resolution: "The verification matrix rows read 'Intent audit (Role 1)' and 'Press-release review' (its staged half appends the lifeboat review's findings), and the two further live brief lines that called the owed queue 'owed fidelity reviews' say 'owed intent audits', matching abcd intent audit --owed. No gate would have caught it: the surface_coverage rule checks verb rows in the surface chapters, not the matrix's row titles."
impact: fix
resolved_by:
  commit: "7f292439e"
---

The verification matrix keeps pre-rename vocabulary for two renamed verbs: its rows read 'Intent fidelity review (Role 1)' and 'Press-release oracle audit', while the intent audit (itd-123, ac-2: brief prose moves to the audit vocabulary) and the lifeboat review (itd-125, ac-5: the brief's oracle-audit prose moves to review) each promised the brief's current-state prose would move. Found by the fidelity audit of both intents; the page is 06-delivery/02-verification-matrix.md.

## Grounds

- pursued: grep over .abcd/development/brief for 'fidelity review' and 'oracle audit' finds only the itd-48 Role 2 design name, which no rename touched; a current-state brief line naming the renamed verb or agent by its old spelling would show it wrong
