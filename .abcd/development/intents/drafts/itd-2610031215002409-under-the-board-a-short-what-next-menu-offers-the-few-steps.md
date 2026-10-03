---
id: itd-2610031215002409
slug: under-the-board-a-short-what-next-menu-offers-the-few-steps
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610031214560142, itd-2610030810370060]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# A 'what next?' menu under the board offers the next step, in a Terminal and in a Claude Code session

## Press Release

> Under the board, a short 'what next?' menu offers the few steps that make sense now: start the next thing to build, see everything that is ready, open the view for the facilitator, or nothing for now. In a Terminal you choose with the arrow keys or by number, drawn the same way as abcd's questions; inside a Claude Code session the same choices arrive as a question. Asking you something marks the status as waiting on you, and answering clears it. Where nobody can answer, such as output going to a file or another program, the board prints with no menu.

## Why This Matters

A board says where things stand. What a product thinker usually wants next is one of a few moves, and today each needs a command they are not assumed to know (the product thinker's request of 2026-10-03, choosing "the board plus a menu"). Ending the board on the moves themselves turns it from a report into a starting point, and it gives the board's "expand to the view for the facilitator" step somewhere to live.

State of the art (2026-10-03, reports/sota-board-visuals.md in the local tier), the findings that shape it:

- A menu after a report is shown only when someone can answer it: the input and the output are both a Terminal, and nothing has asked for no prompts. Otherwise the board prints alone, with no menu and no colour or cursor codes (GitHub CLI's prompts; clig.dev).
- A numbered, type-the-number form is available from the first version, for screen readers and for anyone who sets it, in the text format the accessible prompters of GitHub CLI and huh use. Claude Code's own screen-reader mode turns arrow-key menus into numbered lists the same way.
- One question object feeds both places: a label of 12 characters or fewer, the question, and two to four options, each with a label and what choosing it means. abcd draws it as an arrow-key or numbered menu in a Terminal, and the plugin page hands the same object to Claude Code's question tool. Four options at most keeps the two the same, since the question tool takes no more.
- The standard library and golang.org/x/term (already signed off by itd-2610030810370060) are enough; a menu library would add about 27 modules for one menu.

Typed links: builds on itd-2610031214560142 (the board it ends) and on itd-2610030810370060 (the plain-Terminal question layout it is drawn with), and follows the question layout of itd-2610030810350727 and the asking rules of itd-201. The status it sets when it asks is itd-200's.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Decisions

1. 2026-10-03, the product thinker: the board plus a menu, filed as two drafts (see itd-2610031214560142, decision 1); whether they are one bundle is decided at planning.
2. 2026-10-03, decided without a question (one defensible answer on the record): the menu, not the board, sets the status to waiting on the product thinker, because it is the menu that asks; the interview rules already require the status to be set before any question.
3. 2026-10-03, the product thinker: two records, linked, not one bundle; the board ships when it is ready, and the menu follows once the plain-Terminal question layout exists (itd-2610031215002409 builds on itd-2610031214560142).

## Open Questions

- Which moves the menu offers, and whether they change with the state (for example, "answer the waiting questions" only when something is waiting).
- Whether choosing "start the next thing to build" runs the build loop straight away, or shows what it would do first.
- Whether a person can turn the menu off for good, and where that setting lives.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
