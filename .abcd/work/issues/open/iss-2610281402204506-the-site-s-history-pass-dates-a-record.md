---
schema_version: 1
id: "iss-2610281402204506"
slug: "the-site-s-history-pass-dates-a-record"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "landing #996, crash salvage lane C, 2026-10-28"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/dates.go"
remedy: "date a record only from the commits on the trunk's first-parent chain, reading a merge's first-parent diff as the day the content landed, so a merge of main inside a branch's history cannot re-date main's records; keep a branch-written file's creation day only where the trunk never held it"
---

The site's history pass dates a record from every commit it walks, branch sides included, and reads a merge commit's diff against its first parent. A branch that merges main the usual way, with the branch as first parent, so carries a commit whose diff shows every record main moved since the branch was cut, dated the day of that merge. Once such a branch lands, those records read as having entered their folders on that day. The home page features the most recently shipped intent whose audit is met, so the featured quotation can change for reasons the landing diff does not contain, and the site's banned-token gate then judges a different press release. On 2026-10-28 an unpushed merge of main into the ideate-librarian bundle branch, from 23 Oct, re-dated main's shipped intents. That made itd-2609091034175565 the featured intent, its press release tripped present_tense/no-longer, and site-render failed on a branch whose diff touched no shipped intent. The branch was rebuilt with main as the merge's first parent and passed.
