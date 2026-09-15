---
schema_version: 1
id: "iss-2609020544031234"
slug: "bash-s-deprecated-arithmetic-form-is-not-modelled-as-an-arit"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
---

bash's deprecated $[ … ] arithmetic form is not modelled as an arithmetic context, so a '<<' inside it is read as a here-document redirection: 'echo $[ 1 << EOF ]' followed by any command blocks under heredoc-unterminated on both this branch and v0.7.0, where bash evaluates 1<<0 and runs on. Widening isDelimStart to digits and $-led words (the fix for the fail-open sibling) widens this residual from EOF-shaped operands to every operand, so '$[ 1 << 20 ]' now over-blocks too. Evidence symbol: inArithmetic / the parens stack (internal/core/guard/tokenize.go), which recognises $(( and (( and nothing else. The direction is fail-CLOSED and loud, and the form has been deprecated since bash 2.0, so this is recorded rather than fixed: the fix is a third frame kind on the parens stack ($[ opens, ] closes) and a decision about whether the guard should track a construct bash itself discourages.
