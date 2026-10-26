---
schema_version: 1
id: "iss-2610252214517210"
slug: "11-history-md-52-54-says-show-matches-by"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S09"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md"
remedy: "Replace the bullet with a pointer to commands/history.md's Show section."
---

11-history.md:52-54 says show matches 'by session id (newest when a session has several records) or by record filename'; history.Read matches the filename, then an exact agent id, then a session's main-thread record even when a sub-agent record is newer (internal/core/history/history.go:471-497), as commands/history.md:144-147 says.
