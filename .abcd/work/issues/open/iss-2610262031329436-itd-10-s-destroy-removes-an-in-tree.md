---
schema_version: 1
id: "iss-2610262031329436"
slug: "itd-10-s-destroy-removes-an-in-tree"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 142 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-10-purge-uninstall.md:16"
remedy: "none (filed automatically)"
related_intents: [itd-10]
finding_ref: cfk-6d41c22e3285c1bf
---

itd-10's destroy removes an in-tree lifeboat inside .abcd/, but the platform rules say no in-tree lifeboat home exists

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-10-purge-uninstall.md:16` — "AND the entire `.abcd/` namespace (including lifeboat, dev artefacts, logbook, and config)"
- End B: `.abcd/development/brief/02-constraints/01-platform.md:21` — "There is no in-tree lifeboat home and nothing to gitignore in the source"

itd-10 says destroy removes the entire .abcd/ namespace including lifeboat, dev artefacts, logbook and config. The platform chapter says there is no in-tree lifeboat home and nothing to gitignore in the source, because lifeboats are written out-of-tree to an operator-chosen destination. The list of what destroy deletes assumes the superseded in-tree layout.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 142.
