---
id: itd-2609231013154443
slug: every-release-arrives-with-its-own-press-release-when-a
spec_id: spc-2609231435545473
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-67, itd-73]
severity: minor
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-24, itd-2609212103572513]
related_adrs: [adr-2609212115255771]
impact: additive
---

# Every release arrives with a press release composed from what shipped

## Press Release

> Every feature release arrives with its own press release. When a release is cut, abcd writes a short announcement to `RELEASE.md` at the top of the repository. The headline features are told as the moment a person notices them, the rest are listed by name, and each is in the words its own press release already uses, quotes included. The previous page moves to the release archive, so the top of the repository always shows the latest feature release and the history is one folder away. It looks back only: nothing still planned, no dates, no "coming next". A release of fixes alone leaves the page as it is and says why. The changelog stays the line-by-line record; `RELEASE.md` is the page a person reads first to learn what a release was for.
>
> "I used to work out what a release was for by reading forty changelog lines," said Iris, a product thinker. "Now the first page I open tells me, in the words I used when I asked for each piece."

## Why This Matters

A release is the checkpoint of this record (adr-2609212115255771, decision
3): the version is derived from what shipped and the changelog is composed
from the records that reached a terminal folder. What the checkpoint lacks is
the one paragraph a person reads to learn what the release was for. The phase
carried that paragraph as its `## Expectation`, written before the work;
retiring the phase removed it, and the ADR's last consequence names a release
press release composed from the shipped intents' press releases as its nearest
home and leaves the step unruled. The product thinker ruled it on 2026-09-23
(DECISIONS.md, the 2026-09-23 rulings entry; the interview's fuller wording is
recorded in this intent's Decisions below): each release gets a press release
composed at the cut from its shipped intents' press releases, backward-looking,
and this release is the first to carry one.

The material already exists. Every shipped intent opens with a press release
written as the user moment, and the changelog composer (itd-67) already reads
those records at the cut to word its lines. The changelog answers "what changed,
line by line"; nothing answers "what is this release for" in the words the
product thinker used when each piece was asked for.

An intent's press release is written before its work, so a page composed from
it is only as true as the deliveries behind it. This intent does not claim
otherwise: its Mechanism states the expectation and what would show it wrong.

## Decomposition (itd-84 hand-run, 2026-09-23, confirmed by the product thinker at the planning interview)

Verdict: **SPLIT**. The website half is a second capability and is filed as its
own draft; the rest is this intent, already-recorded stance, or plumbing.

| Part | Type | Home |
|------|------|------|
| `RELEASE.md` at the repository root, composed at the cut from the shipped intents' press releases, with the outgoing page moved to an archive | capability | this intent |
| Release pages on the project website, rendered from `RELEASE.md` and the archive | capability | a separate draft intent, next cycle (planning ruling P7) |
| The release record reports only what shipped: no forecast, no promised date, no still-planned intent | standing stance | **already recorded**: adr-2609212115255771 (decision 3; "a roadmap promises, this view reports"). This intent follows it and declares nothing new |
| Every cited record is data the binary checks, and a mismatch is refused before anything is written | trust rule | **already recorded** for the changelog as spc-11's completeness bijection (itd-67); applied here to a second document, so no new ADR (confirmed by the product thinker) |
| The composer prompt, the ingest step in `launch ship`, the archive move, and their rollback with the changelog | plumbing | the spec `intent plan` mints; the launch surface is `brief/04-surfaces/04-launch.md` |

Relations, written as prose because the schema carries no typed field for them
([iss-2609091256264547](../../../work/issues/open/iss-2609091256264547-three-of-the-four-mandated-typed-relations-cannot-be-written.md)):

- refines adr-2609212115255771: this intent is the step the ADR's last
  consequence names and leaves unruled.
- refines itd-67: it adds a second composed document to the cut that itd-67's
  changelog slice built, read from the same record set.
- builds on itd-73: the cut's record set and the `impact` derivation are
  itd-73's, and the press release reads that set rather than deriving its own.
- itd-24 (release retrospectives) and itd-2609212103572513 (`target_release`)
  are neighbours: the first may read this page as a seed, and the second is the
  forward-looking line this page must never carry.

## What's In Scope

- `RELEASE.md` at the repository root, holding only the latest feature
  release's press release, with a heading that names that release's version.
- At each cut that writes a new page, the outgoing `RELEASE.md` moves to
  `.abcd/development/releases/<its version>.md`, the permanent-record tier.
- The page is composed at the cut by a composer agent from the press releases
  of the user-facing intents that entered `shipped/` since the previous
  release tag: the same record set the changelog is composed from, never a
  second derivation of it.
- Headline intents told as prose, every other intent in the set listed by name;
  the persona quotes of the intents it tells, carried word for word with their
  attribution.
- The binary validates the composer's output before anything is written, and a
  refused output is sent back to the composer with the reasons until it is
  valid.
- The page, the archive move and the dated changelog section land in one cut,
  or none of them does.
- The read-only preview (`abcd changelog`, or the `launch` dry run) listing the
  intents the page will be composed from.
- `RELEASE` added to the `stray_root_docs` allowlist and to CI's inert root list.
- This release: the next cut carries the first one.

## What's Out of Scope

- Any forecast: no `target_release` intents, no still-planned work, no date,
  no "coming next". The `target_release` report (itd-2609212103572513) stays
  in the cut's report and never enters the page.
- Issues as a source: an issue carries no press release; fixes stay in the
  changelog's **Fixed** lines. A release that ships fixes alone writes no page.
- Intents with `impact: internal`, and intents that left the record as Removed
  (superseded): neither is announced.
- Release pages on the project website: a separate draft intent (see
  Decomposition).
- The published GitHub release notes, which stay as they are.
- Pages for releases cut before this one: no backfill and no on-request
  composition.
- Rewording the changelog, or changing its bijection, sections or version
  derivation (itd-67, itd-73).
- The release retrospective (itd-24), which asks what was learned; this
  page says what shipped.
- A press release for the project as a whole: that is the lifeboat's
  `disembark press-release`, a different document from different sources.

## Acceptance Criteria

- **Given** a cut whose record set contains at least one user-facing shipped
  intent, **when** `launch ship` writes the dated changelog section, **then**
  it first validates the composed page, then moves the outgoing `RELEASE.md`
  to `.abcd/development/releases/<its version>.md`, writes the new
  `RELEASE.md`, and writes the changelog heading; if any step fails, the steps
  already taken are rolled back (on a first cut, the new file is removed), and
  the report says so, or says the rollback failed.
- **Given** the page for a cut, **when** it is compared with the cut's
  press-release set (the intents that entered `shipped/` since the previous
  tag, excluding `impact: internal` intents, Removed intents and issues),
  **then** every intent in the set is cited, as a headline or in the name
  list, no record outside the set is cited, and the cut refuses otherwise.
- **Given** an intent that is planned but not shipped, including one carrying
  `target_release`, **when** the page is composed, **then** the page does not
  cite it; and the composer's fixture for a cut beside such an intent shows no
  sentence promising a date or a future release.
- **Given** the read-only preview, **when** it is run before a cut, **then** it
  lists the intents the page will be composed from, and writes nothing.
- **Given** a cut whose press-release set is empty, **when** `launch ship`
  runs, **then** `RELEASE.md` and the archive are left untouched and the report
  says that no page was written because no user-facing intent shipped.
- **Given** composer output with an unknown field, an oversize body, a
  malformed id, an id outside the press-release set, or a heading or fence that
  would break the page, **when** the cut ingests it, **then** it refuses the
  output whole, writes nothing, and sends the output back to the composer with
  the refusal's reasons to rewrite; it repeats until the output is valid, and
  the report names every refused attempt and its reasons.
- **Given** a quote on the page, **when** the cut checks it, **then** it
  matches word for word, with its attribution, a quote in the press release of
  an intent the page cites, and the cut refuses it otherwise.
- **Given** `RELEASE.md`, **when** anyone reads it, **then** its heading names
  the release it describes, so a page left in place by a fixes-only release is
  not mistaken for the latest tag's; and the composer's injection-canary fixture
  shows hostile text inside an intent's press release treated as content, never
  as an instruction.

## Decisions

Ruled by the product thinker on 2026-09-23 at the planning interview, after two
independent adversarial reviews of the draft (design/feasibility and record
discipline):

1. **Truthfulness.** No "true by construction" claim; the Mechanism records a
   falsifiable expectation instead, with no audit-verdict source and no
   owed-check label.
2. **Who writes.** A composer agent, as for the changelog lines, checked by the
   binary before anything is written. The human review is the release pull
   request's review of the roll commit; there is no separate approval step.
3. **Completeness.** Headline intents are told and the rest are listed by name;
   every intent in the press-release set appears, and the cut refuses an
   omission.
4. **Where.** `RELEASE.md` at the repository root holds the latest page only;
   at each cut the outgoing page moves to `.abcd/development/releases/`. This was
   the product thinker's own shape, beside the draft's three options.
5. **A fixes-only release.** No new page; nothing moves; the report says why.
6. **Quotes.** Carried word for word from the intents' press releases, with
   their attribution, knowing the personas are fictional and the page is public.
7. **Where else it is read.** The project website as well, filed as its own
   intent for next cycle; this intent is repository-only and the release is not
   held for the website.
8. **Earlier releases.** From this release onward only.
9. **Refused output.** Rewritten automatically, with no retry limit.

## Open Questions

_None open; decisions 1 to 9 settle the seven this draft carried and the two
the reviews raised._

## Mechanism

We expect the release press release to stay true because it is composed after the cut from records that passed the completeness bijection; shown wrong the first time it announces something an intent's audit found diverged or missing.

## Scope Conditions

- Releases are cut through abcd's own release step (`launch ship`), not tagged by hand. <!-- cond: cond-2609231435540975 -->
- Each shipped intent carries a press release written as the user moment, not a placeholder. <!-- cond: cond-2609231435545969 -->
- A repository has one release line, with no parallel release branches. <!-- cond: cond-2609231435546492 -->
- A release ships few enough intents, about twenty at most, that the headlines and a name list fit one readable page. <!-- cond: cond-2609231435540452 -->

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect it to fill the gap left by retiring phases, a paragraph of purpose per release, without anyone writing it by hand; shown wrong if releases still need a hand-written summary.
