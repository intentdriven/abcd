---
schema_version: 1
id: "iss-2611101318039706"
slug: "abcd-build-plans-an-intent-s-next-step"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/spec/steps.go"
remedy: "Have build read a step as landed when the default branch's history holds a merged pull request or commit the loop composed for it, and offer to write the missing mark; a test lands a step with no mark and finds build plans the next one."
---

abcd build plans an intent's next step from the spec's landed marks alone, so a step that landed without a mark is planned again. On 9 Nov at 22:22 build planned step 1 of the checklist spec, which had already landed, and the lane agent stopped the run and wrote the mark by hand; on 10 Nov at 00:56 steps 1, 2, 3 and 5 carried no mark and build planned step 1 for the second time. The marks were backfilled by hand in 1f06043c5, but nothing keeps it from happening again.
