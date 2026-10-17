---
schema_version: 1
id: "iss-2610171626344967"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-memory.md"
remedy: "Rewrite the passage at .abcd/development/brief/04-surfaces/37-memory.md:21 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
---

The v0.15.1 release review found the brief at .abcd/development/brief/04-surfaces/37-memory.md:21 out of step with the shipped code. It says: `list` is a shipped sub-verb of /abcd:memory; the naming constraint (.abcd/development/brief/02-constraints/04-naming.md:90-94) forbids a plain unfiltered `<verb> list` at design time, a sub-verb having to do something the bare invocation … In the code: `abcd memory list` takes no flags and no argument ('refuses any argument') and lists every note in both layers unfiltered; neither 37-memory.md nor the bare-invocation enumeration in 04-surfaces/README.md records it as an exception to the rule.
