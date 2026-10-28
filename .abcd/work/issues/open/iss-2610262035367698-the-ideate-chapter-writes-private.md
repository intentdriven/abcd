---
schema_version: 1
id: "iss-2610262035367698"
slug: "the-ideate-chapter-writes-private"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 583 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/18-ideate.md:88"
remedy: "none (filed automatically)"
finding_ref: cfk-cfff9aff74f1b24a
---

The ideate chapter writes private verdicts under `~/.abcd.noindex/ideas/<root-sha>/`; the configuration chapter's inventory of the user scope has no ideas store.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/18-ideate.md:88` — "`~/.abcd.noindex/ideas/<root-sha>/` and nothing else: no decision points at it,"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:621` — "**User scope, `~/.abcd.noindex/`** — one per machine, machine-local shared state only: the"

18-ideate says a private verdict is written owner-only under `~/.abcd.noindex/ideas/<root-sha>/`. 03-configuration enumerates what the user scope holds and says that enumeration is the one list. It names no `ideas/` store, so the inventory and the verb disagree about what lives in the user scope.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 583.
