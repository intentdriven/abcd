---
id: itd-2610031215002409
slug: under-the-board-a-short-what-next-menu
spec_id: spc-2610031844158884
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610031214560142, itd-2610030810370060]
severity: minor
origin: researcher-authored
production_mode: hand-written
related_issues: [iss-2609201954342967]
related_intents: [itd-2610030810350727, itd-201, itd-200]
related_adrs: [adr-49]
impact: additive
---

# A 'what next?' menu under the board offers the next step, in a Terminal and in a Claude Code session

## Press Release

> Under the board, a short 'what next?' menu offers at most four steps that make sense now: start building the next item, see everything that is ready, open the view for the facilitator, and 'Nothing for now', always last. When an answer is waiting on you, 'Answer what is waiting' takes the place of 'See everything that is ready'. In a Terminal you choose with the arrow keys or by number, drawn the same way as abcd's other questions; in a Claude Code session the same choices arrive as a question in the host session. Choosing to start building begins at once, because the choice already names what will be built. While the menu waits, the status line says an answer is waiting on whoever the view is for, and it clears when you answer or leave; in a Terminal this is the one thing opening the board ever changes. Where nobody can answer, such as output going to a file or another program, the board prints alone with no menu, and one setting on your machine turns the menu off for good.

_Proposed by the facilitator from decisions 4 to 11 (2026-10-03); confirmed as written by the product thinker, 2026-10-03._

Superseded wording (filed 2026-10-03, kept for the record):

> Under the board, a short 'what next?' menu offers the few steps that make sense now: start the next thing to build, see everything that is ready, open the view for the facilitator, or nothing for now. In a Terminal you choose with the arrow keys or by number, drawn the same way as abcd's questions; inside a Claude Code session the same choices arrive as a question. Asking you something marks the status as waiting on you, and answering clears it. Where nobody can answer, such as output going to a file or another program, the board prints with no menu.

## Why This Matters

A board says where things stand. What a product thinker usually wants next is one of a few moves, and today each needs a command they are not assumed to know (the product thinker's request of 2026-10-03, choosing "the board plus a menu"). Ending the board on the moves themselves turns it from a report into a starting point, and it gives the board's "expand to the view for the facilitator" step somewhere to live.

State of the art (2026-10-03, reports/sota-board-visuals.md in the local tier), the findings that shape it:

- A menu after a report is shown only when someone can answer it: the input and the output are both a Terminal, and nothing has asked for no prompts. Otherwise the board prints alone, with no menu and no colour or cursor codes (GitHub CLI's prompts; clig.dev).
- A numbered, type-the-number form is available from the first version, for screen readers and for anyone who sets it, in the text format the accessible prompters of GitHub CLI and huh use. Claude Code's own screen-reader mode turns arrow-key menus into numbered lists the same way.
- One question object feeds both places: a label of 12 characters or fewer, the question, and two to four options, each with a label and what choosing it means. abcd draws it as an arrow-key or numbered menu in a Terminal, and the plugin page hands the same object to the host's question tool. Four options at most keeps the two the same; [F] that the question tool takes no more than four is read from the tool's schema, not its documentation.
- The standard library and golang.org/x/term are enough; a menu library would add about 27 modules for one menu. golang.org/x/term is signed off by itd-2610030810370060 (its decision 5) and adopted as a direct dependency by whichever of iss-2610031207397996, that intent or this one is built first.

Typed links (front matter): builds on itd-2610031214560142 (the board it ends) and on itd-2610030810370060 (the plain-Terminal question layout it is drawn with); related to itd-2610030810350727 (the question layout it follows, unchanged), itd-201 (the asking rules), itd-200 (the status it sets when it asks) and adr-49 (terminal decoration). Related to iss-2609201954342967, whose part (1), a short next-actions list, is folded into this menu by decision 5 (its part (2) joins the board's facilitator view). The menu is the one exception to bare `abcd`'s read-only promise, in a Terminal only, by the product thinker's ruling (decision 4); the brief chapter 04-surfaces/08-abcd.md and the plugin page commands/abcd.md owe the matching edit.

## Mechanism

[F] We expect a product thinker to take the next step from the board without knowing any command, because the board ends on at most four moves that follow the state, each saying what choosing it does. The claim is falsified if, in the dated receipt of criterion B6, the product thinker reaches for a command they were not shown to take the step they wanted after reading the board.

_Proposed by the facilitator, 2026-10-03; confirmed by the product thinker, 2026-10-03._

## Scope Conditions

- In a Terminal, only where both input and output are a Terminal, in a repository abcd manages whose local tier exists, and the machine setting has not turned the menu off. <!-- cond: cond-2610031844154919 -->
- macOS and Linux, the two systems abcd is released for. <!-- cond: cond-2610031844150092 -->
- In a host session, the host's question tool asks and today's status rules set and clear the waiting-on status; abcd itself writes it only in a Terminal, where it takes the answer. <!-- cond: cond-2610031844155104 -->
- One person answers at a time; the status the menu sets follows the view shown, not a guess at who is at the keyboard. <!-- cond: cond-2610031844153513 -->
- At most four moves, the limit both the Terminal layout and the host's question tool take. <!-- cond: cond-2610031844156951 -->

_Proposed by the facilitator, 2026-10-03; confirmed by the product thinker, 2026-10-03._

## Acceptance Criteria

_Agent-seeded from the design review's B1 to B7 of 2026-10-03 as they survive decisions 4 to 11; all unconfirmed, each walked with its addressee at the planning interview._

- B1 (product thinker; CONFIRMED 2026-10-03; go test with an injected terminal) Given terminals in and out in a managed repository with a READY intent and nothing waiting, when the board ends, then a "Product Q1" menu shows four options, the last "Nothing for now", chosen by the arrow keys or by number (example: 1 Start building <title>; 2 See everything that is ready; 3 Open the view for the facilitator; 4 Nothing for now).
- B2 (product thinker; CONFIRMED 2026-10-03; go test) Given an answer waiting on the product thinker, then "Answer what is waiting" takes the place of "See everything that is ready" and there are still at most four options with "Nothing for now" last; given no READY intent, "Start building…" is absent and at least two options remain (example: 1 Answer what is waiting; 2 Open the view for the facilitator; 3 Nothing for now).
- B3 (technical facilitator; CONFIRMED 2026-10-03; go test) Given stdin or stdout not a terminal, the machine setting turning the menu off, or a repository with no local tier, then the board prints with no menu and no ESC byte, and the status store's bytes are the same before and after.
- B4 (technical facilitator; CONFIRMED 2026-10-03; go test) Given the menu drawn under the product thinker's view, then while it waits the status reads product-thinker and the question marker exists, and under the facilitator's view the chip reads "Tech Q1" and the status reads facilitator; after an answer, Ctrl-C or end of input the status reads managed, the marker is gone and the Terminal is restored.
- B5 (technical facilitator; CONFIRMED 2026-10-03; go test) Given the menu's question object for either view and either state, then question.CheckLimits returns no finding for the Product or the Tech addressee.
- B6 (product thinker; CONFIRMED 2026-10-03; dated receipt) Given the board asked for in a host session, then the same options arrive through the host's question tool with the same labels and descriptions, the status line reads "waiting on the product thinker" while it waits and "abcd-managed" after the answer, and the product thinker takes the step they wanted without a command.
- B7 (product thinker; CONFIRMED 2026-10-03; go test) Given "Start building <title>" chosen, then the build loop starts on that item at once with no second question, and a refusal (an open question, a hold, a peer holding it) is shown in plain words under the board (example: "Not started: another working copy is building <title>.").

_Impact expected: additive. The menu is new and can be switched off for good; the board it ends prints as before wherever nobody can answer._

## Review findings (design and record discipline, 2026-10-03)

Read from reports/review-board-design.md and reports/review-board-records.md in the local tier.

Applied: the write the menu makes is ruled by the product thinker as the one exception, in a Terminal only, with the chapter and page edits owed (records 1, decision 4); decision 2 amended by a decision line so the status the menu sets follows the view shown (records 2, design 3; decision 9); typed links moved to `related_issues`, `related_intents` and `related_adrs`, the question layout `related_intents`, not `refines`, because the menu does not change it (records 3); "the question tool takes no more" marked [F], and golang.org/x/term adopted by whichever lands first (records 4); iss-2609201954342967 named with its part (1) and folded in (records 5; decision 5); the off switch split into the product thinker's yes or no and the location, both now ruled (records 6; decision 8); decision 3 marked as superseding decision 1's bundle question (records 3 of the board; decision 11); "Claude Code" named once in the press release, "the host session" after; Mechanism, Scope Conditions and Acceptance Criteria seeded.

Overtaken: the design review's classing the write as mechanism (design 1, 2), by the product thinker's ruling (decision 4), though its rule stands as the how: abcd sets the status only where it takes the answer itself, sets it before drawing, and clears it on every exit, answer, Ctrl-C and end of input alike, beside the Terminal restore of itd-2610030810370060; its B3 fixing the status to product-thinker, by decision 9; its three moves plus a conditional one displacing any (design 3), by decision 6, which names the one displaced.

Kept for the spec: CheckLimits over the menu object, with "Nothing for now" as the decide-later option (design 3); rule 10's "Now:" and "Change later:" lines, where the product thinker's "Change later" may name no command (design 3); the numbered form forced by TERM=dumb and the accessibility setting (design 4); a per-run form of the off switch (design 6).

## Decisions

1. 2026-10-03, the product thinker: the board plus a menu, filed as two drafts (see itd-2610031214560142, decision 1); whether they are one bundle is decided at planning.
2. 2026-10-03, decided without a question (one defensible answer on the record): the menu, not the board, sets the status to waiting on the product thinker, because it is the menu that asks; the interview rules already require the status to be set before any question.
3. 2026-10-03, the product thinker: two records, linked, not one bundle; the board ships when it is ready, and the menu follows once the plain-Terminal question layout exists (itd-2610031215002409 builds on itd-2610031214560142).
4. 2026-10-03, the product thinker, asked whether bare `abcd`, promised read-only, may write the waiting-on status when the menu asks, with the choices "the menu is the one write bare `abcd` makes, on a Terminal only, and the chapter and page say so", "the menu has its own word or flag and bare `abcd` stays read-only" and "decide later": the menu is the one exception, in a Terminal only, and the brief chapter 04-surfaces/08-abcd.md and the plugin page commands/abcd.md say so. This reverses their "strictly read-only" and "zero writes" promise, by the product thinker's ruling; both texts are owed edits in the spec.
5. 2026-10-03, the product thinker, asked what becomes of iss-2609201954342967, with the choices "fold its next-actions list into the menu and its spec-id rows into the facilitator's view", "keep it as its own lane behind both" and "decide later": fold it in. Its next-actions list joins this menu, its spec-id and in-flight rows join the board's facilitator view (itd-2610031214560142, decision 7), and the issue closes when they ship.
6. 2026-10-03, the product thinker, asked which moves the menu offers, with the example list "start building <title>; see everything ready; open the facilitator's view; nothing for now", and whether "answer what is waiting" appears only when something waits: at most four moves, "Nothing for now" last; "Answer what is waiting" appears when something waits, and then replaces "See everything that is ready".
7. 2026-10-03, the product thinker, asked whether "Start building the next item" starts at once or shows what it would do first, with the choices "starts at once, its description naming the item", "shows what it would do first" and "decide later": it starts at once; its description names the item, so choosing it is the go-ahead.
8. 2026-10-03, the product thinker, asked whether the menu can be switched off for good, with the choices "yes, a setting", "no, only per run" and "decide later": yes, by a setting, a machine setting beside the status-line settings, never the repository.
9. 2026-10-03, the technical facilitator, decided without a question: the status the menu sets follows the view shown (product-thinker under the product thinker's view, facilitator under the facilitator's), amending decision 2's "waiting on the product thinker" for the facilitator's view, under itd-201's rule that the register follows the addressee.
10. 2026-10-03, the technical facilitator, decided without a question: with stdin or stdout not a terminal there is no menu and no escape byte (adr-49, invariant 13).
11. 2026-10-03, the technical facilitator, decided without a question: decision 3 (two records, linked, not one bundle) supersedes decision 1's question of whether the two are one bundle, as itd-2610031214560142's decision 13 records for the pair.
12. 2026-10-03, the product thinker: the revised press release confirmed as written.

## Open Questions

- Which moves the menu offers, and whether they change with the state (for example, "answer the waiting questions" only when something is waiting). _Answered by decision 6._
- Whether choosing "start the next thing to build" runs the build loop straight away, or shows what it would do first. _Answered by decision 7._
- Whether a person can turn the menu off for good, and where that setting lives. _Answered by decision 8._
- [F] How iss-2609201954342967's derived next actions (an owed fidelity audit, an unplanned draft with every decision recorded, a stale claim) map onto at most four moves, and which moves the facilitator's view offers in place of "Open the view for the facilitator": the technical facilitator's, at the spec.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: a product thinker sees what is next and takes the next step without learning abcd's commands (the product thinker, 2026-10-03).
