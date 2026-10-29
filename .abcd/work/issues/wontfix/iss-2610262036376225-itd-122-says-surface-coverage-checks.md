---
schema_version: 1
id: "iss-2610262036376225"
slug: "itd-122-says-surface-coverage-checks"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 661 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-122-the-registry-cannot-wave-its-hands-every.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-122]
finding_ref: cfk-59976fb16e363090
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. itd-122 never claims the tree can verify a bucket, and its audit shows the bucket enum registered and the arguable bucketings ruled by hand; the chapter note states the same limit."
---

itd-122 says surface_coverage checks every sub-verb row against the command tree; the lint chapter's table note says a row's bucket cell is not checked against the tree

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-122-the-registry-cannot-wave-its-hands-every.md:19` — "`surface_coverage` now checks every row against the binary's own command tree, both ways"
- End B: `.abcd/development/brief/04-surfaces/16-lint.md:56` — "wrong but legal passes, and that cell stays a review-grain claim"

itd-122 presents the sub-verb table as recording each verb's bucket and existence, with surface_coverage checking every row against the binary's command tree. The machine-checked note in 16-lint says only existence is verified against the snapshot, and a wrong but legal bucket passes and stays a review-grain claim. Half of each row is not checked as itd-122 says.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 661.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. itd-122 never claims the tree can verify a bucket, and its audit shows the bucket enum registered and the arguable bucketings ruled by hand; the chapter note states the same limit.
