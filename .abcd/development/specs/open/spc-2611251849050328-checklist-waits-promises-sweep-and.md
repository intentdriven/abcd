---
id: spc-2611251849050328
slug: checklist-waits-promises-sweep-and
intent: itd-2610170638184486
origin: researcher-authored
production_mode: hand-written
---
# checklist-waits-promises-sweep-and

## Summary

The second half of itd-2610170638184486, split off from [spc-2610230805429429](../closed/spc-2610230805429429-every-abcd-session-keeps-a-checklist-on.md) when its first half (the checklist on disk, its checked write, its re-injection, the behind check and the crash lab's kill loop) landed. That spec's "Remainder" section is the design this one delivers; it is detailed here as steps. Stories 1, 4, 5, 6, 8, 9, 10 and 11 of the intent are delivered here, and the intent ships on this spec's close.

Two steps are carried from the first half, and each ends in an act only a person's hands can do (dcn-2611251849428568): the live capture in a real session, which the person's three-item walk proves (a surge session's own compaction transcript is provisional evidence only), and the two hard power cuts of the crash lab, whose kill loop is recorded in [the crash lab's study](../../research/notes/2026-11-25-checklist-crash-lab-study.md). Until the power cuts land, the checklist's pages state only what the kill loop proved.

The rest is built in the order the files allow, not the order of the stories: the checklist core first (typed promises, item kinds, the derived state, design-done), then what the stall slice reads from it (fired promises, the wake from the earliest `next_look`, the stop check's listing cache), the away record's list of the window's decisions with `abcd decide`'s away trade-off, what the person sees on return, and last the blocker sweep, which shares those files.

**Settled here** (facilitator, under facilitator.mode auto, dcn-2611241109230165):

- **Leaving without answering the sweep.** When the person sets the away mode and leaves before answering the sweep's blockers, the unattended run starts at the sweep's deadline anyway, and the unanswered blockers are queued for the person's return. The night is never lost to a missed prompt; a session that meets a queued blocker records it as waiting and moves to its next item.
- **Schema.** Every new checklist field is optional under `abcd/checklist/v1`, with no v2. The installed release's strict decode refuses a field it does not know, so no step writes the new fields into a live checkout's list before the release that reads them; tests use scratch repositories only. The same holds for the away record's new fields.
- **Old free-text promises keep reading.** The trigger grammar is enforced on a write that sets or changes a promise; a stored trigger that does not parse reads as untyped, never fires, and the render says so.
- **No new refusal code before the blocker sweep.** The checklist's new refusals reuse its existing field-error path.
- **The wake bound** is the earlier of the earliest `next_look` and the stall slice's existing deadline, so the half-lease bound stays with the code that holds it.
- **Security review** is owed on steps 4 to 7 before each lands: each adds a read or a write in the home store, or injects another working copy's text into a session's context. The checklist reader's own checks (owner, mode, untracked, no hard link) are the only way a sibling copy's list is opened.

## Steps

1. **A live capture in a real Claude Code session**
   Write three items; force `/compact` and show the context after compaction holds them; repeat with an automatic compaction on a long session. Script output alone is not the proof (story 2). The person's own three-item walk is the proof; a surge session's compaction transcript is provisional evidence only (dcn-2611251849428568). Resolves iss-2610222026223032 with its receipt.
   - needs: none
   - packages: internal/surface/cli
   - tests: internal/surface/cli
2. **The crash lab: the power-cycle receipts**
   Two hard power cuts, one on macOS and one on Linux, each cut while a writer runs, and the read after the restart recorded in the crash lab's research note: a valid checklist, last-good or the "nothing" line, never a secret (technical criterion 2, its power-cut half). A step only a person at each machine can do; until it lands, the pages state only what the kill loop proved.
   - needs: none
   - packages: internal/core/checklist
   - tests: internal/core/checklist
3. **The checklist core: typed promises, item kinds, the derived state and design-done**
   A promise's trigger is typed (a pull request merged or closed, a record moving folder, a time, a slot free) and anything else is refused naming the field, never echoing the value; an item may carry a kind (an undecided design decision, a step only a person's hands can do, a dependency on another session, a credential missing or expiring), a feature and the decision record that answered it; the list's state is derived (working if any item can move, waiting if all wait, the earliest look returned); `abcd checklist set` records a wait, a look and an answer, `add` a kind and a feature, and `abcd checklist design-done <itd>` answers yes only when every trade-off listed for the feature names a recorded answer. Tests: the trigger kinds and the refusal; an old free-text trigger still reads; the derived state; the closed list of kinds; design-done yes and no, an answer naming no decision record refused; the render leads with fired promises and the state (stories 4, 5, 9).
   - needs: none
   - packages: internal/core/checklist, internal/surface/cli, commands/checklist.md
   - tests: internal/core/checklist, internal/surface/cli
4. **The stall slice's reads: fired promises, the wake from the earliest look, the listing cache**
   Promises are checked at a session's start, at a wake and at a stop, from records already fetched, and a fired promise becomes the next item; the earliest `next_look` sets the wake, never past half the shortest reservation lease; a session whose items all wait reads as waiting, never stopped, while the person is away; the stop check's forge listing is cached under the away store and reused while younger than ten minutes, a failed read naming the cached listing's age, and a cache older than half the lease is not used. No network call is made while the person is present. Tests: a fired promise is the next item, with no process kept; a promise on an open pull request does not fire; the wake takes the earliest look and keeps the lease bound; all waiting reads as waiting; the cache's three cases (stories 4, 5).
   - needs: 3
   - packages: internal/core/autonomy, internal/surface/cli
   - tests: internal/core/autonomy, internal/surface/cli
5. **The away record keeps the window's decisions; `abcd decide` records an away trade-off**
   While the person is away, a security, speed or compatibility trade-off is decided with `abcd decide`, tagged `away-tradeoff`, with its grounds and how to undo it; every decision made in the window, from any working copy of the repository, is appended to the away record's bounded, counted list under the record's lock. The append fails open: an away record that cannot be read never fails the decide. Tests: a decide appends its id to an open away record, from a sibling worktree too; outside a window no away record is touched; an away trade-off needs its grounds and its undo; the list is bounded and counted; an unreadable away record never fails the decide (stories 8, 10).
   - needs: none
   - packages: internal/core/autonomy, internal/core/decide, internal/core/decisions, internal/surface/cli, commands/decide.md
   - tests: internal/core/autonomy, internal/core/decide, internal/surface/cli
6. **What the person finds on return: the return report, the status line and a bare `abcd mode`**
   On the first prompt after the away mode ends, once, the person is shown across every working copy of the repository the steps only their hands can do, then the window's decisions, away trade-offs first, each with its grounds and how to undo it, and who ended the mode and where; the status line names the away mode and its end; a bare `abcd mode` prints the record, the queue and what is owed. A copy another account owns is never read. Tests: the report's order across two working copies; it is shown once; it names who ended the mode; the bare verb's queue and owed items; the status line; no copy another account owns is read (stories 10, 11).
   - needs: 3, 5
   - packages: internal/core/autonomy, internal/core/statusline, internal/core/mode, internal/surface/cli, commands/mode.md
   - tests: internal/core/autonomy, internal/core/statusline, internal/surface/cli
7. **The blocker sweep**
   Setting the away mode starts the sweep: the away record gains a `sweeping` state, and a deadline, ten minutes by default and set per machine in `~/.abcd.noindex/config.json`, read through the existing reader. The sweep reads the four blocker kinds from every working copy's checklist on disk and asks live sessions, through their next hook event, to bring theirs in before the deadline; a copy whose checklist was not updated since the sweep began is listed as not reporting, with its last known blockers. At the deadline one list is put to the person, and each answer lands as a decision or a queued step. The run starts once the person has answered, or at the deadline when no answer came, the unanswered blockers queued for the person's return. The stop check does not hold a turn while the sweep runs, and a credential blocker never carries a value. Tests: setting away starts the sweep; the four kinds from every copy; a live session is asked on its next hook event; a copy not updated is not reporting; the stop check does not hold while sweeping; the run starts once answered; it starts at the deadline with no answer, the blockers queued; each answer lands as a decision or a queued step; a credential blocker carries no value (stories 1, 6).
   - needs: 3, 5, 6
   - packages: internal/core/autonomy, internal/surface/cli, commands/mode.md
   - tests: internal/core/autonomy, internal/surface/cli
8. **Docs, receipts and the close**
   The checklist and mode chapters, the decide chapter, the configuration internals and invariant 24 describe what landed, in prose that names no flag; the crash lab's note carries the power-cut receipts; the docs review is recorded for the commit the close is made on; the intent ships on this spec's close, with iss-2610222026223032 resolved by step 1's receipt.
   - needs: 1, 2, 3, 4, 5, 6, 7
   - packages: .abcd/development/brief/04-surfaces, .abcd/development/brief/05-internals, .abcd/development/brief/02-constraints/03-invariants.md, commands/checklist.md, commands/mode.md, commands/decide.md
   - tests: internal/surface/cli

## Size

About 8 days of agent work, of which the two carried steps are the person's: about ten minutes for the live capture and about twenty at each machine for the power cuts.
