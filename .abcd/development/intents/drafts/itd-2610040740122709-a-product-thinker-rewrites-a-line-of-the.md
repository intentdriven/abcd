---
id: itd-2610040740122709
slug: a-product-thinker-rewrites-a-line-of-the
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610032150577708]
related_adrs: [adr-2610032150581128]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Acting from the dashboard: rewriting the brief and approving intents

## Press Release

> A product thinker rewrites a line of the brief and approves an intent from abcd's dashboard, and the technical facilitator reviews each change before it enters the record.

_Filed by the facilitator on 2026-10-04 from the product thinker's words at the dashboard interview (itd-2610032150577708 decision 8): "Later additions: Rewrite the brief; approve intents"._

## Why This Matters

_Facilitator-written._

The first version lets the product thinker read the brief, mark it and confirm it, with every touch earmarked for the technical facilitator. Rewriting a line and approving an intent are the next steps: acting on the record rather than flagging it. Both are writes, so they travel only over the encrypted way in, and each reaches the facilitator's review before it enters the record, as decision 17 there requires of everything the product thinker touches.

Typed links: builds on itd-2610032150577708.

## Decisions

1. 2026-10-04, the product thinker (decision 8 there): rewriting the brief and approving intents are later additions.
2. 2026-10-04, the product thinker (decision 17 there): everything the product thinker touches is earmarked for the facilitator's review in their console.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [dashboard-sites SOTA note](../../research/notes/2026-10-23-dashboard-sites-sota.md).

- **The forge's suggested changes are the closest pattern:** anyone may suggest, someone with write access applies, one or a batch becomes "a single commit", on which each suggester "will be a co-author" and the applier is "a co-author and the committer" (GitHub docs). Authority with the applier, credit to the proposer.
- **Documents do the same:** in suggesting mode "you can suggest changes to a document without changing the original text" and the owner "can decide whether to keep them" (Google Docs help).
- **The kernel's tags separate proposed, agreed and checked:** `Suggested-by:` names the idea's source, `Acked-by:` comes from "those responsible for or involved with the affected code" and "does not necessarily indicate acknowledgement of the entire patch", `Reviewed-by:` means reviewed and found acceptable (kernel submitting-patches); Git treats any `Key: value` block at the end of a message as trailers (git-interpret-trailers). A rewrite is a suggestion; an approval is an acknowledgement, not a review.
- **Writes stay on the encrypted way in:** Go's cross-origin layer admits header-less clients on any network (net/http), so the gate and the session do the work everywhere, and only the tailnet adds an encrypted transport and a peer the daemon names; acting is tailnet-only, as this draft says, for that reason and not for the cross-origin layer's.

**Path 2 — native, shaped to the forge's suggested-changes flow: a rewrite travels as a suggestion the facilitator applies, with the product thinker credited on the resulting change; an approval is a recorded acknowledgement the facilitator completes, never the planning sign-off itself.** This answers both open questions in the direction decision 2 points. The independent fit-challenge runs at plan time.

## Open Questions

- What approving an intent from the dashboard does: the planning sign-off itself, or a request the facilitator completes.
- Whether a rewrite is proposed as text the facilitator merges, or written directly and reviewed after.
- (SOTA 2026-10-23) How the product thinker is credited on a change made from their suggestion (a trailer on a commit a human authors), within the attribution gate's rules on who may be named.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
