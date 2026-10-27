---
schema_version: 1
id: "iss-2610262035354961"
slug: "the-mode-chapter-keeps-the-away-record"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 582 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/26-mode.md:36"
remedy: "none (filed automatically)"
finding_ref: cfk-0f2c193c44710bdc
---

The mode chapter keeps the away record in `~/.abcd.noindex/away/<root-sha>/`; the configuration chapter's inventory of the user scope has no away store.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/26-mode.md:36` — "lives in `~/.abcd.noindex/away/<root-sha>/`, keyed on the root commit as the worktree"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:621` — "**User scope, `~/.abcd.noindex/`** — one per machine, machine-local shared state only: the"

26-mode says the away record lives in `~/.abcd.noindex/away/<root-sha>/`, keyed on the root commit. 03-configuration lists the user scope's contents store by store and file by file, and says that list and the ahoy chapter's tree are one list and must agree, yet it names no `away/` store. A store that one chapter writes and the authoritative inventory omits is missed by anything built from the inventory, such as a backup or a cleanup.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 582.
