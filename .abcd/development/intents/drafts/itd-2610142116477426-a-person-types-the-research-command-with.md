---
id: itd-2610142116477426
slug: a-person-types-the-research-command-with
spec_id: null
kind: bundle-member
bundle: ideate-librarian
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610141646126492]
related_issues: [iss-2610100649495706]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A research question becomes a shared, citable note

## Press Release

> **A person asks abcd a research question and gets a shared note their
> decisions can cite.** The person types the research command with a question,
> and with sources if they have some. An agent runs the librarian's engine on it,
> the person reads the private result, and once they approve, abcd writes a
> dated research note into the project: the question, the ranked findings and
> the public sources each rests on. It arrives uncommitted, so it is shared only
> when the person commits it.
>
> "Before choosing how players trade, I asked for the state of the art in local
> multiplayer economies," said Iris, a product lead. "The note went into the
> project, and the design decision cites it, so the team can check what we
> leaned on."

## Why This Matters

A managed repository asked abcd for a command that researches a question and
keeps the result as a research note a decision can cite (iss-2610100649495706).
The librarian's reading pack is private by design, so a shared decision cannot
cite it. The product thinker ruled on 2026-10-14 that the research ideate needs
is agent-driven (the librarian), and that the shared note is human-driven, by a
command the person types: research, on the same engine.

## What's In Scope

- The research command (`/abcd:research <question>`, a page for people) and the
  verb that writes the note from an approved pack.
- The note's shape and where it lives: `.abcd/development/research/notes/`,
  cited by its path.

## What's Out of Scope

- The engine and the pack (itd-2610141646126492).
- Committing the note: that is the person's act.

## Mechanism

We expect shared research to be checkable because a decision can cite a note
whose every finding names its public source, and the note is written only from
a pack whose quotes matched their filed text. Shown wrong if a committed note
holds a finding without a public source, or text from a source the person handed
over as private.

## Scope Conditions

- The note carries only what may be shared: findings from public sources, brief
  quotes within each source's terms, and no material from a source the person
  handed over as private or from a corpus source in the confidential class that
  has not been flipped to citable.
- The person's approval is a question in the session; the guarantee that nothing
  is shared without them is that the note arrives uncommitted.

## Acceptance Criteria

- **Given** a question, **when** the person runs the research command, **then** an
  agent records a reading pack for it, and the person is shown the ranked
  findings before anything is written to the project.
- **Given** the person's approval of a pack, or of chosen items, **when** the note
  is written, **then** it lands uncommitted under
  `.abcd/development/research/notes/` with the date, the question, the ranked
  findings and each finding's public source.
- **Given** a pack item from a source the person handed over as private, or from
  a confidential corpus source not flipped to citable, **when** the note is
  written, **then** the item is left out and the person is told how many were.
- **Given** a note about to be written, **when** a finding holds a banned phrase
  or a name the redactor finds, **then** writing the note is refused, naming the
  finding.
- **Given** a committed note, **when** an intent or ADR cites it by path, **then**
  the record gates resolve the citation.

## Prior Art

- iss-2610100649495706 (open): the request this delivers.
- The research notes under `.abcd/development/research/notes/`: the existing,
  hand-written home of such notes.
- itd-36 (shipped): host-composed content, binary-checked, the same split.

## Open Questions

- The verb that writes the note from a pack (technical facilitator, at
  planning).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
