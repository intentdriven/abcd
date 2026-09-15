---
term: roadmap
bounded_context: core
definition: The sequencing folder .abcd/development/roadmap/, which holds the phase docs and the RFCs. Its README is the roadmap dashboard, a separate sense — a live status render that reads the native spec store and the intent buckets rather than the phase docs.
aliases: ["roadmap folder"]
forbidden_synonyms: ["backlog", "timeline", "release plan"]
status: stable
introduced_in: adr-9
starts_when: null
ends_when: null
not_to_be_confused_with: core/phase
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# roadmap

The **roadmap** is [`.abcd/development/roadmap/`](../../../roadmap/README.md): two things and
no others — [`phases/`](../../../roadmap/phases/README.md), the ordered build plan, and
`rfcs/`, where a proposal is argued before it becomes an ADR. It carries no dates and no
estimates. What has shipped is defined by which phases are complete and which intents sit in
`shipped/`.

## Senses

| Sense | The one spelling | Where it lives |
|---|---|---|
| The folder: phase docs plus RFCs | **the roadmap** | [`development/roadmap/`](../../../roadmap/README.md) |
| The live status render at that folder's README | **the roadmap dashboard** | [`roadmap/README.md`](../../../roadmap/README.md) |

**The dashboard does not read the roadmap.** This is the trap the word sets. The phase docs are
editorial prose about sequence; the dashboard's per-spec truth is read live from the **native
spec store** and from the intent buckets on disk, because hand-maintained counts drift the
moment work ships. So "what the roadmap says" and "what the roadmap dashboard shows" answer
different questions from different sources, and only the second is current by construction.
Per-phase progress on the dashboard stays coarse and editorial for exactly that reason.

**A phase doc carries no status.** Status is never stored in a design doc
([adr-5](../../../decisions/adrs/0005-brief-is-current-state.md)), so the dashboard is the only
place a reader should look for state.

## When to use

Use "the roadmap" for the folder and its contents when the subject is *sequence* — what is
bundled into which phase, in what order, and why. Use "the roadmap dashboard" whenever the
subject is *current state*.

## When NOT to use

Do not use "roadmap" for the intent corpus: most intents are not phased, and an intent listed
in no phase doc's `## Scope` is unscheduled, which is a valid state
([adr-34](../../../decisions/adrs/0034-lifecycle-and-scheduling-orthogonal.md) makes scheduled
and planned orthogonal axes). Do not use it for a release timeline — a release version is
derived from what shipped, never an input that organises work.

## Examples

- "The roadmap holds phases and RFCs; an accepted RFC produces an ADR."
- "The roadmap dashboard reads the native spec store, so its spec counts are whatever the store reports right now."

## Related terms

- [phase](phase.md) — what the roadmap's `phases/` folder holds
- [plan](plan.md) — the phase docs as the ordered build plan
- [spec](spec.md) — the store the dashboard reads
- [record](record.md) — the durable tier the roadmap sits in
