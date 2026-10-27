---
schema_version: 1
id: "iss-2610262031514386"
slug: "the-personas-page-forbids-generic"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 186 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/05-personas.md:5"
remedy: "none (filed automatically)"
related_intents: [itd-57]
finding_ref: cfk-f2b059c305a016b9
---

The personas page forbids generic, unnamed quote attributions, but itd-57's customer quote is credited to 'a facilitator'.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/05-personas.md:5` — "never use generic "a hypothetical user" phrasing (loses voice)"
- End B: `.abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:19` — "behind my back," said a facilitator."

05-personas codifies that press-release quotes use named placeholders from the registry, picked by role, and never generic phrasing, because that loses voice. itd-57's customer quote is credited to 'a facilitator' with no registered name.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 186.
