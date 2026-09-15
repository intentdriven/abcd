---
schema_version: 1
id: "iss-2608300848049813"
slug: "itd-180-sixth-round-nits"
severity: "nitpick"
category: "inconsistency"
source: "impl-review"
found_during: "itd-180 sixth-round security review, 2026-08-30"
found_at: "internal/core/lint/readingoutstanding.go, internal/core/capture/reading.go"
---

itd-180 sixth-round nits: a permission-denied run or item directory aborts the whole outstanding report (and an enabled rule fails the lint run) instead of being routed to Unsafe with a reason as files are; a not-well-formed record is listed as standing indistinguishably from a readable one, and the prescribed hand repair (write supersedes_disposition into the surplus record) is inert when the surplus record is the malformed one because its supersession is discarded — mark illegible ids in the contest message; the board and findReadingItem describe a symlinked item file differently. Pre-existing, out of scope: capture verbs take the working directory as the repo root.
