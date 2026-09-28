---
schema_version: 1
id: "iss-2609270028432249"
slug: "the-xargs-wrapper-grammar-in-the-guard-lacks-the-bsd-value"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainG2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
resolution: "xargs's wrapper grammar names BSD's -J, -R and -S value flags; TestBSDXargsValueFlagsAreStepped holds it."
impact: fix
resolved_by:
  commit: "88d92df6d"
---

The xargs wrapper grammar in the guard lacks the BSD value flags -J, -R and -S, so on the macOS xargs a search piped into an xargs that uses one of them before a kill reads the flag's value as the launched command and warns as an unrecognised launcher instead of blocking as kill-by-search. Pre-existing; found by review-drainG2 (match.go wrapperValueFlags for xargs).

## Grounds

- pursued: a search piped into an xargs using -J, -R or -S before a kill blocks as kill-by-search, and a literal pid through the same flags allows; a warn or allow on the search spelling would show it wrong
