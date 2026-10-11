---
schema_version: 1
id: "iss-2609251645374557"
slug: "the-help-renderer-computes-the-name"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "renderRootHelp sizes each block's name column over its own names (internal/surface/cli/helpgroups.go), so the person's block is byte-identical under --help and --help --agent; TestPeopleBlockIsTheSameUnderBothHelps holds it."
impact: fix
resolved_by:
  commit: "8cf52a722"
---

The help renderer computes the name column width over both blocks, so the person's block renders at width 12 under --help and 21 under --help --agent: the people block is not byte-stable across the two forms (internal/surface/cli/helpgroups.go:266-274; review-helpgroups 2).

## Grounds

- pursued: we expect the person's block to render the same bytes in both help forms; any byte difference between the two would show it wrong
