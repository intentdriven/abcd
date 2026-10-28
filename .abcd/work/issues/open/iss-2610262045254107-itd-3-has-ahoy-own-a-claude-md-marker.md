---
schema_version: 1
id: "iss-2610262045254107"
slug: "itd-3-has-ahoy-own-a-claude-md-marker"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1273 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-3-modular-rules-loader.md:33"
remedy: "none (filed automatically)"
related_intents: [itd-3, itd-2610030814013772]
finding_ref: cfk-ffccf0c207e89974
---

itd-3 has ahoy own a CLAUDE.md marker block; itd-2610030814013772 rules that projects carry no CLAUDE.md at all

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:33` — "**CLAUDE.md marker block** owned by `/abcd:ahoy`. ~30 lines."
- End B: `.abcd/development/intents/shipped/itd-2610030814013772-evaluate-whether-claude-md-can-be.md:81` — "no CLAUDE.md at all. Projects carry only AGENTS.md"

itd-3 makes a 30-line CLAUDE.md marker block, written and owned by /abcd:ahoy, the place a project learns the loader is active. itd-2610030814013772 rules that every project abcd sets up, abcd's own included, carries only AGENTS.md and that abcd writes no tool-specific conventions file. Both are shipped, so they cannot both describe what ahoy writes: a builder following itd-3 writes exactly the file the later ruling forbids.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1273.
