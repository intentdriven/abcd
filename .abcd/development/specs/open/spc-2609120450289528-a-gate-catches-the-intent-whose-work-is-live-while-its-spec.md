---
id: spc-2609120450289528
slug: a-gate-catches-the-intent-whose-work-is-live-while-its-spec
intent: itd-2609111003026787
origin: researcher-authored
production_mode: hand-written
---
# A gate catches the intent whose work is live while its spec is still open

## Summary

A change that delivers an intent says so in its commit trailer, and a merge-time
gate refuses the change unless that intent reaches `shipped/` in the same diff.

This is the symmetric twin of RS001. The issue ledger has been held honest by
`Resolves: iss-N` plus a gate that refuses a trailer whose record does not move;
the intent store has the same hole and no such gate, and 61 planned intents with
open specs are what came of it.

Nothing infers that work is live. The author declares it, and the gate enforces
the declaration — which is the whole reason this is buildable, since specs carry
no task state and intents carry no delivering-commit stamp.

## Scope

**In.** One trailer, one rule, and the refusal messages that make it actionable.

**Out.** `intent ship` and the intent supersede verb, held by the 2026-09-11
ruling until this gate has run a cycle. Clearing the standing 61, which this gate
does not reach. Any change to the closed-spec half (`staleIntents`), which stays
at release time and keeps working as it does.

## Approach

### The trailer

```
Delivers: itd-N
```

Matched exactly as RS001 matches its own, with the id pattern widened to accept
both id shapes the store now holds:

```bash
DELIVERS_RE='^Delivers:[[:space:]]+(itd-[0-9]+)[[:space:]]*$'
```

`itd-[0-9]+` already covers the sequential ids and the minted timestamp ids
alike, so no second pattern is needed.

The verb is `Delivers` rather than `Resolves` because the acts differ: an issue
is *resolved* by the change that fixes it, an intent is *delivered* by the change
that makes its acceptance criteria true. Reusing `Resolves` for both would make
the trailer ambiguous about which store to look in.

### The rule

A sibling of `scripts/check-issue-resolution.sh`, sharing its structure so the
two read as one convention:

1. Compute the set of intent ids **entering `shipped/`** between base and head —
   the mirror of `ids_entering_closed`, over
   `.abcd/development/intents/shipped/`.
2. Walk every commit in the range and collect `Delivers:` trailers.
3. A declared id that is not in the entering set is a refusal, and the
   *diagnosis* is what the message carries.

### The refusal shapes

RS001's lesson, learnt the hard way on a branch 235 commits behind main, is that
one refusal text for several causes is a refusal nobody can act on. The same
diagnoses apply here and are stated separately:

| Situation | Diagnosis in the message |
| --- | --- |
| No record at head, but the base holds it | The branch predates the record: rebase, then close its spec in this change |
| No record at head or base | The id is wrong, or the intent was never filed |
| Already in `shipped/` at the base | The trailer names an intent delivered before this change: drop the trailer |
| In `planned/` at head, spec still open | The ordinary case: run `abcd spec close <spc-N>` in this change |
| In `planned/` at head, `spec_id: null` | The intent has no spec to close; the message says so rather than naming a command that cannot run |

The last row matters because 20 planned intents carry `spec_id: null` and no verb
mints one for them. A refusal that told those authors to run `spec close` would
name a command with no argument.

### Where it runs

Beside the issue gate, at merge time, in the `record-lint` job that already runs
`check-issue-resolution.sh`. Not at the cut: the trailer is written by the author
of the change, so the refusal belongs where that author is still holding it. This
is scope condition 2 made concrete.

## How this satisfies each acceptance criterion

1. **A declared delivery must move the record.** Step 3 above: a `Delivers:` id
   absent from the entering-`shipped/` set is a refusal, and the `planned/`+open
   spec row names `abcd spec close <spc-N>`.
2. **Every unshipped one is named.** The walk collects all trailers before
   judging and emits one refusal per offending id, as RS001 does, rather than
   exiting on the first.
3. **A change declaring nothing is refused nothing.** The rule is keyed on the
   trailer: no trailer, no assertion to check. This is the gate's stated limit,
   and the criterion pins it so a later change cannot quietly widen the rule into
   refusing every planned intent.
4. **A bad id says which.** Rows 1-3 of the table: absent at head, absent
   everywhere, and already terminal are three distinct messages.
5. **Same message shape and exit code as the issue rule.** The rule reuses the
   issue gate's `fail` helper and its exit convention, and a test asserts the two
   refusals share their shape rather than a reviewer judging they feel alike.
6. **The 61 are not refused.** They carry no trailer, so nothing fires on them.
   The `## Scope` section above states that clearing them is a separate act, so
   the criterion is met by what the gate does *not* do, and by the record saying
   so.

## Risks

**The gate is keyed on speech, and the measured problem is silence.** Recorded on
the intent as the mechanism's own falsifier: authors omitting the trailer to
avoid the refusal would show the cause was reluctance rather than forgetting, and
a voluntary declaration cannot reach someone who declines to declare. The cycle
this gate runs before the verbs are reconsidered is what tests it.

**Partial delivery.** Scope condition 3: where an intent arrives across several
changes, no single one finishes it. The trailer must therefore mean "this
finishes it" rather than "this contributes to it", and the wording in
`CONTRIBUTING.md` has to say so, or the gate will fire on the first of five pull
requests and teach authors to stop writing it.

## Open

- The rule's identifier. RS001-RS003 are the issue gates; this needs its own,
  and whether it joins that numbering or starts a sibling series is a naming
  decision the implementer should not take alone.
- Whether the trailer also accepts a spec id (`Delivers: spc-N`). The intent is
  the thing delivered, but the spec is the thing closed, and an author holding
  the spec id would have to look up the intent.
