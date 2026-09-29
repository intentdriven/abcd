---
schema_version: 1
id: "iss-2608311949421873"
slug: "the-include-table-match-grammar-disagrees-with-itself-on-cas"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "Row.Match states the one case rule the include table follows and matches restates it: a form naming a kind of file folds case (an extension), a form naming one file matches its committed spelling (a basename), a form following a tool's own rule keeps it (MatchSuffix). A new form states which it is. No compare changed, so the assembler's admission and version stand; TestTheMatchFormsFollowTheOneCaseRule pins each form."
impact: internal
resolved_by:
  commit: "e037d9840"
---

The include table match grammar disagrees with itself on case for no stated reason: an extension entry is compared with strings.EqualFold while an exact basename entry is compared with ==, so .MD matches but makefile does not match Makefile, and whoever adds a fourth match form has no rule to follow

## Grounds

- pursued: a stated rule gives the next match form a compare to take; a form added without saying which kind it is, or a compare that contradicts its stated kind, would show it wrong
