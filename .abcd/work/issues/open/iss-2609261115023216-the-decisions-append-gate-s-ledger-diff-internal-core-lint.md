---
schema_version: 1
id: "iss-2609261115023216"
slug: "the-decisions-append-gate-s-ledger-diff-internal-core-lint"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-daport"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/decisionsappend.go"
---

The decisions-append gate's ledger diff (internal/core/lint/decisionsappend.go, analyse) pins --unified=0 and the diff algorithm but not the other hunk-shaping options a repository's own config supplies. A repo-config diff.interHunkContext above zero folds neighbouring -U0 hunks into one with context lines between them, and the old-line arithmetic (removed line k is old line a+k-1; an addition sits after a+b-1) then counts across the context: two separate DA002 findings collapse into one at the wrong span. diff.indentHeuristic=false likewise slides an ambiguous hunk, which moves the line a finding names. Config is not in the tree and CI's is trusted, and the shell gate had the same exposure, so the effect is on a local run over a hostile or merely unusual config. Fix: pin --inter-hunk-context=0 and --indent-heuristic (git's default, which the rules and their cases were written against) on the diff.
