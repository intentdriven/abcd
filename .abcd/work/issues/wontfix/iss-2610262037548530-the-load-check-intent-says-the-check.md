---
schema_version: 1
id: "iss-2610262037548530"
slug: "the-load-check-intent-says-the-check"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 757 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/27-implement.md:266"
remedy: "none (filed automatically)"
related_intents: [itd-2609231434459890]
finding_ref: cfk-2376126d574706db
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both were true when written: the intent shipped a warning-only check, then a resolved issue added 'implement load --wait' as a separate waiter, which the brief and code now carry."
---

The load-check intent says the check never refuses and never waits; the implement chapter's load check, with its wait flag, holds make preflight's test stages until the load falls.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/27-implement.md:266` — "`make preflight` runs its own test stages through it: it blocks until the one-minute load is below the wait limit"
- End B: `.abcd/development/intents/shipped/itd-2609231434459890-abcd-s-own-test-lanes-check-the-machine.md:84` — "The check never refuses and never waits."

itd-2609231434459890 puts refusing and waiting out of scope outright: the check never refuses and never waits. 27-implement.md keeps that only for the plain form. With its wait flag the same check is the waiter concurrent sessions gate a preflight with, make preflight runs its test stages through it, and it blocks until the one-minute load is below the wait limit, exiting 1 when it gives up at its timeout. A preflight on a loaded machine carries on by the intent and is held by the brief.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 757.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both were true when written: the intent shipped a warning-only check, then a resolved issue added 'implement load --wait' as a separate waiter, which the brief and code now carry.
