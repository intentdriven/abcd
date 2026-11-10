---
schema_version: 1
id: "iss-2611101222229613"
slug: "a-reviewer-s-verdict-line-that-names-the"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "live drive, 2026-11-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/review.go"
remedy: "In the verdict reader, do not count a run of digits that is part of a record id (prefixed iss-, itd-, spc-, adr-, dcn-, run-) as a name, and say in the review brief that the verdict line carries the head and nothing else that looks like one"
resolution: "The verdict reader takes a record id's digits as that id's, not as a second name, and the review brief says the verdict line names nothing else that reads as a commit or digest name"
impact: fix
---

A reviewer's verdict line that names the judged commit in full and also mentions a record id is refused: the verdict reader takes any run of seven or more hex characters holding a digit as a name, and a timestamp record id such as the digits of an issue id fits. A drive's ruthless reviewer answered SHIP with the full head and the issue it fixes on one line, and the receipt was refused as naming another commit; the review brief asks for the head on that line and says nothing about keeping record ids off it. Each such refusal costs a whole review launch, and three hand the lane back.
