---
id: adr-2609292012006845
slug: now-next-and-later-list-an-intent-in-a
status: accepted
date: 2026-09-29
supersedes: adr-2609212115255771
superseded_by: null
related_intents: [itd-2609212103568351, itd-2609211913453478, itd-2609212103565953, itd-2609212103572513]
related_rfcs: []
related_adrs: [adr-2609212115255771, adr-9]
---

# ADR-2609292012006845: Now, Next and Later list an intent in a lane under Now only; phases stay retired

## Context

adr-2609212115255771 (2026-09-21) retired phases, milestones and the word
roadmap, and its decision 2 defined the rendered Now / Next / Later block:
"Next is every planned intent the gate reports READY; Later is the rest of
planned and the drafts". The block shipped as itd-2609212103568351, and an
intent the build had in a lane then appeared twice: under Now with its lane
state, and again under Next (or Later) where the gate placed it.

On 2026-09-29 the product thinker ruled (DECISIONS.md entry landing with lane
recRulings) that an intent in progress appears only under Now, never also
under Next or Later (BV2), and that the text board shows Later as a count
while the site and `--json` keep the full list (BV1). BV2 makes decision 2's
text false. The same day's ruling H10 settles how such a change is recorded:
an issue names it (iss-2609292011569133), the shipped intent gains an Audit
Notes line, and an ADR whose decision text is now false is superseded rather
than amended in place. This record is that successor. It changes decision 2
only; every other decision stands as adr-2609212115255771 stated it, and is
carried here so the record in force states the whole decision.

## Decision

We retire the phase and the milestone as units of the record, and the word
roadmap with them. Decisions 1 and 3 to 8 are adr-2609212115255771's,
carried forward word for word; decision 2 is revised.

1. **Sequencing is dependencies plus the shelves.** `blocked_by` and
   `builds_on` on the records (itd-78 derives a priority from them; the pick
   in `abcd build next` filters on them) and the lifecycle shelves
   `drafts/ → planned/ → shipped/` carry the order. No stored unit sits above
   the intent for sequence.
2. **Now / Next / Later is a rendered status, never stored.** The bare `abcd`
   board and the site's Status page compute it: Now is every intent in a
   lane (from the build's state file) plus the head of the pick order; Next
   is every planned intent the gate reports READY that is not in a lane;
   Later is the rest of planned and the drafts, again leaving out an intent
   in a lane. An intent in a lane is listed under Now only (BV2). The text
   board gives Later as a count, and the JSON and the Status page list its
   rows (BV1). A started state comes from the state file and is rendered
   (itd-2609212103568351). Nothing is called a roadmap.
3. **The checkpoint is the derived release plus each intent's acceptance
   criteria.** No planned end condition exists apart from them. An intent
   may carry an optional `target_release:` that the cut reports and moves
   forward, never refuses on (itd-2609212103572513); that field is the one
   forward-looking line the derived release keeps.
4. **The unit below a spec is the step**: an ordered `## Steps` section in
   the spec, each step landable as one lane and one pull request; a spec
   with no steps is one step; the loop's `implement step` and the section
   share the word (itd-2609212103565953). No fourth record family.
5. **Two axes, kept apart.** The lifecycle (what has been decided about a
   record) lives in the folders and gates; the position (how soon) is
   rendered from it. `planned/` is not renamed Now, because sixty planned
   intents are not all being worked on and the word would lie.
6. **Issues carry no spec by design.** An issue's design record is its
   `remedy:` field and the failing test its lane writes first; an issue that
   needs design is an intent in disguise and is promoted.
7. **The batch is the run's internal order**, derived from dependencies and
   the pick, and is not a term of the record.
8. **One term per concept.** The glossary marks `phase`, `milestone` and
   `roadmap` superseded with their successors named, gains `bundle` and
   `step`, and one page maps the families (itd-2609211913453478). The
   phase documents stay in the tree as history with a retirement line; the
   ROADMAP rule domain is replaced by this decision's statement.

## Alternatives Considered

- **Amend adr-2609212115255771's decision 2 in place.** Rejected by ruling
  H10: an ADR is not edited after it is accepted, and a decision whose text
  is false is superseded.
- **Supersede decision 2 alone, leaving the other seven in force in the old
  record.** Rejected: the record has no partial supersession, and a record
  marked superseded whose other decisions still bind would read as retired
  to a reader following `superseded_by`. Carrying decisions 1 and 3 to 8
  forward word for word keeps one record in force, as adr-2609151706587280
  did for adr-46.
- **Keep the intent in a lane under Next as well.** Rejected by ruling BV2:
  listing one intent twice says it is both being built and waiting to be.

## Consequences

- adr-2609212115255771 is superseded and retained: later records cite its
  numbered decisions, and those numbers mean the same here.
- `statusblock.Read` leaves an intent in a lane out of Next and Later, so the
  board, `--json` and the site's Status page follow from one read; removing
  the state file returns each such intent to the list the gate places it in.
- itd-2609212103568351's criteria 1 and 3 are read with the Audit Notes line
  the change adds to it; the criterion text is not edited.
