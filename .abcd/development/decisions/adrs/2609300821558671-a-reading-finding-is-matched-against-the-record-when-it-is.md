---
id: adr-2609300821558671
slug: a-reading-finding-is-matched-against-the-record-when-it-is
status: accepted
date: 2026-09-30
supersedes: null
superseded_by: null
reverses: [itd-180]
related_intents: [itd-180, itd-2609212137116617]
related_rfcs: []
related_adrs: []
---

# ADR-2609300821558671: A reading finding is matched against the record when it is stored, reversing itd-180's warm-work-only rule

Typed links: `reverses` [itd-180](../../intents/shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md)
(its ruling that recurrence matching is warm work: run-scoped identifiers join
nothing mechanically); `builds_on`
[itd-2609212137116617](../../intents/shipped/itd-2609212137116617-a-new-capture-or-draft-is-matched-against-the-record-before.md)
(the filing-time match this extends to the reading family).

## Context

The filing-time match (itd-2609212137116617) compares a new issue or intent
with the record before it is written and links a likely double as
`duplicates:` or `refines:`. Ruling DQ2 of 2026-09-29 asked for the check on
every route that files a record, the reading ingest among them
(iss-2609281911024185). The inbox promote and the consistency pass took the
match without difficulty, because both file issues through `capture`.

The reading ingest does not. It writes reading records (`rdi-N`), a separate
family with a closed field list and no link key, and shipped itd-180 ruled
that spotting a recurrence is the researcher's warm work: run-scoped
identifiers join nothing mechanically, the researcher recognises a recurrence
against the ledger, and the disposition's `recurs` citation is the recorded
form of that recognition. A link the matcher wrote at ingest would be exactly
the mechanical join itd-180 forbade, so the route went back to the person as
DQ2b.

The person ruled on 2026-09-30, verbatim: "reading findings and repeats: ALL
THREE — (a) store a 'same as / builds on' link on the stored reading finding
(this REVERSES itd-180's ruling that spotting a recurrence is the researcher's
warm work, never a mechanical join; record it as a typed 'reverses' decision
against itd-180), (b) at storing time also SHOW the likely repeats, and (c)
check again at the promote step."

## Decision

We will match every reading finding against the record when it is stored,
write its likely repeats onto it, show them, and match again when an accepted
item is promoted.

1. **The link is stored on the finding.** A reading record carries the two
   typed links the filing-time match writes: `duplicates:` ("same as") and
   `refines:` ("builds on"), each a list of `iss-N`, `itd-N` or `rdi-N`. The
   reading schema's allow-list gains exactly those two keys.
2. **The match is capture's.** `reading ingest` runs the one canonical match
   (`internal/core/record/match`) with capture's threshold, link cap,
   minimum-term floor and configuration. The candidates are the open and
   resolved issues, the intents, and every reading item already in the ledger,
   so a finding a later reading returns again is linked to the item that first
   carried it. The compared text is the finding's own words: the pattern and
   the position's body, never the envelope every item of a run shares. The
   items one ingest stores are never candidates for each other.
3. **The repeats are shown.** The ingest prints each stored item's match and
   carries it in `--json` as `matches`, through the renderer the capture verb
   uses.
4. **The promote step matches again.** `capture promote <rdi-N>`, which mints
   an intent draft, compares the item's finding with the record a capture is
   compared with and writes the links onto the draft as a capture writes them.
5. **Every link is a proposal.** As on an issue, a person confirms a link by
   leaving it and removes it by deleting the line. The match never refuses a
   write, never proposes `reverses` or `supersedes`, and never names a
   disposition state. No disposition state means "already covered"; the
   researcher's `recurs` citation on a disposition stays the confirmed form of
   a recurrence, and the stored link is the machine's proposal of one.

## Alternatives Considered

- **Report the likely repeats at ingest and write no link** (the lane's
  recommendation in DQ2b). It kept itd-180 whole, but left the proposal
  nowhere durable: a repeat seen once in a terminal is lost to every later
  reader of the ledger. Rejected by the person.
- **Match only at the promote step.** The one point where a reading item
  becomes a record the match already covers, so no schema change was needed.
  It misses every finding that is never promoted, which is most of them.
  Kept as (c), not as the whole answer.
- **Keep recurrence entirely warm** (itd-180 as shipped). Rejected: the person
  ruled that the machine proposes a repeat when the finding is stored.
- **Store, show and re-check (chosen).** All three, as ruled.

## Consequences

- itd-180's ruling "recurrence matching is warm work" is reversed and its text
  says so, citing this record. The assembler still never hands a reading the
  ledger's links: a candidate projection carries two named fields only, and
  the dispositions stay excluded.
- A reading record may carry `duplicates:` and `refines:`; the writer's
  validator and the committed-tree gate accept them, and the gate resolves
  each named id.
- `reading ingest` reads the issue ledger, the intent store and the reading
  store under the ledger lock it writes the run under. `capture promote
  <rdi-N>` matches under the intent mint lock only, as `intent create` does:
  it reads the issue ledger and the reading store without the ledger lock,
  and takes that lock afterwards only to stamp the promoted item. An
  unreadable candidate set files the item unlinked and says why.
- The brief's capture and reading chapters, and the capture and reading
  command pages, state the match on this route.
