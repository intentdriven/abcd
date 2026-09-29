---
schema_version: 1
id: "iss-2609290321312087"
slug: "the-shell-guard-s-arg-values-operand-compare-lost-a-variable"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
resolution: "arg_values compares each operand's written spelling of its variables (segment.spelled), carried into shell strings by a paired second reading; every other matcher reads the unchanged tokens."
impact: fix
resolved_by:
  commit: "9118e6709"
---

The shell guard's arg_values operand compare lost a variable's written spelling when main began writing a parameter expansion as the unknown-value mark: rm -rf $HOME, "$HOME", ${HOME} and $HOME/.* allowed, rm -rf $PWD stopped warning, and rm -rf "$BUILD_DIR"/* and rm -rf $OUT/ blocked as deletes of the root because the variable was read as empty text. The tokenizer needs to carry each word's written spelling of its variables to arg_values without changing what any other matcher reads.

## Grounds

- pursued: rm -rf $HOME, "$HOME", ${HOME} and $HOME/.* block, $PWD forms warn, and "$BUILD_DIR"/* and $OUT/ allow, at top level and inside sh -c and bash -c strings (argspelling_test.go); a verdict diff of 12,515 inputs against main's tip with the two rm entries removed shows zero changes. A variable-bearing target that allows, or any other entry's verdict moving, would show it wrong.
