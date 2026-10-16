---
id: dcn-2610161040382067
slug: a-stuck-drain-lane-hands-itself-back
date: 2026-10-16
---
2026-10-16 — A stuck drain lane hands itself back (facilitator-on-delegation, drain-many-accounts): a drain lane whose step has refused identically three times, or has refused at one stage for sixty minutes, is handed back as stuck and its reservation released; refusals that are waits by design, the preflight-receipt wait and an outage wait, never count; both bounds are configurable in the pace's layers; a refusal that cannot change (the issue finished at the base or on the default branch, or the hold now another's) hands back at once; and implement step --discard accepts a lane handed back as stuck (grounds: dead-letter redrive after a maximum receive count, as SQS maxReceiveCount; iss-2610161015368630).
