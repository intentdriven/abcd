---
schema_version: 1
id: "iss-2610152329330129"
slug: "a-clear-of-a-session-abcd-named-and-the"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "final review of itd-2610111707085098, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname/register.go"
remedy: "On a clear with a person's title, drop or re-key the live entry on this host whose given record is abcd's old name; test that the old name is free after the clear once the process ends nothing else."
---

A clear of a session abcd named and the person then renamed leaves the old id's live entry (abcd's name) held under the same host, under an id no session carries, until the host process exits; that name is wasted for the process's life.
