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
resolution: "recordid.IssuesRelDir is the one spelling of the ledger root; the other packages name it through the constant, and a test refuses a new literal."
impact: internal
resolved_by:
  commit: "ee88b50d0e11ae3079e11505fe40c6104595ebbd"
---

The issue ledger's repo-relative root .abcd/work/issues is spelled as a string literal in seven packages (capture.LedgerRelPath, lint issuesDirOf, changelog issuesLedgerDir, readingitem ledgerRelDir, lifeboat nativeIssuesDir, intent issuesRelDir, recordid familyRoots), so a move of the ledger is seven edits and a missed one is a reader looking at an empty tree. One exported constant in the stdlib-only recordid leaf, referenced by the rest.

## Grounds

- pursued: no non-test source under internal/ spells the ledger root outside recordid; TestIssuesLedgerRootIsSpelledOnce would fail on a new literal
