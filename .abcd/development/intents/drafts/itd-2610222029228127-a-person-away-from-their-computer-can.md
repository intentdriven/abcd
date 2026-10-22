---
id: itd-2610222029228127
slug: a-person-away-from-their-computer-can
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610170638184486, itd-2610170634394993]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Every session's checklist reaches a shared copy and the phone

## Press Release

> A person away from their computer can follow every abcd session's checklist and answer the questions that would stop its work from their phone. Each session's full checklist is copied beyond the computer it runs on: to a private shared copy, everything by default and narrowable, and, if the person turns it on, to a public one that never carries an unpublished advisory. An installable web app on the private shared copy notifies the person whose role a question belongs to only when a session's work would otherwise stop, and lets them choose among the options offered or add a note.

## Why This Matters

_Split out of the checklist draft (itd-2610170638184486) on 2026-10-22,
recorded on the product thinker's ruling that full checklist uploads come
later (dcn-2610222025114805); the text below is carried over from
that draft's decisions of 2026-10-17, not newly confirmed._

The checklist draft keeps each session's state on the computer it runs on,
so a session's work survives compaction and a restart there. From there it
cannot let the person see and answer from somewhere else. On 2026-10-17 the
product thinker asked for that: a private shared copy (for example
through abcdev.app) that shares everything by default, a public option on
GitHub, and the phone, so a question that would stop a session overnight
reaches the right person wherever they are. On 2026-10-22 they ruled that
full uploads, public and private, come in a later piece of work, after the
short status line the coordination intent publishes; this draft is that
later piece.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

_Carried over unchanged from the checklist draft's agent-derived criteria of
2026-10-18; none of it is confirmed._

- **Given** the private shared option is on, **when** a session writes its
  checklist, **then** the checklist is shared in full by default, and a person
  can narrow what is shared.
- **Given** the public option is on, **when** it publishes a checklist,
  **then** nothing carrying the id of an advisory not yet published leaves the
  machine, by the same check announcements apply.
- **Given** the phone app on the private shared option, **when** a session's
  work would otherwise stop on a question, **then** the person whose role it
  belongs to (product questions to Alice, technical trade-offs to Nia) is
  notified, and can only choose among the options offered or add a note.

## Decisions

Carried over from the checklist draft, where the product thinker gave them
on 2026-10-17, restated:

- **Where the copies go:** a private shared option (for example through
  abcdev.app) and a published option on GitHub, both beyond this computer
  (product thinker).
- **What the private shared option shares:** everything by default,
  configurable later (product thinker).
- **The phone surface:** an installable web app on the private shared
  option, rather than a native app, reusing the live brief session's relay
  and GitHub sign-in; a native app only if lock-screen answers or widgets
  prove necessary. An answer from the phone only chooses among the options
  offered or adds a note. Its look is chosen in design rounds in
  lab-261017071130-6fd3032 (product thinker for scope; agent for the form).
- **Who answers from the phone:** each person by their role: product
  questions to the product thinker, technical trade-offs to the technical
  facilitator (product thinker).
- **When the phone notifies:** configurable; by default only when a
  session's work would otherwise stop (product thinker).
- **A public upload passes the embargo check (agent, on brief invariant
  22):** nothing published may carry the id of an advisory not yet
  published, the same check announcements apply (itd-2609151838312703).
- **A shared copy never carries a credential (on adr-2610222026265719,
  brief invariant 24):** the copies start from a checklist that names a
  credential only by kind, scope and expiry.
- **Split from the checklist draft with no question put (2026-10-22):**
  recorded as dcn-2610222025212338; the product thinker's
  answer that full uploads come later settled it.

## Open Questions

- Does another computer resume a dead session from its shared copy, and if
  so, what does it take over (the claims through the coordination intent's
  take-over, the checklist's items, both)?
- Is the private shared copy abcdev.app's store, or the person's own private
  repository on the forge?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
