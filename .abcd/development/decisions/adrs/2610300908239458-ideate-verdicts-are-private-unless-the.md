---
id: adr-2610300908239458
slug: ideate-verdicts-are-private-unless-the
status: accepted
date: 2026-10-30
supersedes: adr-2610142116480836
superseded_by: null
related_intents: [itd-2610141646456128, itd-2610141646452213, itd-2610141646454347]
related_rfcs: []
related_adrs: [adr-41, adr-2610141646525117, adr-2610142116480836]
---

# ADR-2610300908239458: Ideate verdicts are private unless the person says public

## Context

[ADR-2610142116480836](2610142116480836-ideate-verdicts-are-private-or-public-by.md)
made every ideate verdict private or public by the person's choice, never by
default: each run asked, and `ideate record` refused a run that named neither
store. It rejected private-by-default on the ground that it contradicted the
ruling that each run asks. On 2026-10-30 the product thinker withdrew that
ruling: ideate is private by default and does not ask
(iss-2610300838089246).

## Decision

- **A verdict is private unless the person says it is public.** `ideate
  record` with no flag, or with `--private`, writes the private store;
  `--public` writes the public one and is passed only on the person's word,
  never on an inference from the idea or the repository. Nobody is asked.
  Both flags together are refused, writing nothing.
- **A private verdict stays on the person's machine:** under
  `~/.abcd.noindex/ideas/<root-sha>/`, owner-only, through the same redaction as
  a public one, with no decision-log line. It may be cited by another private
  verdict, never by a public one.
- **A verdict whose idea came from a private reading pack is always private.**
- **An idea's origin source is named publicly only when the idea graduates to a
  draft intent**, and the graduating change adds its credit to
  ACKNOWLEDGEMENTS.md. A public verdict on a killed idea leaves its origin
  sources out of the committed record; each claim's primary source stays. A
  source held in the private sources corpus is named publicly only after a
  person's ledger flip (adr-41).

The last three decisions are carried unchanged from ADR-2610142116480836, so
this record is their one home.

## Alternatives Considered

- **Private by default, public on the person's word:** a forgotten flag leaves
  an idea private, which costs nothing and is undone by recording it again
  with `--public`. Chosen.
- **Ask on every run, no default (ADR-2610142116480836):** never guesses, but
  puts a question to the person before every run. Superseded by the ruling of
  2026-10-30.
- **Public by default, private on request:** a forgotten flag commits a private
  idea, and a pushed commit cannot be taken back. Rejected.
- **Name sources once an intent is planned, not drafted:** later and safer, but
  the two drafts that graduated on 2026-10-14 already named theirs at the draft.
  Rejected, as in ADR-2610142116480836.

## Consequences

- An unattended run records privately without needing to say so; it says
  `--public` only when the person has.
- A verdict the person meant to be public stays private until it is recorded
  again with `--public`; nothing moves it on its own.
- A private verdict is invisible to teammates and to the repository's gates; its
  only reader is the person and their own later ideate runs.
- The rule binds itd-2610141646456128 (the store and the naming rule) and
  itd-2610141646452213 (verdicts from private packs).
