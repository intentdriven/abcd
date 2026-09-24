---
schema_version: 1
id: "iss-89"
slug: "capture-no-home-for-dogfood-defects"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "2026-07-13 B1 dogfood: prepare-this-repo audit of Manuscripts"
found_at: "internal/core/capture"
resolution: "An abcd defect found in a managed repository now has a home outside that repository's ledger: abcd report files it into the user account's inbox, and abcd inbox promote files it into abcd's ledger when a person or a session acts, which replaces routing captures by hand and needs no --repo flag (itd-2609221656361680)."
impact: additive
resolved_by:
  commit: "a5e5877d"
---

abcd defects found while dogfooding a target repo have no ledger home: abcd capture writes only to the cwd repo .abcd/work/issues/, so capturing an abcd bug found while onboarding repo X would either pollute X ledger (wrong repo) or require re-running with abcd-cli as cwd. Surfaced during the Manuscripts B1 dogfood -- these seven captures had to be routed to abcd-cli by hand. Seed: a capture routing option that targets the abcd repo (or an upstream/--repo flag) for tool-defect captures. Acceptance: capturing an abcd defect from within a target repo lands it in abcd ledger without a manual cd.

## Grounds

- pursued: we expect abcd defects found while working in another repository to reach abcd's ledger through the inbox rather than being hand-routed or filed into the wrong ledger; shown wrong if such findings keep arriving as chat messages or as captures in the managed repository's own ledger
