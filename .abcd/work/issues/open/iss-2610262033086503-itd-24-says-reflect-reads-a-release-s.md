---
schema_version: 1
id: "iss-2610262033086503"
slug: "itd-24-says-reflect-reads-a-release-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 365 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:526"
remedy: "none (filed automatically)"
related_intents: [itd-24]
finding_ref: cfk-e01be210324dc27e
---

itd-24 says reflect reads a release's intents the way the cut does, but its shipped_in rule is not the one the launch chapter's cut applies

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:526` — "leaving out an entering record whose `shipped_in:` names a release tag the"
- End B: `.abcd/development/intents/shipped/itd-24-reflect-command.md:45` — "less any whose `shipped_in` names another release, plus any whose `shipped_in` names this one"

In the launch chapter, the cut gathers records that entered `intents/shipped/` since the previous tag. It leaves out only an entering record whose `shipped_in:` names a release the previous tag already reaches, keeps a record whose value names no such tag, and adds nothing from outside the window. itd-24 claims to read the intents 'the way the release cut reads them', yet it drops any record whose `shipped_in` names another release, a later one included. It also adds any record whose `shipped_in` names this release, even one that entered outside the window. For those records the retrospective's seed and the cut's record set differ, so the claim that both read the same way cannot hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 365.
