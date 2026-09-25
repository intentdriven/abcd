---
id: itd-146
related_intents: [itd-2609212130136102, itd-2609212113220149, itd-134]
related_adrs: [adr-2609212115255771]
slug: abcd-s-help-renders-in-labelled-command-groups-and-the-group
spec_id: spc-2609212139586554
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# abcd's help renders in labelled command groups, and the grouping is gated like every other surface claim

> **Widened on 2026-09-21** by the product thinker: the grouped list is the person's, one line above it says `--agent` expands it, and `--help --agent` renders two blocks, people then agents and hosts. The scope invariant below ("no verb hidden") stands for execution: every verb runs the same whichever block lists it. The verb consolidation is its own record (itd-2609212130136102), and the one-sentence explainer per verb another (itd-2609212113220149).


## Press Release

> **abcd's command list reads as a map instead of an alphabet.** `abcd --help`
> renders its verbs under labelled groups: set-up, records, conformance checks,
> repository portability, release. A reader looking for the verb that fits the
> job scans five headings rather than twenty lines in alphabetical order. Every
> invocation is unchanged. No verb is renamed, moved, nested, or hidden, and the
> release derives as additive because the surface diff sees nothing to report.
>
> The grouping is also gated, which is the half that makes it last. The
> committed command-tree snapshot records each visible verb's group, a test
> asserts the rendered help actually carries the headings, and a second test
> fails when a newly registered visible verb declares no group.
>
> "I inherited the repo and typed `abcd --help` on day one," said Henry, a new
> hire. "I could read every line, so nothing was broken. What I could not do was
> tell which three of them I needed that morning, because alphabetical order
> puts `ahoy` next to `banlist` and tells you nothing about either."
>
> "The part I care about is that it cannot rot," said Kira, who maintains the
> surface. "If I add a verb and forget its group, the test tells me. If a group
> changes, the snapshot diff shows it. That is the same bar every other claim
> about this surface is already held to."

## Why This Matters

`abcd --help` lists twenty-two lines: twenty visible verbs, plus `help` and
`completion`, which cobra generates. A twenty-first verb, `hook`, is registered
and hidden. Five further verbs are reserved but unbuilt. The list is legible
today and there is no demonstrable legibility failure to point at, which the
research recorded in
[`2026-08-22-ideate-cli-verb-taxonomy-restructure.md`](../../research/notes/2026-08-22-ideate-cli-verb-taxonomy-restructure.md)
tested and confirmed. What alphabetical order costs is not readability but
orientation: it carries no information about what any verb is for.

That same research killed the obvious alternative. Regrouping into a noun-verb
hierarchy fails here for reasons specific to this repository: abcd already
nests under the object wherever the semantic-ambiguity trigger fires
(`lint`, `docs lint`, `memory lint`; `intent audit`; `banlist add`), sixteen of
the twenty-one registered verbs already own sub-commands across seventy-three
command paths, and the pre-1.0 no-alias rule in adr-40 forecloses the migration
path the external precedents relied on.

What is left is a rendering problem, and cobra answers rendering problems
directly. This intent takes that answer and attaches the detectors the
adversarial review found missing from it.

## What's In Scope

- **Command groups on the root:** `AddGroup` on the root and a `GroupID` on
  every visible top-level verb, plus `SetHelpCommandGroupID` and
  `SetCompletionCommandGroupID` so cobra's two generated commands are filed
  rather than falling into the "Additional Commands" bucket.
- **No verb hidden, renamed, moved, or nested.** `rules` and `spec` stay
  listed. They are operator-facing, and the `operator_internal` key in
  `record-lint.json` means *needs no surface chapter*, which is a lint
  exemption rather than a user-visibility verdict. Hiding them would make
  `--help` contradict the `ahoy`-installed marker block, which tells every
  managed repository to run `abcd rules`.
- **The snapshot records the group:** A group field on the surface snapshot's
  `Command`, populated for visible top-level verbs and empty for sub-commands,
  so a regroup is visible in the committed command tree and its drift test.
- **A test asserts the rendered help.** Nothing asserts root help text today,
  so the render is currently unguarded in both directions.
- **A test fails on a visible top-level verb with no group.** Hidden commands
  are exempt: they never render, so a group would be a claim about nothing.

## What's Out of Scope

- Any change to a verb's name, position, or invocation.
- Hiding any currently visible verb.
- Folding `changelog` under `launch`.
- **Adding a kind to the surface break taxonomy.** The taxonomy is closed by
  design, and `surface.Diff` returns only breaks: the release guardrail fails
  any cut whose diff is non-empty unless a record declares `impact: breaking`.
  A "non-breaking regroup kind" is therefore a contradiction, because adding
  the kind is what would make a regroup breaking. A regroup is invisible to the
  diff, and that is the correct behaviour. The snapshot field, not the
  taxonomy, is where a regroup becomes visible.

## Mechanism

We expect grouped help to shorten the time a reader takes to find the right
verb **because the grouping carries information alphabetical order does not**:
the reader discards four groups wholesale rather than reading twenty summaries.
This is falsifiable. If the groups are drawn badly enough that a reader reads
every entry anyway, the change costs a screen of vertical space and returns
nothing, and the honest response is to say so rather than redraw them
indefinitely.

We expect the detectors to matter more than the grouping **because a detector
blind at the grain of a claim is blind at that grain, however sound it is
elsewhere**. `iss-246` is the recorded instance, in its corrected form: the
`surface_coverage` rule existed and passed while sub-verb-level claims in the
record drifted, because the rule could not see inside a surface row. The fix
extended the detector to that grain and corrected the three documents. A
snapshot that records no group is blind to groups in exactly the same way, and
a grouping shipped without one would repeat the defect it was chosen over.

## Scope Conditions

- Holds for a **CLI whose top level is a set of acts rather than resources**. <!-- cond: cond-2609212139580802 -->
  Should abcd grow a genuine resource with several verbs that no existing verb
  owns, the noun-verb question reopens as a real one rather than a tidiness one.
- Holds for **cobra**. The grouping mechanism is a cobra feature, so a change of <!-- cond: cond-2609212139580635 -->
  command framework re-decides this.
- Holds while **no third-party author registers verbs**. An extension ecosystem <!-- cond: cond-2609212139582531 -->
  would break the ungrouped-verb test, because the core cannot assign a group to
  a verb it does not register.
- The **group titles and membership are a presentation choice**, not a taxonomy <!-- cond: cond-2609212139588587 -->
  claim. They carry no adr-40 bucket meaning and must not be read as one.

## SOTA

**The alternative:** Presentational command groups in the CLI framework, as
kubectl renders them (beginner and intermediate sections) and Terraform does
(main commands against all other commands). cobra implements this natively
through `AddGroup` and `GroupID`.

**Maturity:** Mature, and de-facto standard for cobra CLIs of this size.

**Path: adopt the SOTA alternative (path 1).** Path 1 is normally a hard stop
for maintainer approval, because adoption means a new dependency. Here it adds
none: the repository already pins cobra v1.10.2, which carries the feature, so
the gate has nothing to weigh. The three native additions (the snapshot field
and the two tests) are a small complement cobra does not offer rather than a
bespoke build. Whether that reading of the path is right is an open question
below rather than a decision this intent takes for the maintainer.

Rejected alternatives, with reasons recorded in the ideate verdict
[`2026-08-22-ideate-cli-verb-taxonomy-restructure.md`](../../research/notes/2026-08-22-ideate-cli-verb-taxonomy-restructure.md):
a noun-verb restructure with category nouns; hiding operator-facing verbs;
Heroku-style colon topics; an extension-verb growth valve.

## Acceptance Criteria

- **Given** `abcd --help`, **when** it renders, **then** the person's verbs appear under labelled groups (set-up, records, checks, portability, release), with one line above them saying `--agent` expands the list with the verbs agents and hosts call.
- **Given** `abcd --help --agent`, **when** it renders, **then** two blocks appear, the person's groups then an agents-and-hosts block, and every visible verb is in exactly one block.
- **Given** any verb, **when** it runs, **then** it runs the same whichever block lists it; a visible top-level verb registered with no group or block fails a test.
- **Given** the surface snapshot, **when** a verb's group or block changes without regeneration, **then** the release gate fails naming the verb.
- **Given** a verb's command page, **when** it is read, **then** it says which block the verb is in, and each line of the agent block names the page an agent reads next.
- **Given** `abcd rules` and `abcd spec`, **when** `--help` renders, **then** they keep their places; nothing is renamed or nested.

## Decisions

Ruled by the product thinker on 2026-09-21:

1. **Two blocks behind one flag**: the default list is the person's groups with the expanding line above; `--agent` shows both blocks. Nothing is hidden from execution.
2. **Group titles**: set-up (`ahoy`, `update`), records (`capture`, `intent`, `spec`, `decide`, `build`, `drain`, `memory`), checks (`lint`), portability (`embark`, `disembark`), release (`launch`); the agent block holds `implement`, `reading`, `history`, `statusline`, `changelog`, `guard hook`, `intent audit ingest`, `ideate record`, `mode`.
3. **The snapshot's schema version bumps** for the block field; `changelog` sits in the agent block.

## Open Questions

_None open; decisions 2 and 3 settle the four this record carried._

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-cdad1c08a756 -->
Fidelity review OWED (receipt rcp-cdad1c08a756).

## Grounds

- pursued: the run lands build, drain, implement, reflect and reclassify, and an alphabet of forty verbs is where a newcomer gives up; we expect a person to find their verb in the grouped list and an agent to find the second block from the line above it; shown wrong if the first agent transcripts after it ships still grep the binary for verbs
