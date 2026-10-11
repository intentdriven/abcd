---
schema_version: 1
id: "iss-2608221456463223"
slug: "the-guard-tokenizer-does-not-recognise"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "all-dimensions bug-hunt round 7"
found_at: "internal/core/guard/tokenize.go"
resolution: "Tokenizer now decodes $'...' and $\"...\" so the flag reaches command position; watched-fail test on git push $'--force'."
impact: fix
resolved_by:
  commit: "d46a08c"
---

The guard tokenizer does not recognise bash ANSI-C quoting or locale quoting, so \$'--force' tokenises to \$--force and a Tier-1 blocker (git push --force, rm -rf, --no-verify) is silently allowed while bash hands the child byte-identical argv, contradicting the doc.go invariant that quoting affects tokenisation not argument semantics.