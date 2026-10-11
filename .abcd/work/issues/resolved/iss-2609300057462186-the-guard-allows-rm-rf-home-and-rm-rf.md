---
schema_version: 1
id: "iss-2609300057462186"
slug: "the-guard-allows-rm-rf-home-and-rm-rf"
severity: "major"
category: "security"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
remedy: "argValueMatches also compares a field whose leading run of / stands before $HOME, ${HOME}, $PWD or ${PWD} with that run taken out, since each of those values is an absolute path and a leading / before an absolute path names the same directory; grounds: POSIX pathname resolution reads a run of slashes as one."
resolution: "argValueMatches reads a run of / before $HOME or $PWD as that directory; TestSeparatorsBeforeTheHomeTheWrittenCompareReads"
impact: fix
resolved_by:
  commit: "598f47756"
---

The guard allows rm -rf /$HOME and rm -rf /${HOME}, which delete the home: the arg_values compare cleans a run of separators inside a path (//*, $HOME//) but not one written before a home spelling, so /$HOME names none of the values. Found in the fix-guardSet sibling sweep; outside the four review findings.

## Grounds

- pursued: rm -rf /$HOME and //${HOME}/ block while /$HOME/build and /$HOMEDIR stay allowed; another prefix that names the home and still allows would show it wrong
