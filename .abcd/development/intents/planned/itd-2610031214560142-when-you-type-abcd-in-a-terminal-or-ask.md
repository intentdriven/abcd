---
id: itd-2610031214560142
slug: when-you-type-abcd-in-a-terminal-or-ask
spec_id: spc-2610031844142274
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
related_issues: [iss-2610031207397996, iss-2609201954342967]
related_intents: [itd-200]
related_adrs: [adr-49]
impact: additive
---

# The board shows a product thinker what waits on them and what comes next, in a few plain lines

## Press Release

> When you open abcd's board in a Terminal, or ask for it in a Claude Code session, you see in a few plain lines where things stand and what comes next. The top line says whose view it is, 'view for the product thinker', in the colour the status line already gives you, and the words alone carry the meaning, so nothing is lost where colour is off. Below it come what is being built now and the next three things to build, each title fitted to your window, then how many more are ready and how many are parked for later. Every state is said by a word and a symbol as well as a colour. The board always opens on your view, and one step opens the view for the facilitator, with the details a developer works from. In the host session you see the board abcd draws, passed on unchanged rather than retold. Showing the board never marks anyone as owing an answer.

_Proposed by the facilitator from decisions 2 and 5 to 13 (2026-10-03); confirmed as written by the product thinker, 2026-10-03._

Superseded wording (filed 2026-10-03, kept for the record):

> When you type abcd in a Terminal, or ask for the board in a Claude Code session, you see in a few lines what is waiting on you and what comes next, in plain words. The top line says whose view it is, 'view for the product thinker', on the amber the status line already uses for you. Below it come what is being built now and the next three things to build, each title fitted to your window, with how many more are ready or parked for later. Every state is said by a word and a symbol as well as a colour, so nothing is lost without colour. One step expands it to the view for the facilitator, labelled in the facilitator's grey, with the record ids, the other working copies, the reviews and the receipts. Inside a Claude Code session the same board appears, drawn by abcd itself rather than retold by the agent.

## Why This Matters

On 2026-10-03 the product thinker ran bare `abcd` in an 81-column Terminal and got a board that was hard to read (their screenshot was not kept in the record). [F, unverified: "about forty lines"; a piped render in a worktree the same day counted 32.] The long lines wrapped back to the left edge, and most rows answer a developer's question, not theirs: `git repo: true`, `record: true`, `work tiers: [development work work.local]` (raw values printed as Go formats them), unpinned review folders, receipt pins, and sixteen-digit record ids. [F, unverified: the one row that says the reader is needed ("waiting on the product thinker") sat on the fifth line, below the banner.] Inside a host session the agent is told to summarise the board's JSON (the plugin page for the bare command says so), so each session words it differently.

The line-wrapping defect is captured on its own (iss-2610031207397996) and is fixed first; that defect also owns the next-up intent listed under both Now and Next, with its brief-chapter edit (decision 9). This intent is the board a product thinker reads.

State of the art (2026-10-03, reports/sota-board-visuals.md in the local tier), the findings that shape it:

- Lead with what waits on the reader, then sections in a fixed order, then one line naming the next command (GitHub CLI's status view, git's status hints, clig.dev: "By default, don't output information that's only understandable by the creators of the software"). Here the lead is whose view it is (decision 2), the status line carries "waiting" (decision 6), and the product thinker's view names no command: the next step lives in the menu of itd-2610031215002409.
- Every state is shown by a word and a symbol as well as a colour (WCAG 1.4.1).
- [F] The research ranks the 16 colours of the person's own terminal theme second, with red and yellow kept for problems. The view label departs from it: it takes itd-200's fixed role pair, painted as 24-bit colour and provisional until itd-200's third open question lands, so a terminal without 24-bit colour shows the label's words unpainted (decision 10). Every other state on the board keeps to the 16 colours.
- Counts are given as numbers in words. A Now / Next / Later bar is rejected: at 1, 14 and 106 the bar says only that most work is parked. A filled bar earns its place only where one item has a real done-out-of-total, such as the steps of the intent being built.
- Symbols come from a set every common terminal font carries (● ○ • … █ ░ ─), with a plain-text fallback where the terminal does not advertise UTF-8. Activity sparklines are rejected: font gaps, screen readers speak each block aloud, and they answer no question this reader has.
- [F, unverified: the research files these under user reports, not documentation.] Inside the host session the reply renders bold, lists and code blocks but no colour, and wide tables collapse into stacked cards, so the session board is a list, not a table. One renderer in the binary feeds both places: the plugin page relays abcd's own markdown form rather than having the agent reformat the JSON, which drifts from one model to the next and costs more tokens. Whether the agent relays it unchanged has not been measured, so an eval that it does is owed (decision 12).

Typed links (front matter): related to iss-2610031207397996 (the width fix, fixed first; the board draws its titles through the same wrap-to-window helper, internal/textwidth.Wrap) and to iss-2609201954342967 (folded in by decision 7: its spec-id and in-flight rows join the facilitator's view here, its next-actions list joins the menu, and it closes when both ship). Related to itd-200 (the role colour pair the view label takes) and adr-49 (terminal decoration). itd-2610031215002409 (the "what next?" menu) builds on this one. The menu is the one exception to bare `abcd`'s read-only promise, in a Terminal only (decision 5); the board itself still writes nothing.

## Mechanism

[F] We expect a product thinker to name the next thing to be built within one screen of an 80-column Terminal, without scrolling, because their view carries only whose view it is, titles, counts and state words, in a fixed order fitted to the window, and no record id or developer row. The claim is falsified if, in the dated receipt of criterion A4, the product thinker cannot name the next item from the first screen.

_Proposed by the facilitator, 2026-10-03; confirmed by the product thinker, 2026-10-03._

## Scope Conditions

- A Terminal at least 80 columns wide, on macOS and Linux, the two systems abcd is released for. <!-- cond: cond-2610031844149402 -->
- A repository abcd manages, whose intent store and ledger are those of this working copy; a record held only in another working copy is not counted. <!-- cond: cond-2610031844141308 -->
- Output that is not a Terminal (a file, another program) gets plain text with no colour or cursor codes (adr-49, invariant 13). <!-- cond: cond-2610031844146361 -->
- The view label's colour shows only where the terminal paints 24-bit colour; elsewhere, and under NO_COLOR, the words alone carry it. <!-- cond: cond-2610031844148905 -->
- In the host session the board is abcd's markdown form passed on unchanged; that the agent does so is measured by an eval, not assumed. <!-- cond: cond-2610031844148490 -->

_Proposed by the facilitator, 2026-10-03; confirmed by the product thinker, 2026-10-03._

## Acceptance Criteria

_Agent-seeded from the design review's A1 to A7 of 2026-10-03 as they survive decisions 5 to 13; all unconfirmed, each walked with its addressee at the planning interview._

- A1 (product thinker; CONFIRMED 2026-10-03; go test) Given a managed fixture with one intent being built, three more READY and 40 parked, and a stored status naming either role, when `abcd` runs on an 80-column Terminal, then line 1 reads "view for the product thinker", followed by one Now title, three Next titles and "N more ready, 40 parked", with no owed-answer line, no record id and no command word (golden; example: "view for the product thinker", "● building: The board shows…", "○ next: …" three times, "11 more ready, 40 parked").
- A2 (technical facilitator; CONFIRMED 2026-10-03; go test) Given the same fixture, when stdout is a pipe or `--json` is passed, then no ESC byte is emitted and the text form still carries "view for the product thinker" (invariant 13).
- A3 (technical facilitator; CONFIRMED 2026-10-03; go test) Given the markdown form, then the output is a markdown list with no table and no ANSI, byte-equal to its golden, and the plugin page for the bare command runs that exact invocation and tells the agent to paste it unchanged in one fenced block (a page-reading test, as the existing page tests do).
- A4 (product thinker; CONFIRMED 2026-10-03; dated receipt from an eval) Given the board asked for in a host session, then the fenced block in the reply diffs empty against the markdown form run in the same checkout, on three sessions across two models, and the product thinker names the next item from it.
- A5 (technical facilitator; CONFIRMED 2026-10-03; go test) Given the facilitator's view asked for (by the menu in a Terminal, by a flag in a pipe or a session), then the label reads "view for the facilitator", every row of today's board is present (ids, presence, peers, inbox, oracle, reviews, receipts), and each planned intent carries its spec id and an in-flight marker where its spec is open and its branch exists (golden on today's fixture plus one in-flight spec; example: "itd-… spc-… in flight").
- A6 (product thinker; CONFIRMED 2026-10-03; go test) Given NO_COLOR or `--no-color` on a Terminal, then the same lines appear uncoloured, each state word with its symbol (● ○ •), and with the plain-text fallback when the locale is not UTF-8 (example: "* building:" in place of "● building:").
- A7 (technical facilitator; CONFIRMED 2026-10-03; go test) Given a title longer than the window and a title of wide glyphs, when drawn at 80 columns, then no line exceeds 80 columns as internal/textwidth.Columns counts them.

_Impact expected: additive. The board gains a view and a markdown form; nothing a person relies on today is removed, because the facilitator's view keeps today's rows (decision 11)._

## Review findings (design and record discipline, 2026-10-03)

Read from reports/review-board-design.md and reports/review-board-records.md in the local tier.

Applied: typed links moved to `related_issues`, `related_intents` and `related_adrs`, `builds_on` kept for intents only, and both issues name this draft back (records 2); decision 4 marked as superseding decision 1's bundle question (records 3; decision 13); the 16-colour sentence corrected and marked [F] (design 2, records 4); the forty-line count, the fifth-line placement and the session-rendering facts marked [F] and unverified (records 5); the duplicated next-up row dropped here and owned by the defect with its brief-chapter edit (design 4, records 6; decision 9); "Claude Code" named once in the press release, "the host session" after (records 7); Mechanism, Scope Conditions and Acceptance Criteria seeded (records 8); one renderer with a markdown form, relayed unchanged and checked by an eval (design 1; decision 12); the facilitator's view absorbs iss-2609201954342967's spec-id and in-flight rows (design 6; decision 7).

Overtaken by the product thinker's rulings: the design review's owed-answer badge as line 1, and its A1 leading with it (design 3), by decision 6, the words suffice; its default view from the stored status (design 5), by decision 8, always the product thinker's; the reversal named until ruled (records 1), by decision 5.

Kept for the spec: what the board can derive about a waiting answer is a role and the build it sits in, never the question's text (design 3); one-line ellipsis in the product thinker's view and wrapping in the facilitator's, both through internal/textwidth (design 6); `--no-color`'s help widened from "the banner" to the board (design 2).

## Decisions

1. 2026-10-03, the product thinker, choosing among four directions for the board (tidy only; a product-thinker board; that board plus a menu; decide later): the board plus a menu. Filed as two drafts, this one and itd-2610031215002409, because the board does not depend on the plain-Terminal question layout and the menu does; whether they are planned as one bundle is the product thinker's, at planning.
2. 2026-10-03, the product thinker, on how the board uses the role colours: the board leads with whose view it is, "view for the product thinker" on the product thinker's amber, with one step that expands it to "view for the facilitator", whose label then turns to the facilitator's grey. The board only reads the stored status and never sets it; a question the menu asks sets it, under the interview rules.
3. 2026-10-03, decided without a question (one defensible answer on the record): showing the board does not mark anyone as owing an answer, because the board's page promises zero writes and the status means an answer is owed.
4. 2026-10-03, the product thinker: two records, linked, not one bundle; the board ships when it is ready, and the menu follows once the plain-Terminal question layout exists (itd-2610031215002409 builds on itd-2610031214560142).
5. 2026-10-03, the product thinker, asked whether bare `abcd`, promised read-only, may write the waiting-on status when the "what next?" menu asks, with the choices "the menu is the one write bare `abcd` makes, on a Terminal only, and the chapter and page say so", "the menu has its own word or flag and bare `abcd` stays read-only" and "decide later": the menu is the one exception, in a Terminal only, and the brief chapter 04-surfaces/08-abcd.md and the plugin page commands/abcd.md say so. This reverses their "strictly read-only" and "zero writes" promise, by the product thinker's ruling; both edits are owed in the spec (recorded in full in itd-2610031215002409, decision 4). Decision 3 stands for the board itself, which still writes nothing.
6. 2026-10-03, the product thinker, asked whether the role colour meaning "view for" on the board and "an answer is waiting on" in the status line needs more than the words, with the choices "the words suffice", "the board also carries one owed-answer line under the view label" and "decide later": the words suffice. The board carries the view label only; the status line carries "waiting".
7. 2026-10-03, the product thinker, asked what becomes of iss-2609201954342967, with the choices "fold its next-actions list into the menu and its spec-id rows into the facilitator's view", "keep it as its own lane behind both" and "decide later": fold it in. Its next-actions list joins the menu (itd-2610031215002409), its spec-id and in-flight rows join the facilitator's view here, and the issue closes when they ship.
8. 2026-10-03, the product thinker, asked which view bare `abcd` opens, with the choices "the product thinker's, always", "the role the stored status names, else the product thinker's", "the last one chosen" and "decide later": always the product thinker's.
9. 2026-10-03, the technical facilitator, decided without a question: the next-up intent listed under both Now and Next is the defect's (iss-2610031207397996), fixed there with its edit to the brief chapter 04-surfaces/08-abcd.md and the status-block golden, and is dropped from this draft.
10. 2026-10-03, the technical facilitator, decided without a question: the view label's colour is itd-200's fixed role pair; the words carry the meaning, and NO_COLOR is honoured.
11. 2026-10-03, the technical facilitator, decided without a question: the facilitator's view keeps today's rows, with iss-2609201954342967's spec-id and in-flight rows added (decision 7).
12. 2026-10-03, the technical facilitator, decided without a question: in a host session the board is abcd's own markdown form pasted unchanged, and the check that the agent relays it unchanged is an eval.
13. 2026-10-03, the technical facilitator, decided without a question: decision 4 (two records, linked, not one bundle) supersedes decision 1's question of whether the two are planned as one bundle.
14. 2026-10-03, the product thinker: the revised press release confirmed as written.

## Open Questions

- The role colours would now mean two things: on the board "this view is for", and in the status line "an answer is waiting on". The words differ, so the two can be told apart; the question is whether that is enough, or whether the board also shows whose answer is owed as its own line. _Answered by decision 6._
- What the view for the facilitator holds, and whether it replaces today's full board exactly or regroups it. _Answered by decision 11._
- Whether the session board is relayed by injecting abcd's output into the plugin page, or by the agent running it and relaying it, and how the check that the agent relays it unchanged is run. _Answered by decision 12: the agent runs the markdown form and pastes it; the check is an eval._
- Which view a bare `abcd` opens on when nobody is waiting: the product thinker's, or the last one chosen. _Answered by decision 8._
- [F] How the facilitator's view is asked for where the menu cannot be shown (a pipe, a session): a flag on the bare command is the design review's proposal (its finding 5); the spelling is the technical facilitator's, at the spec.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: a product thinker sees what is next and takes the next step without learning abcd's commands (the product thinker, 2026-10-03).
