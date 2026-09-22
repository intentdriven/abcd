---
id: adr-2609212115255771
slug: phases-and-milestones-are-retired-sequencing-is-dependencies
status: accepted
date: 2026-09-21
supersedes: adr-9
superseded_by: null
related_intents: [itd-2609211913453478, itd-2609212103568351, itd-2609212103565953, itd-2609212103572513, itd-24, itd-34, itd-78]
related_rfcs: []
related_adrs: [adr-1, adr-9, adr-45]
---

# ADR-2609212115255771: Phases and milestones are retired: sequencing is dependencies rendered as status, the checkpoint is the derived release

## Context

adr-9 (2026-05-16) made the phase the product layer between the brief and the
intent: an ordered stretch of work bundling intents, ending in a milestone, a
checkable end condition, recorded as a document under `roadmap/phases/`. Four
months on, the record shows the layer thinned to a document nobody anchors to:
no spec carries a `phase:` field (0 of 93), phase membership is recorded
editorially in the phase document, and the roadmap rule "each phase ends in a
milestone" names a unit no verb reads. Meanwhile three other groupings arrived
with verbs behind them: the bundle (several intents sharing one spec because
they ship as one change, itd-34), the derived release (the version computed
from the impact of what shipped, the changelog composed from the records), and
the autonomous run's batch. On 2026-09-21 the product thinker asked, in the
interview that planned the fourteen specless intents, whether phases survive
bundles and milestones survive releases.

An independent research pass over the field (Shape Up, Linear, GitHub,
GitLab, SAFe, the Kanban Guide 2025, DORA, Firefox's trains, Kubernetes
enhancement proposals, Now/Next/Later roadmaps, Cagan, release-please and
changesets) found: every surviving sequencing unit is a timebox whose job is
to carry a date, which this repository's roadmap rule forbids; the undated
lineages run on priority, readiness and dependencies and name no sequencing
unit; where a planned checkpoint survives beside derived releases it collapses
into the release (Kubernetes' "milestone" is the release version, and the end
condition sits on the record as graduation criteria, the role acceptance
criteria play here); and Now/Next/Later, offered as a replacement, is already
present under the names `planned/` and `drafts/` plus priority, so adding it
as a stored unit would be a second name for one concept.

## Decision

We retire the phase and the milestone as units of the record, and the word
roadmap with them.

1. **Sequencing is dependencies plus the shelves.** `blocked_by` and
   `builds_on` on the records (itd-78 derives a priority from them; the pick
   in `abcd build next` filters on them) and the lifecycle shelves
   `drafts/ → planned/ → shipped/` carry the order. No stored unit sits above
   the intent for sequence.
2. **Now / Next / Later is a rendered status, never stored.** The bare `abcd`
   board and the site's Status page compute it: Now is every intent in a
   lane (from the build's state file) plus the head of the pick order; Next
   is every planned intent the gate reports READY; Later is the rest of
   planned and the drafts. A started state comes from the state file and is
   rendered (itd-2609212103568351). Nothing is called a roadmap.
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

- **Keep phases as the sequencing document, retire milestones only.** Rejected:
  the document had already drifted to editorial membership with no anchor,
  the failure mode the research pass names for every hand-maintained
  sequencing unit; a unit no verb reads is a unit nobody keeps current.
- **Rename the folders to now/next/later/done.** Rejected: the folders are
  a lifecycle the gates read (`plan` mints the spec on entering `planned/`),
  and the roadmap words are a position that changes without a decision;
  conflating them would either make Now lie or move the spec mint to lane
  start.
- **Now/Next/Later as a stored bucket on each record.** Rejected: a second
  name for `planned/`, `drafts/` and priority; the rendered form carries the
  same information and cannot drift.
- **A task record family below the spec.** Rejected in favour of a section:
  a fourth family with folders and verbs adds the kind of vocabulary this
  decision trims; Shape Up's scopes and Kubernetes' in-record graduation
  criteria are the precedents for keeping the pieces inside the record.
- **Keep the word roadmap for the rendered view.** Rejected on the
  repository's own documentation rule: a roadmap promises, this view
  reports; the docs are present tense, and the status board is where a
  reader already looks.

## Consequences

- adr-9 is superseded; the brief's mental-model chapter reads brief → intent
  → spec (→ steps), with the bundle as a delivery grouping and the derived
  release as the checkpoint (itd-2609211913453478 makes the edit).
- itd-24 becomes release retrospectives: the release boundary is real and
  the changelog is a ready seed; itd-34 drops the rule that bundle members
  share a phase.
- The run file's batches stay internal; its pull rule is priority plus
  dependency-readiness plus the ceiling.
- The `target_release:` field, the `## Steps` section, the status block and
  the glossary page are four intents planned on 2026-09-21 and built by the
  run; until they ship the folders are the only sequencing signal, which is
  the state the record was in already.
- What is lost: the phase's `## Expectation`, a working-backwards paragraph
  at release granularity. Its nearest home is a release press release
  composed from the shipped intents' press releases; the changelog composer
  is one step from it, and that step is not ruled here.
