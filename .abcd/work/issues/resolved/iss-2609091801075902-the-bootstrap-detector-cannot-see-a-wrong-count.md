---
schema_version: 1
id: "iss-2609091801075902"
slug: "the-bootstrap-detector-cannot-see-a-wrong-count"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "release-gate"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/hooks_selfprovision_test.go"
resolution: "The self-provision detector derives the wired, salvaging and throttled event sets from hooks.json, requires each brief paragraph to name exactly the wired set, checks every count claim against its derived set and fails on an underived count; a synthetic paragraph proves a wrong count and an invented event both fail."
impact: internal
resolved_by:
  commit: "c6231b33295ef6783d66be02ed24ebaf4316b55a"
---

The detector that holds the brief's account of hook self-provisioning to the shipped configuration checks that each salvaging event is named somewhere in the paragraph, which is a weaker property than the one it exists to enforce. The brief said three events self-provision where the configuration wires four, and the detector passed, because the fourth event was mentioned in that paragraph for an unrelated reason and the presence test could not tell a mention from a claim. A count is exactly the kind of statement that goes stale between a change and its record, and it is the class the release cross-check found most of, so a guard over this paragraph that cannot read a number is guarding the wrong property. The failure is quiet in the direction that matters: the guard reports the contract met while the prose misstates the shipped set, and the next reader trusts the guard rather than the sentence. Fix direction: assert the set rather than the mentions, deriving the salvaging events from the shipped configuration and requiring the paragraph to name that set and no other, so both a missing event and an invented one fail; a count stated in prose should be derived in the test rather than compared as a word. Detector: a paragraph claiming a number of self-provisioning events fails when the configuration wires a different number, and fails when it names an event the configuration does not wire, on a tree where the prose and the configuration disagree in either direction.

## Grounds

- pursued: a brief paragraph that misstates how many hook events self-provision fails go test; a paragraph with a wrong count or an unwired event that the detector passes would show it wrong
