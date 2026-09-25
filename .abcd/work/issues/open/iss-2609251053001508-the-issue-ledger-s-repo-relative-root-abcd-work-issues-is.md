---
schema_version: 1
id: "iss-2609251053001508"
slug: "the-issue-ledger-s-repo-relative-root-abcd-work-issues-is"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/recordid/resolve.go"
---

The issue ledger's repo-relative root .abcd/work/issues is spelled as a string literal in seven packages (capture.LedgerRelPath, lint issuesDirOf, changelog issuesLedgerDir, readingitem ledgerRelDir, lifeboat nativeIssuesDir, intent issuesRelDir, recordid familyRoots), so a move of the ledger is seven edits and a missed one is a reader looking at an empty tree. One exported constant in the stdlib-only recordid leaf, referenced by the rest.
