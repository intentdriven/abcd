---
schema_version: 1
id: "iss-2610261617171196"
slug: "the-reviews-charter-self-test-once"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "lowcpu run 2026-10-25: gates on the PR 1134 merge"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-reviews-cases.sh"
remedy: "Match the needle with a here-string instead of a pipe, so that only grep's own status decides."
---

The reviews-charter self-test once reported its case 'RD001 a symlinked review folder' as failing although the output it printed contained RD001, and the case passed on a rerun. scripts/check-reviews-cases.sh checks a case's needle by piping printf into grep -q under set -o pipefail, so the pipeline's status is the writer's too. When grep has matched, a writer that exits non-zero still makes the check report the needle missing.
