---
schema_version: 1
id: "iss-2610160957404827"
slug: "abcd-drain-reads-the-issue-ledger-of-the"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "drain check from the primary checkout, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "Before a run, and in the dry run's output, abcd drain compares the checkout with origin's default branch and refuses (or warns in --dry-run) when it is behind, naming the command that updates it"
---

abcd drain reads the issue ledger of the checkout it runs in and never checks whether that checkout is behind origin's default branch: from a primary checkout 185 commits behind main, the dry run listed 162 eligible issues from the stale ledger, many since fixed or changed, and a run would judge, reserve and work from that list.
