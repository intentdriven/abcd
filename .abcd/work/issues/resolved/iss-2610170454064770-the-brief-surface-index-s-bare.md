---
schema_version: 1
id: "iss-2610170454064770"
slug: "the-brief-surface-index-s-bare"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "PR8 build of the v0.15.0 deferred verb findings"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Say in the Bare invocation section that hidden operator verbs such as hook are outside the list, or add hook to the list and to the bare-render exception table."
resolution: "The Bare invocation section now says its enumeration, the exception table and the bare-render test cover visible verbs only, so hidden hook and the framework's help and completion are outside them."
impact: internal
---

The brief surface index's Bare invocation list (.abcd/development/brief/04-surfaces/README.md, section Bare invocation) names the verbs whose bare call refuses or acts instead of rendering, but bare abcd hook prints its usage only and is in neither that list nor the bare-render exception table (internal/surface/cli/barerender.go). hook is hidden and operator-internal, and the list never says hidden verbs are left out.
