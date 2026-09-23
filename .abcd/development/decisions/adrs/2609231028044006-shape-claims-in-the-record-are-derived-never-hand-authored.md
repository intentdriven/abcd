---
id: adr-2609231028044006
slug: shape-claims-in-the-record-are-derived-never-hand-authored
status: accepted
date: 2026-09-23
supersedes: null
superseded_by: null
related_intents: [itd-147]
related_rfcs: []
related_adrs: [adr-40]
---

# ADR-2609231028044006: Shape claims in the record are derived, never hand-authored

## Context

A brief surface chapter mixed two kinds of content, and they fail in different
ways. **Shape** is the flags and sub-verbs a verb registers. It can be derived
from the command tree and checked against it. **Intent** is why the surface
exists, what it refuses and which trade was made. It cannot be derived, and it
cannot drift against code, because it makes no claim about code shape.

The record kept a second, hand-maintained copy of the shape. It drifted
steadily. A full-tier brief-to-surface crosscheck at the 0.6.2 release gate
returned 147 discrepancies across 23 chapters (iss-2608231346137587). On the
same day, the same reviewers found none in `docs/reference/cli/commands.md`,
which is generated from the tree and drift-tested. The deterministic gate beside
the chapters, `surface_coverage`, checked that rows existed and passed green
while false prose sat next to them.

The split was confirmed on 2026-08-23: a capability to itd-147, this trust rule
to an ADR and a brief invariant, no new stance (`one-canonical-primitive`
already says it), and the plumbing to the brief. itd-147's planning interview
on 2026-09-01 settled the seam.

## Decision

We will derive every shape claim the brief's surface chapters (the chapters
under `.abcd/development/brief/04-surfaces/`) make about the shipped command
surface, and never write one by hand. The enforcement below holds those chapters
and nothing else: a flag or sub-verb spelt elsewhere in the record — another
brief section, an intent, a spec, a plugin command page — is not checked by it.
`docs/reference/cli/commands.md` is generated and drift-tested on its own.

- Each chapter under `.abcd/development/brief/04-surfaces/` ends with one
  generated appendix between two marker comments. The appendix lists each of the
  chapter's commands with its sub-verbs and its own flags. It is composed from
  the same walk of the command tree that builds the compatibility snapshot, and
  `go generate ./internal/surface/cli` writes both.
- A chapter whose command the tree does not register carries the same markers
  around one sentence saying there is no shipped surface. No chapter lacks the
  block.
- The prose above the opening marker states none of abcd's shape: no flag the
  command tree registers (long or its single-dash shorthand), no sub-verb
  command path written as an invocation (backticked, fenced, or prefixed with
  `abcd ` or `/abcd:`), and no backticked name of one of the chapter's own
  sub-verbs. Another program's flag (git's `--force`) and the same words as
  plain English ("the intent plan") are prose, not shape. The chapter's `## Sub-verbs`
  table and its standard note are the one exception, because `surface_coverage`
  checks them against the snapshot.
- The appendix carries flags and sub-verbs only. Exit codes and output fields
  stay prose until the binary records them somewhere a generator can read.
- `surface_coverage` stays as it is and labels every finding as the row-level
  presence check over the surfaces index. It claims nothing about chapter
  prose.

Two tests in `internal/surface/cli` enforce the rule, and both run under
`go test ./...`, which puts them in `make preflight` and in CI:
`TestSurfaceAppendicesMatchCommandTree` and
`TestSurfaceChapterProseStatesNoShape`.

## Alternatives Considered

- **A generated block per section of a chapter.** This keeps the most rationale
  next to the shape it explains, but it needs the most machinery. Rejected at
  the planning interview in favour of one appendix per chapter.
- **Generating the whole chapter.** This loses the rationale, which is the half
  worth a human and the half that cannot drift. Rejected.
- **A gate that makes a surface change touch its chapter.** This forces an edit
  without forcing the edit to be correct. That is a phantom gate, the class of
  defect this decision exists to remove. Rejected.
- **Fixing the measured findings by hand.** This destroys the evidence and
  produces a brief that drifts again at the measured rate. Rejected.
- **One generated appendix per chapter, with the prose held free of shape by a
  check.** Chosen. It copies the one mechanism already shown to work here, the
  generated and drift-tested CLI reference.

## Consequences

- A flag or sub-verb added, removed or renamed without regenerating fails the
  build, and the failure names the chapter and the line.
- A lane that adds a surface adds its register row and its chapter, ends the
  chapter with the two markers and runs the generator. A chapter with no row,
  a row naming a missing chapter, or a chapter without its markers is refused
  by name and fails the build; every other chapter is still regenerated and
  checked, so one unfinished chapter never hides the drift of the rest.
- Prose refers to a capability in plain words ("the cut", "the probe") and never
  spells how it is invoked. The appendix and the CLI reference hold the
  spelling.
- Behavioural claims — ordering, failure semantics, refusals, exit codes, output
  fields — stay hand-maintained and stay a review-grain claim. This decision
  makes no hand-written behavioural claim safe.
- A claim about external state is a third category that is neither derivable
  nor rationale. It is out of scope here and held by iss-2608231607594913.
