---
schema_version: 1
id: "iss-2610262043135705"
slug: "the-product-thinker-persona-is-alice-in"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1123 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-106-abcd-sets-up-the-ci-a-repo-requires-and.md:30"
remedy: "itd-106 line 30: change 'said Alice, a product thinker' to 'said Iris, a product thinker'"
related_intents: [itd-106, itd-135]
finding_ref: cfk-be2154f2456add56
resolution: "itd-106 line 30 now attributes the product-thinker quote to Iris, the name the persona registry gives that role, matching itd-135 and itd-104."
impact: internal
resolved_by:
  commit: "c4fb3355d100b2be7f0a78cc2faa924a0f2831e9"
---

The product-thinker persona is Alice in itd-106 and Iris in itd-135.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-106-abcd-sets-up-the-ci-a-repo-requires-and.md:30` — "said Alice, a product thinker"
- End B: `.abcd/development/intents/shipped/itd-135-iris-opens-abcdev-app-and-understands.md:34` — "said Iris, a product thinker"

Under the persona registry rule the role picks the name, so one role carries one registered name. itd-106 attributes its quote to 'Alice, a product thinker'; itd-135 (and itd-104) attribute theirs to 'Iris, a product thinker'. Two names are in use for one role, so at least one record picked by name rather than by role.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1123.

## Grounds

- pursued: itd-106:30 reads 'said Iris, a product thinker' and personas.json maps 'product thinker' to Iris; a registry that gave the role to another name, or any remaining 'said Alice, a product thinker' in itd-106, would show it wrong.
