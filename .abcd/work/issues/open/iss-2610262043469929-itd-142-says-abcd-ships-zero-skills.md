---
schema_version: 1
id: "iss-2610262043469929"
slug: "itd-142-says-abcd-ships-zero-skills"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1160 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-142-the-brief-creation-interview-abcd.md:13"
remedy: "none (filed automatically)"
related_intents: [itd-142, itd-43]
finding_ref: cfk-84b346a2e3429e80
---

itd-142 says abcd ships zero skills; itd-43 still treats the grill SKILL.md as a live surface to rewrite

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-142-the-brief-creation-interview-abcd.md:13` — "Command-shaped, because it mutates state — abcd ships zero skills"
- End B: `.abcd/development/intents/shipped/itd-43-epic-to-spec-terminology.md:44` — "the grill `SKILL.md` boundary message"

itd-142 makes the interview a command because abcd ships zero skills. itd-43 lists the grill SKILL.md boundary message among the surfaces its prose sweep rewrites. It does not mark that file moot, as it does other files that are absent from the tree. The record cannot both say abcd ships no skills and keep a grill skill file as a surface to maintain.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1160.
