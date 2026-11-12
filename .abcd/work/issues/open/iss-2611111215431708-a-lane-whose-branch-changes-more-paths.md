---
schema_version: 1
id: "iss-2611111215431708"
slug: "a-lane-whose-branch-changes-more-paths"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run, review of the loop-side widening"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/widen.go"
remedy: "Design directory-level holds for a large lane deliberately: lift only the most populous sibling group, one level at a time, with a floor on depth and never to a directory that contains a gate path, keep the lifted hold through the landing's narrowing, and measure the receipt's footprint together with the record paths the landing adds so a lane that cannot land is refused at its receipt; until then, hand such a lane back at its first refusal with the cap named instead of after three launches."
---

A lane whose branch changes more paths than one reservation message can carry cannot be held, and is handed back as stuck. The loop reserves each changed path by name; past the reservation's size cap (about fifty paths of ordinary length) the receipt is refused naming the cap, which no relaunch can change, so after three launches the lane goes back as stuck. In a band of a few paths just under the cap the receipt passes and the lane fails later, at the push fence, when the landing's two record paths no longer fit, after its validators have passed. Holding a directory in place of its files would lift the limit but was tried and withdrawn in review: lifting every path lets a small change hold whole trees, and the landing's narrowing undoes it.
