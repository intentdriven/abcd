---
schema_version: 1
id: "iss-2609201954342967"
slug: "the-bare-status-board-lacks-two-rows-the-record-dispatcher-a"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "overtaken-intent review with the product thinker, 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
deferred_after: "v0.11.1"
deferral_reason: "a lane of its own: the two board rows were ruled wanted by the product thinker on 2026-09-20, but they are new rendering on the bare status board (a next-actions list derived like the record dispatcher's, and each planned intent with its spec and an in-flight marker), needing their own spec and brief-chapter change; the ruling owed is when to schedule it (drain lane drainRest, run A, 2026-09-29)."
remedy: "Waits on the scheduling ruling for its lane: extend the board's Now / Next / Later block (internal/core/statusblock, rendered by internal/surface/cli/board_status.go) with each planned row's spec id and a trailing next-actions list built from the per-record next move internal/core/record derives (an owed fidelity audit, an unplanned draft with every decision recorded, a stale claim), updating the brief's board chapter (04-surfaces/08-abcd.md) in the same change, proven by a golden board test on a fixture ledger holding one of each."
related_intents: [itd-2610031214560142, itd-2610031215002409]
---

The bare status board lacks two rows the record dispatcher already answers for a single record: suggested next actions for the repository, and the planned intents with their spec and whether it is in flight. Ruled by the product thinker on 2026-09-20 while superseding itd-20 (the Python-era board built on [redacted-user]-sync and the logbook) by itd-121: those two pieces of itd-20 are still wanted and are captured here so the supersession loses nothing. Wanted: (1) the board ends with a short next-actions list derived the way abcd <record-id> derives one record's next move (an owed fidelity audit, an unplanned draft with all decisions recorded, a stale claim); (2) the board lists each planned intent with its spec id and an in-flight marker where the spec is open and its branch exists. Both are read-only rows on the existing board; no new verb.

## Remedy grounds (2026-09-29)

- The base already lists planned intents with their lane state on the board (internal/surface/cli/board_status.go:44-70, statusblock.Row carries no spec id), so the remedy extends that block rather than adding a parallel row.
- Rejected: a new verb, which the record rules out.

## Note (2026-10-03)

Folded in by the product thinker on 2026-10-03, asked to choose between folding it in, keeping it as its own lane behind both drafts, and deciding later: part (1), the next-actions list, joins the "what next?" menu (itd-2610031215002409, decision 5), and part (2), each planned intent with its spec id and in-flight marker, joins the board's view for the facilitator (itd-2610031214560142, decision 7). This issue closes when they ship.
