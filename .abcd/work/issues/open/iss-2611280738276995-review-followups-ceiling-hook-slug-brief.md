---
schema_version: 1
id: "iss-2611280738276995"
slug: "review-followups-ceiling-hook-slug-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/status.md"
remedy: "narrow the ceiling wording to what maxlive.go reads, make the hook header name both causes, correct the resolution phrase, and align the brief's settle sentence with commands/reserve.md"
---

Review of landed changes left four small wording follow-ups: the ceiling row's absence condition in commands/status.md and commands/peers.md claims any refused value withholds it, though the cap reads only reservation.max_live and key names; the commit-msg hook's record-cap header names only a slug over the cap, though the record-size refusal shares it; the resolution phrase of iss-2611070737092992 says a trailing comment hid the slug field, though the old scanner counted it into the length; and the reserve brief says the next listing settles ended releases, though only abcd peers and a reservation act do
