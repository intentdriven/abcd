---
schema_version: 1
id: "iss-2609091801085579"
slug: "the-docs-release-gate-flag-is-armed-by-nothing"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "release-gate"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/lint.go"
---

The documentation lint carries a release-gate flag that promotes an overdue citation from a warning to a blocker, and nothing runs it. The release workflow, the continuous-integration workflow, the make target and the template scaffolded into managed repositories all invoke the bare lint, and the launch verb computes its own citation preflight rather than calling the flag, so the stricter mode exists only for a person who types it. A blocker that only a human can trigger is not a gate, which is the shape the enforcement-claims-are-facts principle refuses: the flag reads as a release control and controls nothing. The cost is a citation that has aged past its threshold reaching a release with a warning nobody had to answer, while the record implies the release could not have carried it. Fix direction: either arm the flag where a release is actually gated, so the promotion happens on the path that matters, or retire it and let the launch verb's own preflight be the single citation control, saying so where the flag was documented. Detector: a release cut carrying a citation older than the threshold refuses on the path a release actually takes, or the flag does not exist.
