---
schema_version: 1
id: "iss-2610262032226321"
slug: "ahoy-calls-the-remote-apply-the-one-verb"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 259 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:90"
remedy: "none (filed automatically)"
finding_ref: cfk-a2e7414a99a625bb
---

Ahoy calls the remote apply the one verb that mutates state outside the machine; launch says the reservation verb means a push to origin

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:90` — "the one abcd verb that mutates state outside this machine"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:757` — "A push from it would change what the verb means, where the reservation verb already means exactly that push."

Ahoy says the remote apply is the one abcd verb that mutates state outside this machine. Launch has the cutting session take the release hold with the reservation verb, and says that verb already means exactly a push of the reservation to origin, a remote write every other session's land step then reads. Both cannot be the one remote-mutating verb.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 259.
