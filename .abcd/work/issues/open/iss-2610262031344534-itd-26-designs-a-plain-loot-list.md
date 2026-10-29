---
schema_version: 1
id: "iss-2610262031344534"
slug: "itd-26-designs-a-plain-loot-list"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 148 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:46"
remedy: "Edit itd-26: drop the `loot list` sub-verb (line 46) and let bare `/abcd:loot [--source ...]` render the looted entries, which the first acceptance criterion already says bare `/abcd:loot` does."
related_intents: [itd-26]
finding_ref: cfk-263626fabca94c28
---

itd-26 designs a plain loot list sub-verb, which the naming discipline forbids

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:46` — "`loot list [--source <source>]` — show all looted entries in the current project"
- End B: `.abcd/development/brief/02-constraints/04-naming.md:100` — "`<verb> show`, `<verb> stats`, `<verb> view` and a plain unfiltered `<verb> list`"

itd-26's loot list shows all looted entries in the project, and without its optional flag it is an unfiltered list of the namespace's state. The naming chapter forbids < verb> show, < verb> stats, < verb> view and a plain unfiltered < verb> list at design time. The draft's sub-verb set contradicts the rule.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 148.
