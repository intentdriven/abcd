---
schema_version: 1
id: "iss-2610262038521887"
slug: "itd-10-s-destroy-deletes-a-lifeboat-and"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 824 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-10-purge-uninstall.md:16"
remedy: "none (filed automatically)"
related_intents: [itd-10]
finding_ref: cfk-db5e9428ce061d51
---

itd-10's destroy deletes a lifeboat and a logbook inside .abcd/; 03-configuration says neither namespace exists in any tree

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-10-purge-uninstall.md:16` — "AND the entire `.abcd/` namespace (including lifeboat, dev artefacts, logbook, and config)"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:1367` — "Three namespaces are deliberately absent from every tree: `voyage/`, which is user-scope and never committed; `logbook/`, retired in favour of the local tier's logs; and `lifeboat/`, because the lifeboat is out-of-tree output at an operator-chosen destination."

itd-10's destroy removes the whole .abcd/ namespace, which it says includes a lifeboat and a logbook. 03-configuration says logbook/ is retired and lifeboat/ is absent from every tree, because the lifeboat is out-of-tree output. The deletion list names directories the configuration model says are never there.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 824.
