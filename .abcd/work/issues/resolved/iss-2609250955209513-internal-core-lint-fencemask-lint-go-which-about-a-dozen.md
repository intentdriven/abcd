---
schema_version: 1
id: "iss-2609250955209513"
slug: "internal-core-lint-fencemask-lint-go-which-about-a-dozen"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
resolution: "lint's fenceMask is mdrecord.FencedUnderEveryRule: tilde fences are fences, a backtick line inside a tilde block is content, and a line is masked only where every fence reading agrees, so a disagreement reads as prose (loud) rather than hiding a finding (silent). Commented text stays visible to the rules, as before."
impact: fix
resolved_by:
  commit: "cb48c449"
---

internal/core/lint fenceMask (lint.go), which about a dozen rules use to skip example text (citations, persona, context currency, delivery state, cross-store, subverbs and the rules in lint.go), is a private backtick-only toggle that diverges from mdrecord.Mask. It flips on any line whose trimmed text starts with three backticks. So a tilde-fenced example is linted as live prose, which gives false findings. And a three-backtick line inside a ~~~ block flips the mask on, so the live prose after the block is masked until the next backtick line, and a real finding there is silently missed. Probe at 70daf701: in lines [~~~, three-backtick-go, ~~~, blank, LIVE PROSE LINE], fenceMask marks the live line masked, and in [~~~, example prose, ~~~] it marks the example unmasked. Moving onto mdrecord.Mask is not a mechanical swap. mdrecord also masks HTML comments, so every rule would stop reading commented text, and it recognises only the 0-3 space fence indent. Both are per-rule decisions.

## Grounds

- pursued: the capture's two probes (live prose after a tilde block holding a backtick line; a tilde-fenced example) mask correctly (TestFenceMaskReadsTildeFencesAndTheirContents) and links_resolve reports the live broken link after such a block (TestLinksResolveAfterATildeBlockHoldingABacktickLine), with record-lint and docs-lint still clean on the tree; a masked live line or a new blocker on the committed corpus would show it wrong
