---
schema_version: 1
id: "iss-2610262032238170"
slug: "ahoy-presents-its-abcd-noindex-tree-as"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 262 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:526"
remedy: "none (filed automatically)"
finding_ref: cfk-c17ab4808f07d5a6
---

Ahoy presents its ~/.abcd.noindex tree as the one inventory; the board's session naming writes a sessions/names.json register it does not list

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:526` — "the two are one list and must agree"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:190` — "reserves the next free name in the machine register, `~/.abcd.noindex/sessions/names.json`"

Ahoy lays out the user-scope directory and says its tree and the configuration chapter's prose are one list and must agree. The board chapter has the session-start hook reserve names in a machine register at ~/.abcd.noindex/sessions/names.json, which appears nowhere in ahoy's inventory. Either the inventory is incomplete or the register lives somewhere the inventory does name.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 262.
