---
schema_version: 1
id: "iss-2610261651489204"
slug: "abcd-intent-consistency-ingest-writes-a"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "lowcpu run 2026-10-25: gates on the consistency branch"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/consistency.go"
remedy: "Have the report writer keep 00-summary.md to the header and the findings table and write each finding's ends and explanation to 01-findings.md, refusing before it writes if the summary alone would pass 1 MiB."
resolution: "A consistency report that would pass the 1 MiB the reviews board reads keeps its header and findings table in 00-summary.md and writes each finding in full to 01-findings.md, and a pass whose table alone would pass the limit is refused before the first finding is filed."
impact: fix
resolved_by:
  commit: "7d5d46ba3"
---

abcd intent consistency ingest writes a 00-summary.md that the reviews-charter gate refuses once a pass files enough findings. renderConsistencyReport puts the table of every finding and every finding's full text into the one summary, and RD004 (scripts/check-reviews.sh) refuses a summary past the 1 MiB the board reads. The 2026-10-26 whole-corpus pass, with 1307 findings, wrote 1.64 MB, and nothing said so until lint-reviews ran on the branch. The summary was split by hand: its table stayed in 00-summary.md and the full findings moved to 01-findings.md.
