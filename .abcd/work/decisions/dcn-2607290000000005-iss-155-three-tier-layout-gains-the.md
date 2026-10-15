---
id: dcn-2607290000000005
slug: iss-155-three-tier-layout-gains-the
date: 2026-07-29
---
2026-07-29 — iss-155: `three-tier-layout` gains the placement half of the tier convention — local-tier artefacts (`NEXT.md`, `scratch/`, `logs/`) found directly in a committed tier are now errors, each finding carrying a move-to-`.abcd/.work.local/` fix. The rule verified tier presence and the `.work.local` gitignore but never that local ephemera were ABSENT from `.abcd/work/` and `.abcd/development/`, which is exactly how a handover file carrying host infrastructure detail reached a public repo unflagged. Presence is checked on the filesystem, matching the tier checks themselves: an untracked NEXT.md in a committed tier is one `git add -A` from history. The existing rule extends rather than minting a sibling ID — one convention, one rule.
