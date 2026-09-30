---
schema_version: 1
id: "iss-2609301128211767"
slug: "the-loop-admits-a-zero-padded-issue-key-issueidre-in"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/receipt.go"
remedy: "Refuse a leading zero in the loop's one issue-id shape, ^iss-[1-9][0-9]{0,19}$, so the key check, a drain lane's issue, a state file's key and a receipt's resolves all refuse a padded id; test iss-02609292352131344 refused at the key."
resolution: "The loop's issue-id shape refuses a leading zero, so no padded key reaches a run, a drain lane, a state file or a receipt."
impact: fix
resolved_by:
  commit: "3c41d67203647bfbfe42337a5adc05d13e044dcf"
---

The loop admits a zero-padded issue key: issueIDRe in internal/core/implement/loop/receipt.go is ^iss-[0-9]{1,20}$, so abcd build iss-02609292352131344 opens a run keyed by the padded spelling, which the brief, the DECISIONS lookup, the branch, the PR title and capture resolve all carry, and which drain's == dedupe of runs misses (review-drainLoop finding 1).

## Grounds

- pursued: abcd build iss-02609292352131344 is refused at the key check and iss-1 still passes; a padded id accepted anywhere the loop reads an issue key would show it wrong (TestAPaddedIssueIdIsNoIssueKey, TestAnIssueKeyIsRefusedUnlessItsShapeAndTheRuleAdmitIt).
