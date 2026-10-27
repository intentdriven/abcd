---
schema_version: 1
id: "iss-2610262031516342"
slug: "the-personas-page-forbids-generic"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 187 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/05-personas.md:5"
remedy: "none (filed automatically)"
related_intents: [itd-64]
finding_ref: cfk-ccc801910cf7caca
---

The personas page forbids generic, unnamed quote attributions, but itd-64's customer quote is credited to 'a maintainer'.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/05-personas.md:5` — "never use generic "a hypothetical user" phrasing (loses voice)"
- End B: `.abcd/development/intents/drafts/itd-64-benchmark-driven-config-optimization.md:20` — "this agent-instruction revision," said a maintainer."

05-personas codifies that press-release quotes use named placeholders from the registry, picked by role, and never generic phrasing, because that loses voice. itd-64's customer quote is credited to 'a maintainer' with no registered name.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 187.
