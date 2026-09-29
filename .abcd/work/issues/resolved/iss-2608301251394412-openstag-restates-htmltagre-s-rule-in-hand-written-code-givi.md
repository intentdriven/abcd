---
schema_version: 1
id: "iss-2608301251394412"
slug: "openstag-restates-htmltagre-s-rule-in-hand-written-code-givi"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "itd-183-round-9-ruthless"
found_at: "internal/core/reading/project.go"
resolution: "opensTag and htmlTagRe build from one constant, htmlTagOpen, in internal/core/reading/project.go: htmlTagRe appends the rest of the tag and opensTag asks htmlTagOpenRe, the anchored opening half. The hand-written copy disagreed with the pattern on autolinks, which it opened on; TestOpensTagIsHTMLTagResRule now holds the two to agreement."
impact: internal
resolved_by:
  commit: "c8732bf07"
---

opensTag restates htmlTagRe's rule in hand-written code giving one file two definitions of what opens a tag

Found by the round-9 adversarial RUTHLESS review of build/itd-183.

`opensTag` (project.go:670-679) restates `htmlTagRe`'s rule in hand-written
code -- its own comment says "on htmlTagRe's own rule". That is two definitions
of "what opens a tag" in one file.

Repo law (one-canonical-primitive): flag for consolidation on the second copy,
never let a third appear. Flagged here so the floor intent inherits it.

## Grounds

- pursued: we expect the attribute walk to open exactly where htmlTagRe matches on any input the pattern reads to its closing bracket; an input where opensTag and htmlTagRe disagree would show it wrong
