---
schema_version: 1
id: "iss-2610292103166339"
slug: "owed-is-inferred-from-refusal-wording"
severity: "minor"
category: "drift"
source: "impl-review"
found_during: "fidelity review rcp-6083de2cef52 of itd-2610290746077716"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/remote_baseline.go"
remedy: "Decide owed from the owner's plan and the repository's visibility, read through the plan lookup abcd already makes, and keep the refusal wording only as a fallback when the plan cannot be read, with a test of a plan refusal worded without upgrade"
---

abcd ahoy --remote decides that a setting is owed by the plan from the repository being private or internal and from GitHub's refusal text containing the word upgrade, not from the owner's plan. The fidelity review of itd-2610290746077716 (rcp-6083de2cef52) found its scope condition cond-2610290749118129 (a setting the plan or the visibility does not offer is reported as owed) narrowed: a refusal GitHub words any other way reads as unknown, though never as a failure (internal/core/ahoy/remote_baseline.go:452, remote_baseline.go:236). iss-2610161632245527, which asks abcd to probe what the forge allows before picking a route, is the wider form of this.
