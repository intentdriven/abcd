---
id: itd-2610240913343804
slug: before-abcd-starts-anything-the-product
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610031214560142]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# No surprises: abcd says up front what it will do on its own and what it will ask

## Press Release

> Before abcd starts anything, the product thinker and the technical facilitator can see which steps will run on their own and which will stop and wait for them, and why. Each flow (setup, planning, a build, a drain, a release) says up front, in plain words, what it will do automatically, what it will ask, and who will be asked; then it does exactly that. A step that turns out to need a person is asked as itself, at the moment it is needed, never discovered after the answers are in; and an agent working through abcd never stops to ask permission for a step the flow already said it would take.

## Why This Matters

_Drafted by the agent from the 2026-10-24 session; not yet confirmed by the product thinker._

On 2026-10-24 the product thinker said there are "too many surprises at the moment for the PT and TF", and that "it must be clear to them what will be automated". The same day, in one session:

- Setup asked every status-line question, then refused the status line afterwards for a prerequisite nobody had mentioned (iss-2610240823574047).
- A no to a generic "Change the settings listed above?" silently skipped the identity question the person wanted (iss-2610240824005655).
- Setup ran a tool install through Homebrew that could never succeed on the account, then passed on Homebrew's `sudo` advice (iss-2610240824571059).
- The agent stopped to ask "should I start the two reviews?" for a step the planning flow already requires, and the product thinker asked why.
- Two other sessions wrote records into the same working tree, unannounced, which put two generated indexes out of date for everyone.

Each was a step the person did not know was coming, or a question about a step they had already agreed to. A note in one agent's memory fixes none of it for the next person: the flow itself has to say what it will do.

The board already shows what waits on the product thinker and what comes next (itd-2610031214560142, which this builds on); this intent asks every flow to say the same thing before it starts, and to keep to it.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- Where does a flow's statement of what it will automate and what it will ask live: printed at the start of each flow, in the flow's command page, on the board, or all three? (raised 2026-10-24)
- Is "an agent never asks permission for a step its flow already announced" a principle for every abcd-managed repository, carried by the rules loader, rather than part of this intent? (raised 2026-10-24)
- Does this cover peer sessions too: should a session announce to the person, not only to its peers, the records it is about to write into a shared working tree? (raised 2026-10-24)

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
