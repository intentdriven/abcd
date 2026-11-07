---
schema_version: 1
id: "iss-2611040712384537"
slug: "land-record-review-refuses-a-return-with"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "the v0.16.1 cut, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/land.go"
remedy: "Accept a verdict line directly after the Reviewed head line, or name the heading in the agents' prompts and in the refusal; a test records both shapes."
refines: [iss-2610311339293870]
resolution: "the reviewers' pages and the refusal name the ### Verdict heading a verdict is read under, and a test records both shapes: a verdict directly under the head's line is refused naming the heading, and the same verdict under the heading is recorded"
impact: fix
resolved_by:
  commit: "497e0231ac1ad4d3a9d7364faa05ec1e30fac739"
---

land --record-review refuses a return with no ### Verdict heading, but nothing tells a reviewer so when the prompt asks for a verdict line, and a reviewer's verdict on the line after its head line is unambiguous: two reviews on 2026-11-04 had to be reformatted by hand.
