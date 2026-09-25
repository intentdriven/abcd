---
schema_version: 1
id: "iss-2609252029191920"
slug: "the-scaffold-s-runbook-template-renders-the-abcd-profile-s"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/scaffold/templates/runbook.md.tmpl"
---

The scaffold's runbook template renders the abcd profile's 'Deterministic gates' heading glued onto the end of the rehearsal paragraph ('...first real release.## Deterministic gates (CI-enforced)'): the '<%- if not .Abcd %>' before the merge-gate and audit sections trims the blank line ahead of it, and the '<% end -%>' after them trims the newline behind, so when the block is skipped nothing separates the paragraph from the heading. The heading is then not a heading, and a gate_lockstep-style reader of the rendered runbook finds no deterministic-gate list at all. Found by a lockstep test over every rendered profile (TestRunbookGateListMatchesVerifySteps).
