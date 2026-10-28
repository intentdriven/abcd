---
schema_version: 1
id: "iss-2610262032326065"
slug: "ahoy-calls-the-remote-apply-the-one-verb"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 284 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:90"
remedy: "none (filed automatically)"
finding_ref: cfk-be4689508c61d8ae
---

Ahoy calls the remote apply the one verb that mutates state off this machine; reserve exists to write a branch on the shared remote

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:90` — "the one abcd verb that mutates state outside this machine"
- End B: `.abcd/development/brief/04-surfaces/37-reserve.md:4` — "it reserves the record on the shared remote as a branch nobody merges"

01-ahoy states that the remote apply is the one abcd verb that mutates state outside this machine. 37-reserve's whole purpose is to write the reservation, as a branch, on the shared remote that other people and computers read. That is a push to origin, which is state outside this machine. A reader who trusts ahoy's claim would not count the reservation verb among abcd's outward writes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 284.
