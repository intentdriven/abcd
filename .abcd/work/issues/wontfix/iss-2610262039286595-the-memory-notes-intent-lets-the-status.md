---
schema_version: 1
id: "iss-2610262039286595"
slug: "the-memory-notes-intent-lets-the-status"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 862 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:73"
remedy: "none (filed automatically)"
related_intents: [itd-2610091918433290]
finding_ref: cfk-6f1b9368f7f5f46a
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. No fill writer exists in the tree, so the page is true today; the intent states its exception to the earlier ruling openly, and only the ship-time documentation edit is unplanned."
---

The memory-notes intent lets the status line write one fill file, but the configuration page says the status line writes nothing at all

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:73` — "This is an exception to the product thinker's ruling of 2026-10-05 that the status line writes nothing"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:728` — "line excepted, which writes nothing at all"

itd-2610091918433290 makes an exception to the rule that the status line writes nothing: one fill file per session under `~/.abcd.noindex/fill/`. The configuration page exempts the status line from the home-ignore step because it writes nothing at all, and its user-scope inventory has no `fill/`. One document has the status line writing a file under `~/.abcd.noindex`, the other says it writes nothing.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 862.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. No fill writer exists in the tree, so the page is true today; the intent states its exception to the earlier ruling openly, and only the ship-time documentation edit is unplanned.
