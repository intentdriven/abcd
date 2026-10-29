---
schema_version: 1
id: "iss-2610262032282645"
slug: "ahoy-s-inventory-of-abcd-noindex-has-no"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 272 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:420"
remedy: "Add the away/ store (one folder per root-commit SHA holding the away record and its lock) to the user-scope tree in 01-ahoy.md and to the mirrored inventory prose in 05-internals/03-configuration.md."
finding_ref: cfk-3df0b743fcf4ed75
---

ahoy's inventory of ~/.abcd.noindex/ has no away store; mode keeps the away record in ~/.abcd.noindex/away/< root-sha>/

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:420` — "USER SCOPE — one per machine (machine-local state only)"
- End B: `.abcd/development/brief/04-surfaces/26-mode.md:35` — "The record lives in `~/.abcd.noindex/away/<root-sha>/`, keyed on the root commit as the worktree store is."

01-ahoy.md presents the user-scope tree as the inventory of what abcd keeps under ~/.abcd.noindex/, one list that must agree with the configuration chapter, and it lists no away/ directory. 26-mode.md says the away record and its lock live in ~/.abcd.noindex/away/< root-sha>/, and the guard's created-branch recorder writes into the same store. Either the inventory is incomplete, or the away store lives somewhere else.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 272.
