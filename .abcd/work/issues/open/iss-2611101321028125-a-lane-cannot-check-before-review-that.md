---
schema_version: 1
id: "iss-2611101321028125"
slug: "a-lane-cannot-check-before-review-that"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/docfidelity/docfidelity.go"
remedy: "Waits on a ruling first: then let a lane run the doc-fidelity review over its own diff before validation, or require each new behaviour claim in its docs to name the test that shows it."
---

A lane cannot check before review that the docs claims it writes are true of the code: precheck and lint docs pass prose that is false, and the doc-fidelity review runs only at a shipping spec close and at the release cut. On 10 Nov at 04:09 a lane wrote docs claims for two deadline routes but tested only one, and only the review pair caught it. A person must choose whether a lane runs a per-change doc-fidelity check, or a brief rule asks each new behaviour claim to name the test that shows it.
