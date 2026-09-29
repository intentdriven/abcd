---
schema_version: 1
id: "iss-2609290625482831"
slug: "the-shell-guard-s-rm-rf-root-or-home-compares-an-operand-to"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
resolution: "arg_values also compares each operand with its redundant separators taken out (cleanSeparators): a slash run is one, a /./ segment is dropped and a /../ directly under the root is the root; TestRootAndHomeWithRedundantSeparators pins 20 operands bare and in bash -c."
impact: fix
resolved_by:
  commit: "a797dfa11"
---

The shell guard's rm-rf-root-or-home compares an operand to its arg_values as one exact word, so a root or home operand written with repeated slashes, a trailing double slash, a /./ segment or a /../ segment under the root allows: rm -rf //*, rm -rf $HOME//, rm -rf ~//*, rm -rf /./* and rm -rf /../* all allow, while rm -rf /* and rm -rf $HOME/ block. The kernel reads each as the root or the home (a slash run is one separator, . is the directory itself and the root is its own parent). rm-rf-working-directory has the same gap (rm -rf .//* allows). Present at main 285455056 and at d53e21cf3.

## Grounds

- pursued: rm -rf //*, $HOME//, ~//*, /./* and /../* block while //tmp/x, /tmp//x and $HOME//x allow; a root or home operand spelled with redundant separators that allows would show it wrong
