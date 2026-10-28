---
schema_version: 1
id: "iss-2610262038197762"
slug: "universal-patterns-say-the-log-tier-and"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 787 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md:161"
remedy: "none (filed automatically)"
finding_ref: cfk-562cfd830d0ec912
---

Universal patterns say the log tier and the voyage store are both tracked under the visibility rule; the voyage entry says the voyage is never committed.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:161` — "Both are tracked under the visibility rule; they answer different questions"
- End B: `.abcd/development/brief/glossary/core/voyage.md:24` — "never inside the source repository and therefore never"

04-universal-patterns says both `.work.local/logs/` and `~/.abcd.noindex/voyage/` are tracked under the visibility rule, which commits them in private repositories. The voyage glossary entry puts the voyage at the operator level, outside every repository and never committed, and the local tier is gitignored under every visibility. Neither store can be tracked by a visibility switch.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 787.
