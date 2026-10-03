---
id: itd-2610031214560142
slug: when-you-type-abcd-in-a-terminal-or-ask-for-the-board-in-a
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# The board shows a product thinker what waits on them and what comes next, in a few plain lines

## Press Release

> When you type abcd in a Terminal, or ask for the board in a Claude Code session, you see in a few lines what is waiting on you and what comes next, in plain words. The top line says whose view it is, 'view for the product thinker', on the amber the status line already uses for you. Below it come what is being built now and the next three things to build, each title fitted to your window, with how many more are ready or parked for later. Every state is said by a word and a symbol as well as a colour, so nothing is lost without colour. One step expands it to the view for the facilitator, labelled in the facilitator's grey, with the record ids, the other working copies, the reviews and the receipts. Inside a Claude Code session the same board appears, drawn by abcd itself rather than retold by the agent.

## Why This Matters

On 2026-10-03 the product thinker ran bare `abcd` in an 81-column Terminal and got a board of about forty lines that was hard to read (their screenshot was not kept in the record). The long lines wrapped back to the left edge, and most rows answer a developer's question, not theirs: `git repo: true`, `record: true`, `work tiers: [development work work.local]`, unpinned review folders, receipt pins, and sixteen-digit record ids. The intent marked next also appears twice, under Now and under Next. The one row that says the reader is needed ("waiting on the product thinker") is the fifth line, below the banner. The same happens inside a Claude Code session, where the agent retells the board in its own words, so each session draws it differently.

The line-wrapping defect is captured on its own (iss-2610031207397996) and is fixed first; this intent is the board a product thinker reads.

State of the art (2026-10-03, reports/sota-board-visuals.md in the local tier), the findings that shape it:

- Lead with what waits on the reader, then sections in a fixed order, then one line naming the next command (GitHub CLI's status view, git's status hints, clig.dev: "By default, don't output information that's only understandable by the creators of the software").
- Every state is shown by a word and a symbol as well as a colour (WCAG 1.4.1). Colours come only from the 16 the person's own terminal theme sets, and red and yellow are kept for problems.
- Counts are given as numbers in words. A Now / Next / Later bar is rejected: at 1, 14 and 106 the bar says only that most work is parked. A filled bar earns its place only where one item has a real done-out-of-total, such as the steps of the intent being built.
- Symbols come from a set every common terminal font carries (● ○ • … █ ░ ─), with a plain-text fallback where the terminal does not advertise UTF-8. Activity sparklines are rejected: font gaps, screen readers speak each block aloud, and they answer no question this reader has.
- Inside a Claude Code session the reply renders bold, lists and code blocks but no colour, and wide tables collapse into stacked cards, so the session board is a list, not a table. One renderer in the binary feeds both places: the plugin page relays abcd's own drawing rather than having the agent reformat the JSON, which drifts from one model to the next and costs more tokens. Whether the agent relays it unchanged has not been measured, so a check that it does is owed.

Typed links: builds on iss-2610031207397996 (the width fix, whose wrap-to-window helper this board draws through). Interacts with iss-2609201954342967, which adds a next-actions row and each planned intent's spec to the same board; one of the two absorbs the other at planning. Uses the role badge colours of itd-200 (amber for the product thinker, light grey for the facilitator) and the decoration rules of adr-49. itd-2610031215002409 (the "what next?" menu) builds on this one.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Decisions

1. 2026-10-03, the product thinker, choosing among four directions for the board (tidy only; a product-thinker board; that board plus a menu; decide later): the board plus a menu. Filed as two drafts, this one and itd-2610031215002409, because the board does not depend on the plain-Terminal question layout and the menu does; whether they are planned as one bundle is the product thinker's, at planning.
2. 2026-10-03, the product thinker, on how the board uses the role colours: the board leads with whose view it is, "view for the product thinker" on the product thinker's amber, with one step that expands it to "view for the facilitator", whose label then turns to the facilitator's grey. The board only reads the stored status and never sets it; a question the menu asks sets it, under the interview rules.
3. 2026-10-03, decided without a question (one defensible answer on the record): showing the board does not mark anyone as owing an answer, because the board's page promises zero writes and the status means an answer is owed.
4. 2026-10-03, the product thinker: two records, linked, not one bundle; the board ships when it is ready, and the menu follows once the plain-Terminal question layout exists (itd-2610031215002409 builds on itd-2610031214560142).

## Open Questions

- The role colours would now mean two things: on the board "this view is for", and in the status line "an answer is waiting on". The words differ, so the two can be told apart; the question is whether that is enough, or whether the board also shows whose answer is owed as its own line.
- What the view for the facilitator holds, and whether it replaces today's full board exactly or regroups it.
- Whether the session board is relayed by injecting abcd's output into the plugin page, or by the agent running it and relaying it, and how the check that the agent relays it unchanged is run.
- Which view a bare `abcd` opens on when nobody is waiting: the product thinker's, or the last one chosen.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
