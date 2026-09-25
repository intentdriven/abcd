---
schema_version: 1
id: "iss-2608241612007530"
slug: "the-issue-resolution-gate-is-triggered-by-the-trailer-so-a-f"
severity: "major"
category: "process"
source: "agent-finding"
found_during: "v0.6.4 release validation 2026-08-24"
found_at: "scripts/check-issue-resolution.sh"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Does the inverse detector warn or refuse when a change touches code an open record describes and declares no resolution?"
---

the issue-resolution gate is triggered by the trailer, so a fix that lands without one is invisible to it. RS001 fires only on a commit carrying a Resolves trailer, and RS002/RS003 only on stamps that already exist, so a merged fix whose commit names no issue passes all three rules and leaves its record in open/ — the exact backlog iss-2608241347321757 was built to stop, reached from the other direction. Measured 2026-08-24: 1315 commits name an iss- id somewhere in the message and 3 carry the trailer. iss-202 is the standing instance: its fix merged on 2026-08-24 in a commit ending with a bare 'iss-202' line rather than the trailer form, and its record sits in open/ with the fix shipped. The missing detector is the inverse direction — a change touching code an open record describes, declaring no resolution.