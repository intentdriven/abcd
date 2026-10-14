---
id: itd-2610141646452213
slug: given-a-web-address-or-a-list-of-them
spec_id: null
kind: bundle-member
bundle: ideate-librarian
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Ideate starts from a source through the librarian

## Press Release

> **A person gives ideate a web address, or several, and gets the ideas in it
> tested without rewriting them by hand.** Ideate asks the librarian for a
> reading pack, shows the ranked ideas, and the person picks which to test; with
> `--auto` it ranks them and tests one at a time. Every verdict cites the pack
> items it tested, so a later reader sees exactly which sentence of which source
> an idea came from.
>
> "I pasted the address of an engineering write-up and got five ideas back, each
> with its sentence," said Carol, a tech lead. "I picked two. Both verdicts point
> at the paragraph they came from, and the reviewers had the same pack I did."

## Why This Matters

Ideate today takes one idea already worded. Given a source, the runner improvises
the extraction, and its choices are recorded nowhere (iss-2610141647294219). The product
thinker ruled the selection on 2026-10-14: "I pick from a list, --auto or
something similar let's the agent loose on it's own; ranking, then one-by-one".
One at a time also keeps an automatic run inside the machine's sub-agent cap,
since every idea tested costs a reviewer pair.

Decided in the product thinker's planning interview of 2026-10-14 (session abcd-66), which followed lab lab-261014142726-9c4d759, a hand-run of /abcd:ideate on one external write-up.

## Mechanism

We expect source-to-verdict runs to be reproducible and honest because the
extraction is moved out of the runner's head into a recorded pack (itd-2610141646126492), the
person's pick is recorded, and the verdict's citations to pack items are
resolved by the binary like any other citation (itd-2610141646454347).

## Scope Conditions

- Input is a web address, a list of them, or local files; a typed idea keeps
  working exactly as today.
- The leg-3 reviewers receive the reading pack as material; authorship signals
  stay stripped as the command page requires.

## Acceptance Criteria

- **Given** a web address as ideate's input, **when** the run starts, **then** the
  librarian (itd-2610141646126492) produces a reading pack and the person is shown its ranked
  ideas to pick from.
- **Given** `--auto`, **when** the run starts, **then** the ideas are ranked and
  tested one at a time, with no question asked, and never more reviewers at once
  than one idea needs.
- **Given** a recorded verdict from a source run, **when** it is read, **then** it
  cites the pack items it tested, and each citation resolves.
- **Given** a typed idea, **when** ideate runs, **then** no librarian run is
  required and the verdict is recorded as today.

## Open Questions

- The flag's name (`--auto` or another) and how a ranking is shown (technical
  facilitator, at planning).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
