---
schema_version: 1
id: "iss-2609020219265817"
slug: "sanitizerulebody-indents-every-continuation-line-of-a-rule-b"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/rules.go"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Are rule bodies escaped, fenced, or left as the host parser contract defines them?"
---

sanitizeRuleBody indents every continuation line of a rule body by two spaces, which defuses the line-start contract the host-side parser splits on but not CommonMark: a two-space-indented heading line inside a list item still renders as a heading INSIDE that item. A repo-overridden domain can therefore put an unmarked heading in front of any reader that renders the injected block as markdown, wearing no repo-override label of its own. The line-start contract holds and this is already stronger than the pre-fix behaviour, which indented nothing at all; closing the CommonMark half needs a decision on whether rule bodies get escaped, fenced, or left as the parser contract defines them. Evidence: sanitizeRuleBody in internal/core/rules/rules.go.
