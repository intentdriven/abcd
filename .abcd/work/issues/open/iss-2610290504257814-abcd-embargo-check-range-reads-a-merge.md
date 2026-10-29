---
schema_version: 1
id: "iss-2610290504257814"
slug: "abcd-embargo-check-range-reads-a-merge"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "embargo close-out, a range check after a merge of main"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/embargo/check.go"
remedy: "Judge a merge commit by the lines it adds against every parent (git's combined diff), since a line from a parent in the range is read in that parent's commit and one from a parent outside it is already behind the range's start; and pass an advisory-format id the repository's own GitHub advisories list as published."
---

abcd embargo check --range reads a merge commit against its first parent, so on a branch that merged the default branch every line the merge brings in from it is judged as the branch's own: text already public on the default branch is refused, including the id of an advisory this repository has already published, which the store need not hold. Two sessions met it on 2026-10-29 and had to check around the merge by hand.
