---
id: itd-2610030810370060
slug: a-person-can-run-abcd-s-interviews-in-a
spec_id: spc-2610030911534855
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610030810350727, itd-112]
severity: minor
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609201916056194, itd-2610030821294016]
related_adrs: [adr-49]
impact: additive
---

# abcd's interviews run in a plain Terminal, with the same question layout

## Press Release

> A person can run abcd's interviews in a plain Terminal, without Claude Code. The same layout the host shows (who the question is for, where they are, the thing being decided, the options and what each means) is drawn by abcd itself, reads on a narrow window and uses a wide one, and the answers land in the same records an interview inside Claude Code writes. A long list of choices can be searched by typing and moved through with the arrow keys.

_Confirmed as written by the product thinker at the planning interview, 2026-10-03._

## Why This Matters

The product thinker asked on 2026-10-03 that running abcd in the Terminal, outside Claude Code, be considered alongside the question layout (itd-2610030810350727, which this builds on). Today every abcd interview (planning, retrospective, routing, setup) is run by an agent in a host session: abcd's binary renders seeds and validates answers, and the host's agent asks the questions through the host's question view.

A plain-Terminal front door changes who composes and who draws the question. The binary would draw the layout itself, at the window's width. The questions an interview asks are mostly composed by an agent from the record (adr-25: the language model is host-delegated by default), so either a runner the person routed on their own machine composes them (itd-2609201916056194, personal routes only under rulings RN2 and OC2), or the plain-Terminal path covers only interviews whose questions are fixed by the binary (the setup interview, a routing confirmation).

Typed links: builds on itd-2610030810350727 (the layout it draws); relies on itd-2609201916056194 (the command-line runner, shipped in v0.12.0 and so far proven only against stand-in harnesses, no real one; under rulings RN2 and OC2 a role reaches a runner only on the person's own route with a paid key, so a person without one gets the binary-fixed interviews only) where a question must be composed.

Arrow-key lists (routed here by the product thinker on 2026-10-03): a long list of choices, such as a model service's hundreds of models, is moved through with the arrow keys and searched by typing, the way opencode's model picker works; model choice for connecting a service (itd-2610030821294016) is its first example.

State of the art (2026-10-03, reports/sota-question-layout.md in the local tier): a numbered, typed-answer list is the accessible primary mode (Claude Code's screen-reader mode, GitHub CLI's accessible prompter, huh's accessible mode); full-screen redraw harms screen readers; the standard library plus the existing terminal helpers covers the numbered mode with no new module, while an arrow-key list needs raw terminal mode, which the research ranks as costly and worse for accessibility unless the numbered mode stays available.

## Mechanism

- We expect the plain Terminal and Claude Code to look and read the same because abcd draws each question from the same few parts Claude Code is given.
- We expect the AI-written interviews to work without Claude Code because the person's own connection writes each question as those parts and abcd draws it.

_Both claims confirmed by the product thinker at the planning interview, 2026-10-03; each is checked on its own._

## Scope Conditions

- A Terminal at least 80 characters wide and 24 rows tall. <!-- cond: cond-2610030911536412 -->
- macOS and Linux, the two systems abcd is released for. <!-- cond: cond-2610030911539656 -->
- AI-written interviews only with the person's own paid connection set up. <!-- cond: cond-2610030911539816 -->
- Output that is not a Terminal (a file, another program) gets plain text, never a drawn screen. <!-- cond: cond-2610030911530244 -->

_Confirmed by the product thinker at the planning interview, 2026-10-03._

## Acceptance Criteria

_Agent-seeded from the design review of 2026-10-03, unconfirmed: each is walked with its addressee at the planning interview. B3, B4 and B7 are the technical facilitator's; B1, B2, B5 and B6 are the product thinker's._

- B1 (product thinker; CONFIRMED, reworded for the arrow-key default, 2026-10-03) Given an interview whose questions abcd fixes, in a Terminal 80 columns wide, when a question appears, then it shows who it is for and which question this is, the material being decided, the options each with what it means, and the way to decide later, and takes the answer by the arrow keys and Enter, or by its number (example: setup asks where the key is kept; the person sees 'Setup Q1', the question, three options with one line each, and 'decide later').
- B2 (product thinker; CONFIRMED as written, 2026-10-03) Given a Terminal 160 columns wide, then each option and what it means sit side by side and no line of prose runs past 80 columns.
- B3 (facilitator; CONFIRMED as written, 2026-10-03) Given output or input that is not a terminal, then plain text with no colour or cursor codes, answers taken from flags or an answers file, and the record identical to the terminal run.
- B4 (facilitator; CONFIRMED as written, 2026-10-03) Given NO_COLOR or TERM=dumb, then no colour codes, and nothing is conveyed by colour alone.
- B5 (product thinker; CONFIRMED, 2026-10-03) Given one interview answered in Claude Code and once in a Terminal, then the records are the same except the field saying where each answer was given (decision 4).
- B6 (product thinker; CONFIRMED, reworded for the arrow-key default, 2026-10-03) Given 300 choices, when the person types part of a name, then the list narrows, the arrow keys move through what is left, and the Terminal is restored on every exit, an interrupt included (example: typing 'claude' narrows the list to the matching models; Ctrl-C leaves the Terminal behaving normally).
- B7 (facilitator; CONFIRMED as written, 2026-10-03) Given material carrying terminal control codes, when it is drawn, then it is made safe first (invariant 13).

## Review findings (design and feasibility, 2026-10-03, reports/review-qlayout-design.md in the local tier)

Applied: the runner is shipped, and composed interviews in a plain Terminal hold only for a person with their own paid route (finding 6).

Recorded, not settled (each an interview question): the first cut the review and the state-of-the-art pass both recommend is binary-fixed interviews (setup, routing confirmation), with composed interviews under a scope condition, and a runner returning the question object for abcd to draw (6); the review recommends a numbered list with a typed filter as the default and arrow keys later or out of scope, and suggests the arrow-key list belongs with the long-list picker of itd-2610030821294016, while the product thinker routed arrow-key lists here on 2026-10-03, so the placement and the default are the product thinker's to confirm (7, 8); colour on the 16 basic colours, never meaning by colour alone, and material sanitised before it is drawn (9).

Flagged for the product thinker to confirm (record-discipline review, 2026-10-03): brief invariant 3 says every capability is reachable with no plugin host present, which is this draft's strongest grounds; a first cut of binary-fixed interviews only leaves the composed interviews (planning, retrospective) unreachable without a host or a paid route of the person's own. The press release promises interviews in a plain Terminal but not the arrow-key, searchable long list routed here; whether the promise carries the list in one sentence or the list becomes its own record is the product thinker's.

## Decisions

1. 2026-10-03, the product thinker at the planning interview: the first version covers both kinds of interview, those whose questions abcd writes itself (setup, routing confirmation) and those an AI writes from the record (planning, retrospective); the AI-written ones run only through the person's own paid connection (rulings RN2 and OC2). This is wider than the first cut the state-of-the-art pass and the design review recommended; the runner returning a question for abcd to draw becomes part of this intent.
2. 2026-10-03, the product thinker: the searchable long list belongs to this feature, and its promise gains the sentence "A long list of choices can be searched by typing and moved through with the arrow keys."
3. 2026-10-03, the product thinker: arrow keys are the default way to choose from a list, with typing to narrow it; the numbered list is a setting, and is always used where no keyboard mode is possible (output not a terminal, a screen reader). This is against the state-of-the-art pass's and the design review's recommendation of a numbered default, chosen with the accessibility trade-off stated in the question.
4. 2026-10-03, the product thinker: the record marks where each answer was given ("answered in: Terminal" or "answered in: Claude Code"); otherwise the records are the same.
5. 2026-10-03, the technical facilitator, signing off a new dependency: raw terminal mode and window size come from golang.org/x/term (v0.46.0; depends only on golang.org/x/sys, already an indirect dependency), adopted as a direct dependency when this is built. ACKNOWLEDGEMENTS.md gains it in the same change.

## Open Questions

- Which interviews run in a plain Terminal: only those whose questions the binary fixes (setup, routing confirmation), or also those an agent composes from the record (planning, retrospective) through a runner the person routed.
- Keyboard, screen reader and copy-paste behaviour in a plain Terminal, and what happens when output is piped (no terminal): plain text, never a broken frame.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: abcd is meant to work without any one AI tool; today its interviews only run inside Claude Code, and we expect a plain-Terminal path to let you plan and set up from any Terminal, and to show where the interview depends on Claude Code.
