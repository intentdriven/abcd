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
remedy: "Waits on the product thinker's ruling on how private advisories enter the drain's view: list them in the drain's plan from the forge without their content and always hand them back; keep a private mirror outside the public ledger that the drain reads; or file a content-free placeholder issue per advisory that the drain can route."
---

A drain reads only the issue ledger, so security advisories that others submit through the forge's private vulnerability reporting never reach it. Unpublished advisories are not named by any record, so nothing in abcd shows whether they were triaged. They cannot simply be copied into the ledger as issues, because the ledger is public and an advisory stays private until it is fixed; and even a ledger issue in the security category is handed back by the drain under its default rule.
