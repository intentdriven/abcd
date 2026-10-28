---
id: adr-2610141646523373
slug: ingested-pages-are-private-by-default
status: accepted
date: 2026-10-14
supersedes: null
superseded_by: null
related_intents: [itd-2610141646467500, itd-36]
related_rfcs: []
related_adrs: [adr-41, adr-2610091918443997, adr-2610142116480836]
---

# ADR-2610141646523373: Ingested pages are private by default and reach the project only by a person's promotion

## Context

itd-36 shipped `library ingest` writing its distilled pages into the project's
`.abcd/memory/`, committed and shared: the point was knowledge that compounds
for a team. Its safeguards concern copyright (no originals by default, bounded
quotation), not confidentiality, and ingest consults no confidentiality label.
Later records went the other way: the sources corpus separates confidential
sources (adr-41, still proposed; brief invariant 9 carries its force), memory
notes stay on the machine until a person promotes them (adr-2610091918443997;
its promotion step is designed in the dashboard draft itd-2610111655362931, not
shipped), and the product thinker ruled on 2026-10-14 that ideate's verdicts are
private or public by the person's choice (adr-2610142116480836).

## Decision

Pages that `library ingest` distils are written to a machine store by default,
`~/.abcd.noindex/library/<root-sha>/pages/`, owner-only, and reach
`.abcd/memory/` only when a person promotes them with `library promote`, after
the redactor and the banned-names check pass; an agent never promotes. This
reverses itd-36's shared default (itd-2610141646467500 carries the typed
reversal). Ruled by the product thinker in the planning interview of
2026-10-14.

## Alternatives Considered

- **Private by default, a person promotes** (chosen): matches memory notes and
  ideate's private-first rule; teammates see a page only once promoted.
- **Follow the source's label in the corpus** (confidential or unlabelled stays
  private, public is shared): reuses an existing label, but a wrong label shares
  too much or too little without anyone deciding.
- **Ask on every ingest:** never guesses, but adds a question to every ingest.
- **Keep the shared default:** no change, and confidential material keeps being
  committed on first read.

## Consequences

- Breaking: anyone relying on ingested pages appearing for teammates must now
  promote them.
- Pages already committed stay where they are; nothing is moved retroactively.
- Promotion needs a person's act. The binary cannot prove one; it does what the
  corpus ledger's flip does: the verb asks at a terminal, and a guard entry
  refuses it from an agent's shell. The memory notes' promotion is planned the
  same way.
- The rule that a promotion in a repository with more than two maintainers needs
  another maintainer's agreement (adr-2610091918443997) is not carried over:
  an ingested page is the person's own reading, not a lesson that steers other
  agents.
