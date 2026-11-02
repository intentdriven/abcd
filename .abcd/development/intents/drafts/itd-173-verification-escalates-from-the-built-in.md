---
id: itd-173
slug: verification-escalates-from-the-built-in
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# Verification escalates from the built-in check to an outside audit, and the product thinker picks the rung

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

How much verification a piece of work deserves is a property of the project, not of the framework. A personal tool where a fault inconveniences only its author needs less than software other people pay for. Fixing the level in the framework gets one of those two wrong.

So verification is a ladder and the product thinker chooses the rung, in the plain terms of what is at stake rather than in the terms of what the machinery does. The rungs run from the built-in check, to two models from the same provider, to models from different providers where more than one is available, to an outside service, to a paid human audit. Evidence supports the shape: a single automated review finds a minority of real problems, and reviews from different model families find different ones, so each rung buys a measurably different thing rather than more of the same.

**Security is the first rung to build out and the least served today.** It is the case where the product thinker most clearly cannot judge what they are being told, where the consequences reach furthest beyond them, and where the framework currently offers no way to escalate at all. A repository can already obtain private security review through its forge, so the early rungs of a security ladder need no new relationship with anyone.

The default stays small. Nothing here obliges a project to climb, and a project that never leaves the bottom rung should not feel broken.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [verification-assurance SOTA note](../../research/notes/2026-10-23-verification-assurance-sota.md).

- **The security standards leave the level to the organisation's risk analysis.** ASVS 5.0 does not prescribe a level: "an organization should analyze its risks and decide what level it believes it should be at"; L1 is the minimum, most applications "should be striving to achieve" L2, L3 is for the highest assurance (ASVS 5.0, What is the ASVS). This is the ruling's shape, stated in terms of what is at stake.
- **A level says how hard anyone looked, never how good the thing is.** Common Criteria's EAL1 to EAL7 run from "Functionally tested" to "Formally verified", and "The EAL level does not measure the security of the system itself, it simply states at what level the system was tested"; higher costs more and takes longer (Evaluation Assurance Level summary).
- **Tracks let one aspect climb without blocking on another.** SLSA v1.0 runs its Build track L0 to L3, each level naming what it defends against, so that "progress made in one aspect of security" is recognised "without blocking on an unrelated aspect" (SLSA levels). Security first, as ruled, is one track; correctness and docs can climb on their own.
- **The claimed rung needs a report that names what was checked and what was not.** ASVS's assessment guidance: state the level attempted and the requirements included, summarise every requirement checked rather than only failures, and "Merely running an automated tool without thorough testing is insufficient" (ASVS 5.0, Assessment and Certification).
- **The first security rung is free on the forge.** Private vulnerability reporting lets anyone report "directly and privately to the maintainers", the reporter becomes a credited collaborator on a draft advisory, a temporary private fork holds the fix, and a CVE can be requested before publication (forge documentation on private reporting and repository advisories). NIST SSDF RV.1.3 asks for a disclosure policy first; Scorecard's Security-Policy check scores one; the OpenSSF Best Practices gold level requires a security review within five years, from any party.
- **What a second model buys is already measured in this repository's notes, and it is not a second vote.** Panels fall short of the best single judge; on code, consensus underperforms a naive baseline; the one measured debiasing effect is re-presenting the work in a fresh session as an artefact of unknown authorship, at no cost; no published number exists for cross-family benefit on code review; the circulated "40 to 60 per cent more issues" figure traces to no study ([cross-model review note](../../research/notes/2026-07-14-cross-model-review.md); [model-matrix note](../../research/notes/2026-10-16-model-matrix-sota.md)).
- **Where research touches the ruling:** the ruling names "two models from one provider" as a rung, two different models from the same vendor. What the evidence says about that rung: distinct models give a second opinion, not a second vote (panels do not beat the best single judge), so its findings are reported beside the first rung's and never merged by vote; and the gain of the different-provider rung over it is unmeasured. A fresh-session rerun of the same model, the one measured debiasing effect, is a cheaper rung below it that the ruling does not list; whether to add it is the product thinker's call. The body's "measurably different thing" has no published support (no cross-family number exists for code review) and stands as the product thinker's stance, not as evidence. The ruling stands as the ladder's shape; the plan promises nothing for the third rung until a run here has measured it.

**Path 2: a native ladder shaped to ASVS's risk-based choice and SLSA's per-track progression, with Path 1 for the security floor (the forge's private reporting and a security policy).** The product thinker is asked in the terms of who is harmed and how far it reaches, never in model counts; the rung chosen per track is recorded as a claim with the ASVS report shape (attempted, checked, not checked), and that one record shape is the seam an outside service or paid audit later writes into. No dependency. The independent fit-challenge runs at plan time.

## Open Questions

**Decided 2026-08-29 (product thinker).** The ladder is the answer, not a single fixed level: built-in check by default, two models from one provider where a subscription allows it, different providers where more than one is available, an outside dependency for higher-value work, and a paid audit where the stakes justify it. Security is where the ladder gets built first.

- (SOTA 2026-10-23) Is the ladder one ladder or one per track (security, correctness, docs), given SLSA's finding that tracks let a project climb one aspect without blocking on another?
- (SOTA 2026-10-23) What does the record of a rung hold, so that a forge advisory, an independent session's findings and a paid audit's report all land in the same shape?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
