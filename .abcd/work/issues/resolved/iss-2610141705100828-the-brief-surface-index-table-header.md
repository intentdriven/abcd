---
schema_version: 1
id: "iss-2610141705100828"
slug: "the-brief-surface-index-table-header"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.1 release review, iss35 brief-surface crosscheck (Direction B), content commit 74165cf97"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Mark the Memory section staged under its intent, or drop the /abcd:memory name from the header."
resolution: "No longer true: abcd memory ships (commands/memory.md, brief chapter 37-memory.md), merged in #1003, so the surfaces index's Memory (/abcd:memory) header names a real command."
impact: internal
resolved_by:
  commit: "3a7e1eddd"
---

The brief surface index table header reads Memory (/abcd:memory) and is not marked staged, but no /abcd:memory command exists: there is no commands/memory.md and no memory verb, and 07-library.md:11-16 says the old memory command was folded into library with its name not kept; memory notes are an open intent (itd-2610091918433290).
