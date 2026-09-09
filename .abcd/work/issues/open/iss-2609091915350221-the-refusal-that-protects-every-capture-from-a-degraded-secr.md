---
schema_version: 1
id: "iss-2609091915350221"
slug: "the-refusal-that-protects-every-capture-from-a-degraded-secr"
severity: "major"
category: "tech-debt"
source: "agent-finding"
found_during: "fidelity audit of the capture intent"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/history.go"
---

The refusal that protects every capture from a degraded secret scanner is asserted by no test on any path, and the spec claimed the opposite. The guard is real and correctly placed: capture refuses outright when the scanner reports itself unavailable, rather than storing a transcript under weakened redaction. But a search across the history package's tests for that refusal, or for the scanner's own unavailability signal, returns nothing, so nothing would notice if the guard were removed, reordered behind the write, or made conditional. The spec asserted that the tests exercise it on the sub-agent path specifically rather than inferring it; the four tests it named cover lineage redaction, a surviving blocking span and a malformed scalar, and not one of them degrades a scanner. The record has been corrected to say what is true. The guard also predates this work, so this delivery inherits it rather than establishing it, which is why it was never given a detector of its own. That is the whole argument for arming it now: an unasserted guard on a fail-closed path is indistinguishable from an absent one until the day it matters, and this repository refuses to trust an unarmed detector everywhere else.
