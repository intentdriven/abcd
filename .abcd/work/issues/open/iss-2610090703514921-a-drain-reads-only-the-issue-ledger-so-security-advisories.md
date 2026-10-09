---
schema_version: 1
id: "iss-2610090703514921"
slug: "a-drain-reads-only-the-issue-ledger-so-security-advisories"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "2026-10-09 product thinker request after the v0.13.3 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/drain.md"
remedy: "Ruled 2026-10-09 by the product thinker: private advisories get a separate security-advisory drain (for example `abcd drain --security`), distinct from the general drain, whose rule stays as it is; its design starts from a primary-source state-of-the-art check on coordinated disclosure, and its requirements are drawn from a hand-run drain recorded as an abcd lab. That hand-run drain keeps each advisory's issue out of the public ledger until its fix lands, holds the fixes on local branches, and lands them together just before the release cut that publishes the advisories."
---

A drain reads only the issue ledger, so security advisories that others submit through the forge's private vulnerability reporting never reach it. Unpublished advisories are not named by any record, so nothing in abcd shows whether they were triaged. They cannot simply be copied into the ledger as issues, because the ledger is public and an advisory stays private until it is fixed; and even a ledger issue in the security category is handed back by the drain under its default rule.

## Ruling 2026-10-09

The product thinker ruled three things on 2026-10-09, after unpublished advisories on the forge were found that no record named:

1. The first drain of private advisories is run by hand, outside `abcd drain`,
   and recorded as an abcd lab: "this is an abcd lab to ensure we're on the
   right track". The lab is lab-261009081338-7549ca2, in the user-level lab
   store; its harvest lists the steps the hand-run drain did that a command
   would have to do.
2. Later, a separate security-advisory drain, distinct from the general drain
   ("a future e.g. /abcd:drain --security or similar command"), designed after
   a state-of-the-art check.
3. The fixes are held on local branches and land together just before the
   release cut that publishes the advisories.
