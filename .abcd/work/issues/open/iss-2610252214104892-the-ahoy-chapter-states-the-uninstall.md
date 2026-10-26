---
schema_version: 1
id: "iss-2610252214104892"
slug: "the-ahoy-chapter-states-the-uninstall"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S01"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md"
remedy: "Replace the uninstall paragraph at 01-ahoy.md:1194-1204 with a pointer to commands/ahoy.md § uninstall, or restate the three writes it makes."
---

The ahoy chapter states the uninstall leaves 'the entire .abcd/ namespace and the history store intact' and omits the README parts and status-line hand-back, while internal/core/ahoy/apply.go:2163-2169, the chapter's own line 65-70 and commands/ahoy.md:1014-1036 say it removes README parts, hands the status line back and clears readme.created.
