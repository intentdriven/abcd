---
schema_version: 1
id: "iss-173"
slug: "three-tier-layout-s-local-artefact-placement-check-matches-e"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "iss-155 adversarial review round 2026-07-29"
found_at: "internal/core/audit/rule_layout.go"
resolution: "Fixed: three-tier-layout lists directories instead of probing fixed paths, matches NEXT.md, scratch/ and logs/ in any case, reports them at the .abcd/ root as well as directly under a committed tier, and reports NEXT.md at any depth in a committed tier. scratch/ and logs/ are not flagged below a tier's top level, since a durable record may legitimately nest a study's own logs/; that residual is stated."
impact: fix
resolved_by:
  commit: "f2ec7c8f0"
---

three-tier-layout's local-artefact placement check matches exact names directly under a committed tier only — residual evasions flagged at review: a nested artefact (a NEXT.md one directory below a tier root), a NEXT.md at the .abcd/ root itself (one directory off the modelled incident), and lowercase name variants on case-sensitive filesystems all pass clean. Widening (depth, roots, case folding) must not start flagging legitimate tier content.

## Grounds

- pursued: a nested handover, a local artefact at the .abcd root, and another-case spelling are each flagged, and legitimate nested tier content is not; a fixture of any of those shapes that lints clean, or a study's nested logs/ that is flagged, would show it wrong
