---
id: itd-99
slug: a-team-of-product-thinkers-decides-as
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
---

# A Team of Product Thinkers Decides as One

## Press Release

> _Facilitator-seeded draft (2026-07-25 role-ownership review) — the product
> thinkers own the final press-release framing._

> **The product thinker can be a team.** Each thinker takes their own
> thinking session — the same interview a solo thinker gets, wherever they
> work, on their own time — and abcd collates the authored inputs into one
> joint proposal: agreements merged with citations, disagreements surfaced
> side-by-side with their authors' names, nothing averaged into mush. The
> team then holds one collective discussion over that proposal and records
> the decision. Throughout, abcd never asks a product thinker to make a
> technical decision — only to decide on a clear proposal that names its
> options, its recommendation, and its price.
>
> "Alice, Bob and I each wrote our version of the why separately," said
> Carol. "The joint proposal showed where the three of us actually disagreed
> — two places, not the ten we expected. The discussion took an hour, the
> decision is on record with all three names on it, and my dissent about
> scope is preserved next to the adoption instead of papered over."

## Why This Matters

The governing assumption is already doctrine: a verdict is a proposal and
the human's adoption is the gate (verifier-selects-gates-decide); itd-41
phases roadmaps as decide-knowing-the-price proposals; itd-90 hands a
product thinker an interview whose answers land authored-by a named person
on a date. But everything shipped and recorded assumes **one** product
thinker. Three things are missing, in dependency order:

1. **A participant model** — abcd has no representation of multiple named
   humans (personas.json holds the fictional Alice/Bob/Carol; attribution
   knows one maintainer).
2. **Person-keyed thinking sessions** — the itd-90 interview pattern,
   extended so each thinker's input is captured separately and stamped as
   theirs.
3. **Collation and decision machinery** — the press-release-composer already
   composes one grounded document from many sources under cite-or-be-dropped,
   and itd-81 supplies the independence discipline a human panel equally
   needs; neither is wired to human inputs.

## What's In Scope

- **The participant model**: named product thinkers with authorship
  stamping; every team-workflow artefact records who wrote what, when.
- **Individual thinking sessions**: per-person interviews (itd-90 pattern)
  captured independently — thinkers do not see each other's answers before
  writing their own (independence before aggregation, per itd-81's
  rationale).
- **Automatic collation into a joint proposal**: cite-or-be-dropped against
  the authored inputs; agreements merged with citations; disagreements
  listed as first-class items with their authors; uncited synthesis dropped.
- **The proposal contract for teams**: any technical choice in the joint
  proposal is framed as options + recommendation + stated costs — never a
  question requiring technical design from a thinker.
- **The decision record**: the collective discussion's outcome lands with
  named adopters, the date, and dissent preserved verbatim beside the
  adoption.

## What's Out of Scope

- **Real-time meeting/chat tooling** — the discussion happens wherever the
  team talks; abcd records its outcome, not its transcript.
- **Voting and quorum arithmetic** in v1 — a decision is a recorded
  adoption, not a vote count.
- **Agent-to-agent coordination** — itd-33 owns agents; this intent is about
  humans.
- **Facilitator staffing** — works identically under duo or solo (itd-97).

## Acceptance Criteria

- **Given** three thinkers each complete an individual session, **when**
  collation runs, **then** exactly one joint proposal lands in which every
  claim cites the authored input(s) it came from and every disagreement is
  listed with its authors — uncited synthesis is dropped, not kept.
- **Given** a joint proposal contains a technical choice, **when** it is
  presented to the team, **then** it appears as options with a
  recommendation and stated costs, never as an open technical question.
- **Given** the collective discussion concludes, **when** the decision is
  recorded, **then** it names the adopters and the date and preserves any
  recorded dissent verbatim alongside the adoption.
- **Given** a named participant's session is missing, **when** collation is
  attempted, **then** the gap is loud (the missing participant is named) —
  never a silent smaller-team proposal.
- **Given** an individual session's content, **when** it appears anywhere
  downstream (proposal, brief, decision), **then** it is stamped as authored
  by that person on that date, never disguised as something extracted from
  the repository (itd-90 symmetry).

## Open Questions

- **Where do participants live, and the privacy collision**: the PII
  discipline forbids committing real names/emails, yet authorship stamping
  wants named humans in the record. A deliberate carve-out (participants
  opt in to being named in their own repo) or an indirection (participant
  ids mapped in an uncommitted local file)? This must be settled before any
  schema exists.
- **Async vs. sync** — is the collective discussion an artefact the team
  edits asynchronously, or a session whose outcome one person records?
- **Dissent and the lifecycle** — can a dissenting thinker block planning,
  or is adoption by the recorded decision rule always sufficient?
- **Collation engine home** — transpose the press-release-composer agent, or
  a new dedicated collation agent? (One-canonical-primitive says extend.)

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- itd-90 (brief-interview-for-the-blanks) — the authored-answer,
  role-separated interview pattern this extends from one thinker to many.
- itd-41 (phase negotiator) — decide-knowing-the-price proposals.
- `.abcd/development/principles/verifier-selects-gates-decide.md` — the
  decision contract, unchanged for teams: the collated proposal is the
  verdict; the team's adoption is the gate.
- `.abcd/development/intents/disciplines/itd-81-judge-calibration.md` —
  independence-before-aggregation, transposed from LLM judges to human
  panels.
- itd-97, itd-98 — the sibling drafts filed with this one.
