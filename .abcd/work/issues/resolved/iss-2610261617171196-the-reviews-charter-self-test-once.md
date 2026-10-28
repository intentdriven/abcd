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
resolution: "scripts/check-reviews-cases.sh now matches a case's needle with a here-string, so only grep's own status decides; all 23 cases pass under bash 5 and /bin/bash 3.2."
impact: internal
resolved_by:
  commit: "3b9846dcb5dfe20767bc15d094e5d5fc16339669"
---

The reviews-charter self-test once reported its case 'RD001 a symlinked review folder' as failing although the output it printed contained RD001, and the case passed on a rerun. scripts/check-reviews-cases.sh checks a case's needle by piping printf into grep -q under set -o pipefail, so the pipeline's status is the writer's too. When grep has matched, a writer that exits non-zero still makes the check report the needle missing.

## Grounds

- pursued: the case misread came from the pipe's status under pipefail, not from the gate's output; a case that again reports a needle missing while its printed output holds it would show this wrong
