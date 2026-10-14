---
id: adr-2610141646523373
slug: ingested-pages-are-private-by-default
status: accepted
date: 2026-10-14
supersedes: null
superseded_by: null
related_intents: [itd-2610141646467500, itd-36]
related_rfcs: []
related_adrs: []
---

# ADR-2610141646523373: Ingested pages are private by default and reach the project only by a person's promotion

## Context

itd-36 shipped `library ingest` writing its distilled pages into the project's
`.abcd/memory/`, committed and shared: the point was knowledge that compounds
for a team. Its safeguards concern copyright (no originals by default, bounded
quotation), not confidentiality, and ingest consults no confidentiality label.
Later records went the other way: the sources corpus separates confidential
sources (adr-41), memory notes stay on the machine until a person promotes them
(adr-2610091918443997), and ideate's verdicts are private or public by the
person's choice.

## Decision

Pages that `library ingest` distils are written to a machine store by default
and reach `.abcd/memory/` only when a person promotes them; an agent never
promotes. This reverses itd-36's shared default. Ruled by the product thinker in
the planning interview of 2026-10-14 (session abcd-66).

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
- Promotion needs a person's act the binary can tell from an agent's, as the
  memory notes' promotion and the corpus ledger's flip already require.
