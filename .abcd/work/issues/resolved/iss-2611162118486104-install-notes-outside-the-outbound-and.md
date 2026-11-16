---
schema_version: 1
id: "iss-2611162118486104"
slug: "install-notes-outside-the-outbound-and"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "correctness review of the iss-2610152204588951 fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/identity_establish.go"
remedy: "Put each of those errors through causeInWords (or, where only the working tree's own path is in play, hand the raw error to a redaction that takes the working tree before the home), with a stale-lock and a broken-git test for each site family."
resolution: "Every install note and refusal that carried git's root-commit or config-write message now says why in words through causeInWords and names the git command that shows git's reason: the identity proposal and write, the remote verbs' refusal, the install's session-store refusal, and the install's adoption refusal, which also named the checkout's location and keeps git's error in the chain through wordedError; a git that does not answer in time is said so. TestRootCommitRefusalsKeepGitsMessageOut, TestIdentityProposalRefusalKeepsGitsMessageOut and TestStepGitIdentity_WriteRefusalKeepsGitsMessageOut hold it."
impact: fix
---

Install notes outside the outbound and arm steps still carry git's message or name the checkout by its home path: identity_establish.go:214 puts git config's combined output (a stale lock names the shared config by its full path in a linked worktree) into a refusal; identity_establish.go:192 and remote.go:390 carry RootCommit's stderr (identity_establish.go:131 does not: its git reads give only the exit status); and apply.go:1337 runs errText before RedactRoot, so home redaction defeats the working-tree redaction for a checkout under the home, the order the outbound receipts just dropped.

## Grounds

- pursued: no install note or refusal names a path outside the working tree or quotes git, while each still says what failed and how to see git's reason; a note that a person cannot act on would show it wrong
