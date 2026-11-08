---
id: dcn-2611081138386461
slug: in-a-model-matrix-cell-a-provider-s
date: 2026-11-08
---
2026-11-08 — In a model-matrix cell, a provider's throttling answer (HTTP 429, through opencode or any adapter) counts as the copy not started, as an outage does: it is retried on the same pin within the copy's try bound and never scored as the model's failure. Grounds: throttling measures the provider's capacity at that moment, not what the model can do, and scoring it would rank models by their provider's load. Facilitator decision on delegation during the away window; reverse by recording a decision that a throttled copy is scored, and dropping the 429 case from the outage reading.
