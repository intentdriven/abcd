---
id: itd-2610171228577260
slug: abcd-has-no-established-procedure-for
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610170713227840, iss-2610180913376757]
origin: extracted-from-record
production_mode: hand-written
---

# A product thinker settles a visual design in rounds that abcd runs end to end

## Press Release

> A product thinker settles a visual design in rounds that abcd runs end to end. Alice, a solo founder who is the product thinker, wants to see what the phone screen for answering their agents could look like. They ask once; abcd opens a design round for the question, writes the brief from the record, and shows five finished-looking designs side by side, on phone and tablet, light and dark, with one screens bar that switches every design at once. Alice tags each with a first impression, compares them in pairs, and says what they like in their own words. From that feedback abcd builds the next round on its own, without being asked again: a synthesis of what won, the faults Alice named fixed, each screen checked before it is shown. When Alice adopts a design, it becomes the visual reference for the build. Nothing is downloaded or copied by hand: every judgement is saved into the project's record as it is made, first from Alice's own computer and later on abcdesign.app, where a colleague can be invited to rank too.

## Why This Matters

_Written by the agent on 2026-10-17 from the issue it was promoted from and the
product thinker's words that day; the planning interview confirms or rewrites
it._

abcd has no procedure for design work, so each visual design question is run
from memory. Three times the same process has been rebuilt by hand: the
control-panel lab in another managed repository, the brief session page for
abcdesign.app (lab-261016142621-a46cdaa, three rounds), and the phone and
tablet answer screen (lab-261017071130-6fd3032, four rounds). Each time the
brief, the render script, the gallery and the comparison tool were copied from
the previous lab, and every result had to be downloaded and handed back.

The product thinker's rulings on 2026-10-17, restated:

- Design work becomes an established abcd procedure (iss-2610170713227840).
- Once the product thinker has given their feedback on a round, the next round
  is built autonomously: abcd synthesises what won and fixes what they named,
  and puts the result in front of them; it never waits to be told to continue.
- They do not want to download results: judgements land in the record as they
  are made.
- abcdesign.app becomes the place this happens, after the brief and before
  intents (recorded with the abcdesign.app session kinds on the model-matrix
  plan's branch; linked once it lands).

What the answer-screen rounds taught the tool, each now a requirement: one
shared screens bar for every design, with the device and theme taken from the
page's own controls; the "look closer" view filling the window with the page
behind it washed out and still; tags on top and the letter and magnifier inside
the design's frame; the left/right choice at the top, a reason asked in an
overlay after the choice and never for "too close"; every new pair starting
from the first screen; keyboard hints behind a switch, and keys 1 to 9 picking
the screen. The pairwise step discriminated weakly between close variations
(one of three repeats agreed in round 3), which is a reason to keep it for
choosing between distinct directions and to let the person skip it.

Two more rulings from the product thinker the same day, restated:

- **When each comparison is used.** Side-by-side comparison is for the round
  where the designs differ obviously, the taste-based choice between
  directions. After that, the round makes several variations of one specific
  element (navigation, say) rather than of the whole design, so the person
  compares like with like.
- **Best practice is recorded over time, so abcd ships decent defaults.**
  Every round records what the person kept, changed or rejected, and across
  rounds, projects and people abcd learns which choices change least: those
  become the defaults new design rounds start from (the answer-screen rounds
  already show some: one shared screens bar, the iPhone in portrait and the
  iPad in landscape, a floating palette that never covers navigation, a reason
  asked only after a choice). What keeps changing stays a question; what
  settles becomes standard.

The practical experience of both labs (the procedure as it ran, the brief's
hard rules, the tool's behaviours as requirements for abcdesign.app's design
stage, the candidate defaults, the faults and the measures) is recorded in
`.abcd/development/research/notes/2026-10-17-design-rounds-practice.md`.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
