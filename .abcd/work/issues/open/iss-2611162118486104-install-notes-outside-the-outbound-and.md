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
---

Install notes outside the outbound and arm steps still carry git's message or name the checkout by its home path: identity_establish.go:214 puts git config's combined output (a stale lock names the shared config by its full path in a linked worktree) into a refusal; identity_establish.go:131 and :192 and remote.go:390 carry RootCommit's stderr; and apply.go:1337 runs errText before RedactRoot, so home redaction defeats the working-tree redaction for a checkout under the home, the order the outbound receipts just dropped.
