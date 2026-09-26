---
id: adr-41
slug: corpus-trust-boundary
status: proposed
date: 2026-08-16
supersedes: null
superseded_by: null
related_intents: [itd-76, itd-126, itd-127, itd-74]
related_rfcs: []
related_adrs: [adr-30]
---

# ADR-41: Documents and ledgers never leave the user tier; a public citation requires both gates

## Context

The sources corpus ([itd-76](../../intents/shipped/itd-76-source-provenance-ledger.md))
lets an agent consult material the user is not free to name in public. Anything
that makes consultation safe rests on two boundaries holding mechanically,
regardless of which surface — personal verbs, team share/ingest
([itd-126](../../intents/drafts/itd-126-a-team-shares-one-bibliography-without-sharing-anyone-s-corp.md)),
or paper reconstruction
([itd-127](../../intents/drafts/itd-127-a-paper-is-reconstructed-from-the-provenance-ledger-claims-g.md)) —
is doing the moving.

## Decision

1. **Documents and influence ledgers never leave the user tier.** What crosses
   into a repo — via share, render, or any future surface — is *citation
   data* only. There is no flag, override, or convenience path that commits a
   corpus document or a ledger file into a repository.
2. **A public citation requires both gates, independently.** The source's
   `permission_status` grants the *right* to cite; the ledger line's
   `cited_publicly` flag — flipped only by the human — *exercises* it. Either
   gate failing blocks the citation, mechanically, at every rendering or
   sharing surface.

## Consequences

- Confidential material is protected by construction, not by reviewer
  vigilance: the share surface refuses `confidential: true` entries, and a
  render fails structurally on an unpermitted key.
- The rule is enforceable per-surface and testable per-surface; each consuming
  intent cites this ADR rather than restating the rule, so there is exactly
  one place to strengthen or amend it.
- The boundary covers literal movement and literal strings only. Identifying
  *paraphrase* is out of mechanical reach and stays a behavioural rule plus
  human publish review — stated plainly in itd-76's "What It Cannot Enforce".
