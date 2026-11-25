---
id: adr-2611251808065076
slug: abcd-asks-for-its-newest-release-on-a
status: accepted
date: 2026-11-25
supersedes: null
superseded_by: null
related_intents: [itd-130, itd-111, itd-2611062149308723]
related_rfcs: []
related_adrs: [adr-38, adr-2611062149324491, adr-2610050711287503]
---

# ADR-2611251808065076: abcd asks for its newest release on a standing yes

## Context

adr-38 says implicit operations never touch the network and that automatic update discovery needs superseding in the open. The product thinker said on 2026-11-25 that it is not fixed: it was made with security in mind, and may go if the security questions are answered. Keeping every copy of abcd current by hand had left a session on a copy five releases old (iss-2611251739422396, iss-2611251742484774). The question was judged as an idea, and reframed (dcn-2611251807556458). The record gives adr-38's reasons as privacy, consent, a bound on what may arrive, and testability; the security reasons stand beside it, in the neighbouring rules.

## Decision

Accepted by the product thinker on 2026-11-25 (dcn-2611251823160636). This refines adr-38 in the open, for this one request: its first and second tiers stand for every other operation of abcd's, and its third is unchanged. No code makes the request yet; until some does, abcd asks the network nothing on its own.

1. **A check, on a standing yes.** The person gives it on their own machine and can withdraw it; it is off until then. abcd asks at most daily, reads release metadata alone, and tells the person alone. It installs, stages, and executes nothing, runs in no per-command hook, sends no version or identifier, puts nothing in the model's context, and never reports current when it failed.
2. **Applying is a second, separate yes**, and stays the person's act until it is ruled on by itself. Outside a drive (adr-2611062149324491) nothing applies a release unasked.
3. **No repository file, environment variable, or agent gives either yes**, and abcd never switches the host's own auto-update.

## Alternatives Considered

- **A check on a standing yes, apply by hand (chosen).** Every threat of a check is answered, or reduced to a wrong notice.
- **Keep adr-38 as it is.** The copies can still be compared from disk with no request; a person on one copy who never asks never learns of a release.
- **A check on by default, with an off switch.** Rejected by the verdict: disclosure is part of any check.
- **Check and apply on one yes.** Rejected by the verdict: applying's threats are reduced and not removed. A bad change built by the genuine workflow, a release that judges wrongly, and one that disables its own successor stay.

## Consequences

- The disk comparison of every copy needs none of this and comes first.
- Brief invariant 7 names the exception with this acceptance; the update surface's chapter, the update command page, and the two tests that hold the network out change with the code that makes the request, in one change.
- Two requests already made from a hook, the bootstrap's for a fresh plugin root (itd-132) and the opted-in dogfood build's, are named or removed in that same change as the code.
- A yes kept on the account can be written by anything running as it, which is why it covers a notice and no more.
