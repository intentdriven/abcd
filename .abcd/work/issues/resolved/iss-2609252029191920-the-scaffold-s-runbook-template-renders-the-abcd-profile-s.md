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
resolution: "The runbook template's opening trim marker trims after itself, so the abcd profile's Deterministic gates heading renders on its own line after a blank line."
impact: internal
resolved_by:
  commit: "a55a86d58e0fd0bb552c7c00619b22a9ab7742cc"
---

The scaffold's runbook template renders the abcd profile's 'Deterministic gates' heading glued onto the end of the rehearsal paragraph ('...first real release.## Deterministic gates (CI-enforced)'): the '<%- if not .Abcd %>' before the merge-gate and audit sections trims the blank line ahead of it, and the '<% end -%>' after them trims the newline behind, so when the block is skipped nothing separates the paragraph from the heading. The heading is then not a heading, and a gate_lockstep-style reader of the rendered runbook finds no deterministic-gate list at all. Found by a lockstep test over every rendered profile (TestRunbookGateListMatchesVerifySteps).

## Grounds

- pursued: every rendered profile's runbook parses a numbered deterministic-gate list equal to its verify job's gate steps (TestRunbookGateListMatchesVerifySteps); a rendered runbook whose heading shares a line with the paragraph before it would show it wrong
