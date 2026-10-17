---
id: prn-a-model-run-is-re-runnable-not-reproducible
claim_type: criterion
reference: "the model-matrix run"
comparison: "A model run repeated from the same pinned request, compared against the claim that it returns identical output."
evidence: null
---

# A model run is re-runnable, not reproducible

**The rule.** A record that compares models is called reproducible only in the sense that
the run can be repeated from its manifest: everything that can be pinned is pinned and
recorded, both as asked and as served; everything that cannot be pinned is named as
unpinned or self-reported; and a second run reports where it drifted from the first. It
is never claimed to return the same output, and a comparison between models states how
many repeats it rests on, why that many, and the uncertainty that remains.

**Why.** Identical output is not on offer. On hosted models, temperature zero with a
fixed seed still varies, because serving kernels are not invariant to how many other
requests share a batch; providers call their own seeds best effort. Infrastructure alone
has moved an agent benchmark by six points. And spread inside one model-and-harness
pairing has been measured larger than the spread between pairings: three-run comparisons
ranked the weaker pairing first in up to 44 percent of resamples, and separating the
gaps observed needed tens to over a hundred runs per arm. The sources and their evidence
tiers are in the
[model-matrix SOTA note](../research/notes/2026-10-16-model-matrix-sota.md).

A record that promises identical output is a false green: the first rerun that differs
reads as a defect when it is the floor. A record that pins nothing makes every
difference uninterpretable, the case itd-166 states: two runs under one model identifier
can differ in every setting that shapes behaviour. Between the two sits the honest claim.

**Bounds.**

- **Forward only.** It binds comparisons between models written after the run manifest
  exists (itd-166 defines it). Records written earlier, and rulings resting on them, are
  not reopened by it; the 2026-10-14 reviewer-pair proposal (iss-2610140652230444)
  already asks for measurement before it hardens.
- **What cannot be pinned is said, not hidden.** A host-run model reports only what the
  host exposes: abcd cannot set its sampling or read its served version, and the host
  harness's version moves on its own. A retired provider endpoint ends a re-run. Each is
  marked in the manifest, and the report says so.
- **The repeat count is sized, not picked.** A comparison declares the smallest
  difference that would change a decision and sizes its repeats by a power estimate from
  a pilot; one that runs fewer says it can only detect larger differences.
- A local server at batch size one may come closer to identical output; that is a
  measured property of one setup, never assumed.

**Ladder.** Principle: this file (2026-10-16). Enabling convention: the run manifest
itd-166 is to define (absent). Tool: the model-matrix run of itd-2610161733316518
(absent), which reports the repeat count, its power estimate and the unpinned fields
with every comparison.
