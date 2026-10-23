---
id: spc-2610230805429429
slug: every-abcd-session-keeps-a-checklist-on
intent: itd-2610170638184486
origin: researcher-authored
production_mode: hand-written
---
# every-abcd-session-keeps-a-checklist-on

## Summary

The design record for the first half of itd-2610170638184486, planned on the product thinker's sign-off of 2026-10-23 (dcn-2610230805325396): the checklist on disk survives compaction and crashes. It resolves iss-2610222026223032. The second half is carried below as this spec's remainder and split off with `spec close --remainder` when this spec lands. Drafted on 2026-10-22 and revised after the Opus and Sonnet review round 1. Every step is test-driven: each test is watched fail first.

It builds on itd-2610180912189125 (the stall fix, #1101) and reuses, never copies, the away record, `abcd hook stop`, `internal/core/autonomy`, `hook_away.go` and the SessionStart path. It is bound by brief invariant 24 ([adr-2610222026265719](../../decisions/adrs/2610222026265719-a-session-s-checklist-names-a-credential.md)).

Each working copy keeps one checklist (dcn-2610180913444971), a JSON file in its local tier. The session writes it as it works, through `abcd checklist` sub-verbs; the stop hook and the prompt hook say when it is behind; the session-start hook re-injects the last good copy at every start and after every compaction.

**The file.** `.abcd/.work.local/checklist.json`, one per working copy, so a resumed or a fresh session in that copy reads the same list. It is JSON, never Markdown (the draft's format decision). Top level: `schema: "abcd/checklist/v1"`, `updated_at`, `updated_by` (the host's session id) and `items[]`. Each item: `id` (a short stable slug), `title` (at most 200 characters), `state` (working, waiting, done or dropped), `waits_on` (a record id, a pull request number, `person:<role>` or `slot:<kind>`), `next_look` (a timestamp), `promise` (a trigger and the act it promises), `owner_refs` (the reservations, pull requests and branches it is about), and an optional `credential` holding kind, scope and expiry only, from fixed lists and a timestamp (invariant 24). The away mode is not copied in; it is read from the repository's one away record.

**The write.** Through `internal/fsutil.WriteFileAtomic`: a temp file, an fsync, a rename, then a directory fsync. A write is refused when the JSON does not validate against the schema, or when a field matches the native secret scanner (`internal/adapter/scanner`, invariant 24); the refusal names the field and the pattern kind, never the match. A write is never refused for the file's total size; only the render is cut. After a write succeeds, the file just written (validated and scanned) is also written as `checklist.last-good.json`. The previous file is never promoted unscanned, so a file changed by anything other than the verb never becomes last-good until a verb write validates it.

**The verbs**, all in-process, none touching the network: `abcd checklist add "<title>" [--waits-on X --next-look T] [--promise "<trigger> => <act>"]`, `abcd checklist set <id> --state …`, `abcd checklist show`, and `abcd checklist render` (the injection text). The agent is told to use the verbs. A PreToolUse refusal of the file tools on the checklist path is a mistake filter only: a shell write still reaches the file. The enforcement is the reader.

**Re-injection.** The SessionStart hook renders the checklist into `additionalContext` for every source: startup, resume, clear and compact. The reader validates and scans the file; if it fails, the reader validates and scans last-good and injects it with one line saying why; if both fail, it injects nothing but that line. The render budget is the whole hook output's 10,000 characters, the host's cap (the hooks reference, read 2026-10-22), shared with the away block and the update notices the same hook already prints; the checklist gets what those leave, minus a margin of 500. Over budget, the render shows open items first, newest first, then says how many it left out and how to see them (`abcd checklist show`).

**Keeping it current.** No hook makes a session act before an automatic compaction: PreCompact can only run a command or block, and PostCompact has no decision control. So currency is held as work happens. The checklist is behind when the work the session owns, read the way the stall slice reads owed work (its reservations, its pull requests, its branches), holds a reference no item's `owner_refs` names, or has changed since `updated_at`. While the person is away, the stop hook blocks a turn's end once, inside the slice's continuation bound, naming the unlisted references. While the person is present, the stop hook's message reaches the person, not the model, so the next prompt hook adds the same note to the model's context. A step the session never recorded and that left no reference cannot be detected; that limit is in the intent's scope conditions. PreCompact records the compaction (time and trigger) in `checklist.events.jsonl` and never blocks.

**Crash safety.** A session killed mid-write leaves either the old file or the new one (the atomic rename). A torn temp file is ignored and removed on the next write. The crash claim is confirmed by a lab, not asserted: a writer killed at random points 1,000 times, and two hard power-cycles on macOS and Linux; in each case the reader gets a valid checklist, last-good, or the "nothing" line, and never a secret.

**Settled here** (facilitator, on state-of-the-art grounds or earlier records): one file per working copy (dcn-2610180913444971); JSON with a schema and typed fields, which also lets the secret check run field by field; the render shares the hook's 10,000-character cap with everything else the hook prints, since the host caps the whole string and swaps an over-long one for a preview; the reader is the enforcement, and the file-tool refusal is a mistake filter on which no claim rests.

## Footprint

- packages: internal/core/checklist, internal/surface/cli, internal/core/autonomy, internal/core/guard, hooks, commands/checklist.md, .abcd/development/brief/04-surfaces, .abcd/development/brief/05-internals
- tests: internal/core/checklist, internal/surface/cli, internal/core/autonomy, internal/core/guard

## Steps

1. **Schema, store and atomic write**
   Tests: round-trip; a torn temp file is ignored; a schema violation is refused; a secret in each field kind is refused and the match is never echoed; last-good is written only from a validated, scanned write; a hand-edited file with a secret never becomes last-good (technical criterion 3, story 7).
   - needs: none
   - packages: internal/core/checklist
   - tests: internal/core/checklist
2. **The `abcd checklist` verbs**
   Tests: each verb and its JSON output, and the generated surface chapter.
   - needs: 1
   - packages: internal/surface/cli, commands/checklist.md
   - tests: internal/surface/cli
3. **Render and injection for all four sources**
   Tests: each source injects; the budget is computed against the hook's whole output with the away block present; over budget cuts and says so; a corrupt or secret-bearing file falls back to a scanned last-good; both failing inject only the line (technical criteria 1 and 3, story 2).
   - needs: 1
   - packages: internal/core/checklist, internal/surface/cli
   - tests: internal/core/checklist, internal/surface/cli
4. **The behind check**
   Tests: an unlisted reservation, pull request or branch blocks once while away, naming it; the slice's bound holds; while present, the next prompt carries the note and the stop is not blocked; no network call while present (story 3).
   - needs: 1
   - packages: internal/core/autonomy, internal/surface/cli
   - tests: internal/core/autonomy, internal/surface/cli
5. **The file-tool refusal on the checklist path**
   Tests: each file tool is refused; a shell write is not refused, and the reader catches it (step 3's tests).
   - needs: none
   - packages: internal/core/guard, hooks
   - tests: internal/core/guard
6. **The PreCompact event sidecar**
   Tests: it records, and it never blocks.
   - needs: 1
   - packages: internal/surface/cli, hooks
   - tests: internal/surface/cli
7. **The crash lab**
   The kill loop and the power-cycle receipts, recorded in a dated research note (technical criterion 2).
   - needs: 1
   - packages: internal/core/checklist
   - tests: internal/core/checklist
8. **A live capture in a real Claude Code session**
   Write three items; force `/compact` and show the context after compaction holds them; repeat with an automatic compaction on a long session. Script output alone is not the proof (story 2).
   - needs: 2, 3
   - packages: internal/surface/cli
   - tests: internal/surface/cli
9. **Docs and brief**
   The command page, a new surface chapter and the format in the internals, then `go generate ./internal/surface/cli`; invariant 24 is rewritten to the present tense, naming the tests that hold it.
   - needs: 2, 3, 4, 5, 6
   - packages: commands/checklist.md, .abcd/development/brief/04-surfaces, .abcd/development/brief/05-internals, .abcd/development/brief/02-constraints/03-invariants.md
   - tests: internal/surface/cli

## Remainder: waits, promises, wakes, the blocker sweep and the return lists

Split off with `spec close --remainder` when the steps above land, and detailed then. It delivers the rest of the intent on this spec's file (stories 1, 4, 5, 6, 8, 9, 10 and 11):

- **Waits.** The earliest `next_look` sets the stall slice's wake, bounded by half the shortest reservation lease. The state is derived: working, if any item can move; waiting, if all items wait; stopped, only while the person is present.
- **Promises.** `--promise "pr:42 merged => add the wrapper"`. The trigger kinds are a pull request merged or closed, a record moving folder, a time, and a slot free. They are checked at start, at a wake and at a stop, from records already fetched; no process is kept running. A fired promise becomes the next item.
- **The blocker sweep** (dcn-2610222025217495). Setting the away mode starts it. The away record gains a `sweeping` state, done in the remainder's first step. The sweep has a deadline, 10 minutes by default and set per machine. It reads the four blocker kinds (an undecided design decision, a step only a person's hands can do, a dependency on another session, a credential missing or expiring) from every working copy's checklist on disk, and asks live sessions, through their next hook event, to bring theirs in before the deadline. At the deadline the orchestrator puts one list to the person; a copy whose checklist was not updated since the sweep began is listed as "not reporting", with its last known blockers; the unattended run starts once the person has answered, and each answer lands as a decision or a queued step.
- **Overnight trade-offs** (dcn-2610222025225277). While the person is away, a security, speed or compatibility trade-off is decided with `abcd decide`, tagged `away-tradeoff`, with its grounds and "to undo:". The return list shows these first.
- **The facilitator's mode** reads the existing `facilitator.mode` of itd-2610150602547271; no second toggle is added.
- **Design-done.** `abcd checklist design-done <itd>` answers yes only when every genuine trade-off listed for the feature has a recorded product-thinker answer.
- **The return lists**, shown on the first prompt after the away mode ends, across every working copy of the repository: the steps only the person's hands can do, and the window's decisions, each with its grounds and how to undo it.
- **Work moved back from the stall slice:** the return report, the window's decisions across working copies, the status line, a bare `abcd mode`, the guard's `go run ./cmd/abcd` reading, and the stop check's listing cache.

Footprint: this spec's, plus `internal/core/mode`, `internal/core/decide` and the away record's store. Size: about 8 days, after this spec.

## Size

About 5 days of agent work for the steps above.
