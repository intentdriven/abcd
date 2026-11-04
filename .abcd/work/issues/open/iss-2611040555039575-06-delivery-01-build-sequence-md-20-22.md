---
schema_version: 1
id: "iss-2611040555039575"
slug: "06-delivery-01-build-sequence-md-20-22"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md"
remedy: "Rewrite the sentence as the target shape, stating that a seam stands only once a milestone consumes it."
---

06-delivery/01-build-sequence.md:20-22 says every capability after the first milestone is a native default behind an already-wired interface with an optional external adapter beside it. The same chapter says at :33 that the scanner is the one capability seam standing and at :41-47 that the oracle, history, spec and run seams are planned, with history and spec shipping as core packages with no adapter interface behind them. internal/adapter has gitleaks, hosting, openaiapi, repoprompt, scanner and tailscale only.
