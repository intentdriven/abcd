---
id: itd-2609231013154443
slug: every-release-arrives-with-its-own-press-release-when-a
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-67, itd-73]
severity: minor
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-24, itd-2609212103572513]
related_adrs: [adr-2609212115255771]
---

# Every release arrives with a press release composed from what shipped

## Press Release

> Every release arrives with its own press release. When a release is cut, abcd composes one short announcement from the press releases of the intents that shipped in it and writes it beside the changelog: what a person can do with this release that they could not do with the last one, told as the moment they notice it, in the words the intents already use. It looks back only. It names nothing still planned, promises no date and forecasts no next release, so it can never say more than the release holds. The changelog stays the line-by-line record; the press release is the page a person reads first to learn what the release was for.
>
> "A phase used to open with a paragraph saying what the world would look like when it ended, and nobody checked it afterwards," said a product thinker at a cut. "Now the paragraph comes at the end, from what actually shipped, and I did not have to write it."

## Why This Matters

A release is the checkpoint of this record (adr-2609212115255771, decision
3): the version is derived from what shipped and the changelog is composed
from the records that reached a terminal folder. What the checkpoint lacks is
the one paragraph a person reads to learn what the release was for. The phase
carried that paragraph as its `## Expectation`, written before the work and
never checked against it; retiring the phase removed it, and the ADR's last
consequence names this record's shape as its nearest home and leaves the step
unruled. The product thinker ruled it on 2026-09-23: each release gets a press
release composed at the cut from its shipped intents' press releases, written
beside `CHANGELOG.md`, backward-looking, with no forecast, and this release is
the first to carry one.

The material already exists. Every shipped intent opens with a press release
written as the user moment, and the changelog composer (itd-67) already reads
those records at the cut to word its lines. The changelog answers "what changed,
line by line"; nothing answers "what is this release for" in the words the
product thinker used when each piece was asked for. Composing it at the cut,
from what shipped rather than from what was hoped, keeps the paragraph true by
construction where the phase Expectation was true only by intention.

## Decomposition (itd-84 hand-run, 2026-09-23, advisory: not yet confirmed by the product thinker)

Verdict proposed: **FILE-AS-IS with flags**. One part is a capability; the rest
is already recorded or is plumbing the planning interview places.

| Part | Type | Home |
|------|------|------|
| A release press release composed at the cut from the shipped intents' press releases, written beside `CHANGELOG.md` | capability | this intent |
| The release record reports only what shipped: no forecast, no promised date, no still-planned intent | standing stance | **already recorded**: adr-2609212115255771 (decision 3 and the rejected "keep the word roadmap": a roadmap promises, this view reports) and the present-tense documentation rule. This intent respects it and declares nothing new |
| Every claim in a release record cites the record it reports, and the binary refuses a mismatch before writing | trust rule | **already recorded** for the changelog as spc-11's completeness bijection (itd-67). Whether it extends to this document is open question 4; extending it applies the existing rule to a second document and needs no new ADR, while a composer whose output the binary writes uncited would be the new trust rule, and would go to an ADR |
| The composer prompt, the ingest step in `launch ship`, the file's shape and its rollback with the changelog | plumbing | the brief's release internals and the spec `intent plan` mints |

Typed links:

- `refines` adr-2609212115255771: this intent is the step the ADR's last
  consequence names and leaves unruled (the successor to the phase
  `## Expectation`).
- `refines` itd-67: it adds a second composed document to the cut itd-67's
  changelog slice built, read from the same record set.
- No `supersedes`, `reverses` or `duplicates` link found. itd-24 (release
  retrospectives) and itd-2609212103572513 (`target_release`) are neighbours,
  not refinements: the first may read this document as a seed, and the second
  is the forward-looking line this document must never carry.

## What's In Scope

- A press release for one release, composed at the cut from the press
  releases of the intents that entered `shipped/` since the previous release
  tag: the same record set the changelog is composed from, never a second
  derivation of it.
- Writing it beside `CHANGELOG.md` in the same cut, so the changelog and the
  press release land in one commit or neither does.
- Backward-looking content only: what shipped, in the user's terms.
- The read-only preview (`abcd changelog` or the `launch` dry run) showing
  which intents the press release will be composed from.
- This release: the next cut carries the first one.

## What's Out of Scope

- Any forecast: no `target_release` intents, no still-planned work, no date,
  no "coming next". The `target_release` report (itd-2609212103572513) stays
  in the cut's report and never enters this document.
- Issues as a source: an issue carries no press release; fixes stay in the
  changelog's **Fixed** lines.
- Rewording the changelog, or changing its bijection, sections or version
  derivation (itd-67, itd-73).
- The release retrospective (itd-24), which asks what was learned; this
  document says what shipped.
- A press release for the project as a whole: that is the lifeboat's
  `disembark press-release`, a different document from different sources.

## Acceptance Criteria

> _Candidates for the planning interview to confirm, amend or drop. Each is
> written to hold under every construal of the open questions below; where one
> depends on a question it names it._

- **Given** a cut whose record set contains shipped intents, **when** `launch
  ship` writes the dated changelog section, **then** it writes the release
  press release beside `CHANGELOG.md` in the same write, and a failure of
  either leaves neither written.
- **Given** the release press release for a cut, **when** it is compared with
  the cut's record set, **then** every intent it draws on shipped in that cut,
  and it names no intent outside it (the completeness half is open question 4).
- **Given** an intent that is planned but not shipped, including one carrying
  `target_release`, **when** the press release is composed, **then** the
  document does not mention it, and no sentence in it promises a date or a
  future release.
- **Given** the read-only preview, **when** it is run before a cut, **then** it
  lists the intents the press release will be composed from, and writes
  nothing.
- **Given** a cut whose record set contains no shipped intent, **when** `launch
  ship` runs, **then** it does what open question 3 rules, and says so in its
  report.
- **Given** the release press release is untrusted composer output (if a
  composer is used, open question 2), **when** it carries text that reads as an
  instruction, a record id not in the cut, or markup that breaks the file's
  structure, **then** the cut refuses it whole and writes nothing.

## Open Questions

For the planning interview. Each lists at most three construals and the null
answer; the order carries no preference and nothing here is recommended.

1. **Where does it sit beside the changelog?** (a) One file at the repository
   root, newest release on top, like the changelog; (b) one file per release in
   a folder of its own; (c) a short section under each dated heading inside
   `CHANGELOG.md` itself; or none of these yet.
2. **Who writes the words?** (a) A composer agent, as the changelog lines are
   written, checked by the binary before anything is written; (b) no agent:
   abcd stitches together the opening sentence of each shipped intent's press
   release; (c) an agent drafts and the product thinker approves the text before
   the cut completes; or none of these yet.
3. **What happens in a release that shipped fixes but no intent?** (a) No press
   release, and the cut says why; (b) a one-sentence press release pointing at
   the changelog; (c) a short press release written from the fixes; or none of
   these yet.
4. **Must every shipped intent be in it?** (a) Yes, each one cited, and the cut
   refuses a press release that leaves one out, as it does for the changelog;
   (b) the headline few are told and the rest are listed by name; (c) the
   composer chooses, and nothing checks completeness; or none of these yet.
5. **Does it carry quotes?** (a) No quotes, prose only; (b) quotes carried word
   for word from the intents' own press releases; (c) one composed quote for
   the whole release; or none of these yet.
6. **Where else is it read?** (a) Only in the repository, as a file; (b) also as
   the published release's notes, in place of the forge's generated list of
   merged pull requests; (c) also on the site's release pages; or none of these
   yet.
7. **What about releases already cut?** (a) From this release onward only; (b)
   composed once for every earlier tag as well; (c) earlier releases get one on
   request; or none of these yet.

The Mechanism and Scope Conditions sections below are left as their seeded
prompts: both are the product thinker's claims to make at the interview.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
