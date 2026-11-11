---
schema_version: 1
id: "iss-2611081138442219"
slug: "in-a-model-matrix-cell-a-copy-whose"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "run-2611080653222827 lane-1 validate round 3 (ruthless-reviewer note)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner"
remedy: "The cell's outage reading counts a 429 error event (any adapter) as not started, retried on the same pin within the copy's try bound, with a test through the opencode fake writing a 429 error event"
resolution: "A provider's 429 with nothing spent now reads as the provider down, so a throttled copy is retried on its pin and ends not started, never scored; a 429 after a costed step stays the cell's failure."
impact: internal
resolved_by:
  intent: "itd-2610161733316518"
  spec: "spc-2610161949559050"
---

In a model-matrix cell, a copy whose provider answers with HTTP 429 is scored as the model's failure: the runner's outage reading counts only a 5xx and its own retries as the provider down, so a throttled OpenRouter call through opencode reads as the model failing, against dcn-2611081138386461.
