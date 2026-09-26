---
schema_version: 1
id: "iss-2608300848049813"
slug: "itd-180-sixth-round-nits"
severity: "nitpick"
category: "inconsistency"
source: "impl-review"
found_during: "itd-180 sixth-round security review, 2026-08-30"
found_at: "internal/core/lint/readingoutstanding.go, internal/core/capture/reading.go"
resolution: "Unlistable reading and disposition directories go to Unsafe with a reason, contests mark illegible standing ids, and both locators name a symlinked record file as not a regular file."
impact: fix
resolved_by:
  commit: "869ccf13"
---

itd-180 sixth-round nits: a permission-denied run or item directory aborts the whole outstanding report (and an enabled rule fails the lint run) instead of being routed to Unsafe with a reason as files are; a not-well-formed record is listed as standing indistinguishably from a readable one, and the prescribed hand repair (write supersedes_disposition into the surplus record) is inert when the surplus record is the malformed one because its supersession is discarded — mark illegible ids in the contest message; the board and findReadingItem describe a symlinked item file differently. Pre-existing, out of scope: capture verbs take the working directory as the repo root.

## Grounds

- pursued: a mode-000 run or item directory leaves the report and the lint run intact with an Unsafe line, a contest with a malformed record names it, and a symlinked rdi or dsp file is refused as not a regular file; an abort, an unmarked contest or an unknown-id message would show it wrong
