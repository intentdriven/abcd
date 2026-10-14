---
id: itd-2610141646456128
slug: a-private-ideate-verdict-is-written-by
spec_id: null
kind: bundle-member
bundle: ideate-librarian
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610141646454347]
related_issues: [iss-2610090740139804, iss-2610090748299348, iss-2610090742488191, iss-2610141647309012]
related_adrs: [adr-2610142116480836, adr-2610141646525117]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A private ideate verdict stays private

## Press Release

> **A person who keeps an idea private records its verdict without anything
> reaching the project.** Every ideate verdict is recorded either privately or
> publicly, and abcd asks which; it never guesses. A private verdict goes to
> the person's own store, readable by them alone, with no line in the project's
> decision log. The grill reads that store too, so an earlier private verdict on
> the same idea is found and cited. An idea's source is named in public only
> once the idea graduates to a draft intent, with its credit added then.
>
> "I test half-formed ideas from things I read," said Alice, a solo founder.
> "The killed ones and their sources stay on my machine. The one I adopted went
> public with its source credited, in the same change."

## Why This Matters

The product thinker ruled on 2026-10-14 that each ideate run asks whether the
idea is private or public, private verdicts going to
`~/.abcd.noindex/ideas/<root-sha>/` with no decision-log line. The binary can
only write into the repository, so every private run records in a throwaway
worktree and copies the note out by hand (iss-2610090740139804); the lab of
2026-10-14 found that hand copy left the notes readable by others until fixed.
The grill's reading list omits that store, and a private verdict cannot be cited
(iss-2610141647309012). The two source rules wait on the same change: a public
source named only on acceptance (iss-2610090748299348), and its credit added on
adoption (iss-2610090742488191).

## What's In Scope

- `ideate record --private` and `--public`; with neither, a question at a
  terminal and a refusal elsewhere.
- The private store, its file mode, its listing, and citing a private verdict
  from another private verdict.
- The rule that names an idea's source publicly, and credits it, at graduation.

## What's Out of Scope

- The verdict schema itself (itd-2610141646454347).
- Verdicts on ideas from a private reading pack, which are always private
  (itd-2610141646452213).

## Mechanism

We expect no private verdict, and no source behind one, to reach the project,
because the binary itself picks the store, the file mode and whether a
decision-log line is written, and refuses to pick when the person has not said.
Shown wrong if a verdict recorded with `--private` appears in a working tree or
in the decision log.

## Scope Conditions

- One person's machine; the private store is keyed on the repository's root
  commit, so the verb still needs a repository to run in.
- An idea counts as accepted when it graduates to a draft intent; that is the
  moment its public source is named (the precedent: two draft intents of
  2026-10-14 graduated this way, with their source credited).
- A source held in the private sources corpus is named publicly only after a
  person's ledger flip, as adr-41 requires.

## Acceptance Criteria

- **Given** `--private`, **when** a verdict is recorded, **then** it is written
  under `~/.abcd.noindex/ideas/<root-sha>/` with owner-only permissions, it
  passes the same redaction as a public verdict, no decision-log line is
  appended, and the working tree is unchanged.
- **Given** neither `--private` nor `--public`, **when** `ideate record` runs at
  a terminal, **then** it asks which; **when** it runs without one, **then** it
  refuses with exit 2 and writes nothing.
- **Given** a private verdict, **when** another private verdict's grill cites it
  by its key, **then** the citation resolves; **when** a public verdict cites
  it, **then** the public verdict is refused.
- **Given** a private verdict recorded on the same day under the same slug,
  **when** a second is recorded, **then** it is refused, as in the public store.
- **Given** a public verdict for an idea that is killed, **when** it is
  recorded, **then** its origin sources (the sources the idea came from) are
  left out of the committed record, while each claim's primary source stays.
- **Given** a draft intent graduated from a private verdict, **when** the change
  that adds it is checked, **then** it names the idea's public origin source and
  adds its credit to ACKNOWLEDGEMENTS.md, or the check refuses it.

## Prior Art

- iss-2610090740139804, iss-2610090748299348, iss-2610090742488191 (open): the
  three records this intent delivers.
- adr-41 (proposed): the sources corpus's confidential and public classes and
  the person's ledger flip, which the naming rule follows.
- library add: the same refuse-unless-told rule for a source's class.

## Open Questions

_None._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
