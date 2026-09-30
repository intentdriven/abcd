---
schema_version: 1
id: "iss-2609300057326778"
slug: "the-guard-allows-rm-rf-x-and-x-which-bash-3-2-and-bin-sh"
severity: "major"
category: "security"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
remedy: "spellParameterAt reads the parameter name as bash does (paramNameEnd): a name, a positional digit run, or one special byte, after an optional indirection !, and applies the operators to each; an indirection value past an operator reads as capped, since the variable it names is not in the line; grounds: printf under bash 3.2, /bin/sh, dash and bash 5.3."
resolution: "spellParameterAt reads indirect, positional and special parameters through paramNameEnd; TestIndirectAndSpecialDefaultsTheWrittenCompareReads"
impact: fix
resolved_by:
  commit: "598f47756"
---

The guard allows rm -rf ${!X:-/} and ${!X-/}, which bash 3.2 and /bin/sh print as / with X unset: spellParameterAt returns the expansion as written whenever the body does not begin with a name byte (review-guardSet MAJOR-4). The same test drops every positional and special parameter, so ${1:-/}, ${@:-/}, ${!:-/} and ${#:+/} are allowed though every shell prints /.

## Grounds

- pursued: ${!X:-/}, ${1:-/}, ${@:-/}, ${!:-/} and ${#:+/} block while ${!X}, ${!X*}, ${#X} and ${1:-dist} stay allowed; a parameter spelling bash reads with operators that still allows would show it wrong
