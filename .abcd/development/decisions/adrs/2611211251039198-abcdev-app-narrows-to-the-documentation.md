---
id: adr-2611211251039198
slug: abcdev-app-narrows-to-the-documentation
status: accepted
date: 2026-11-21
supersedes: null
superseded_by: null
related_intents: [itd-2610142145236613]
related_rfcs: []
related_adrs: [adr-2611210848450372, adr-48, adr-2610142147242544]
---

# ADR-2611211251039198: abcdev.app narrows to the documentation without forwarding the record's old addresses

## Context

[adr-2611210848450372](2611210848450372-abcdev-app-s-deploy-narrows-to-the.md) amends adr-48 so that abcdev.app's deploy narrows to the documentation, behind two switches in `.abcd/site.json`'s `record.elsewhere`. Its point 2 forwards the record's old addresses on abcdev.app to the same path at the record's Pages address once `forward` is on. Its point 3 turns `forward` on before `narrow`, so no old address is left without an answer.

At the cut-over the person ruled on what abcdev.app does with the record ([dcn-2611211021237477](../../../work/decisions/dcn-2611211021237477-pages-cut-over-product-thinker-s-ruling.md)). The Pages site stays published. abcdev.app narrows to the documentation only. The record's old addresses on abcdev.app are not forwarded to the Pages site. [dcn-2611211223308175](../../../work/decisions/dcn-2611211223308175-step-4-of-spc-2610142146250176-names.md) records what that ruling sets aside: the forwarding act of spc-2610142146250176 step 4, the order point 3 gives the switches, and the acceptance criterion of itd-2610142145236613 that a record link shared before the move lands on the same page at its new address.

adr-2611210848450372 is accepted, and an accepted record is not edited, so this record carries the change.

## Decision

This record amends adr-2611210848450372. It replaces that record's points 2 and 3; its points 1 and 4 stand.

1. **The record's old addresses on abcdev.app are not forwarded** (replaces point 2). `record.elsewhere.forward` stays off. Once a release deploys the narrowed build, abcdev.app serves nothing at `/record`, `/record/`, any page below it or `/record.json`, and sends none of them to the Pages site. The documentation's own 301s for the paths moved under `/docs/` stay. The forwarding code stays in `abcd site build`, so a later decision can still turn the switch on.
2. **`narrow` is turned on without `forward`** (replaces point 3). The narrowing is one committed change of its own, on the person's word, and no forward goes before it. With `narrow` on, abcdev.app's build also ships no `record.js` and no `_headers` block for a record path, since it serves none of the record.

Read as one decision, the chain says: abcdev.app's deploy builds the documentation only (adr-2611210848450372 point 1), the record's old addresses are not forwarded (point 1 here), and both sites deploy on release from released bytes (adr-2611210848450372 point 4, adr-48).

## Alternatives Considered

- **Keep adr-2611210848450372's points 2 and 3: forward, then narrow.** Every link to the record shared before the move would still land on its page. The person ruled against forwarding, so this is not taken.
- **Leave the record on abcdev.app as well as on the Pages site.** No old address would break. But the record would be published twice, from two builds that can disagree, and abcdev.app would not narrow to the documentation as ruled.
- **Narrow without forwarding, and record the change in an amending record (chosen).** abcdev.app carries only the documentation, the Pages site carries the record, and the written record says what the earlier one asked for that is no longer done.

## Consequences

- A link to abcdev.app's record shared before the move answers with the host's not-found response once a release deploys the narrowed build. It does not point to the Pages site. The criterion of itd-2610142145236613 that asks for the same page at the new address is not met by this cut-over.
- abcd's record is reached at its Pages address, through the links abcdev.app's documentation draws there.
- Turning `forward` on later needs a decision of its own that amends this record, and a committed change that flips the switch.
- adr-2611210848450372 stays accepted and is not superseded. Its points 1 and 4 still govern what abcdev.app builds and when both sites deploy.
