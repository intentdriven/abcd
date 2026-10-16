---
schema_version: 1
id: "iss-2610152329345301"
slug: "the-pre-lock-filter-mayhold-admits-every"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "final review of itd-2610111707085098, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname/register.go"
remedy: "Narrow mayHold's clear clause to a title this host's entries account for; test that a clear of a person's own title reads no process table."
---

The pre-lock filter mayHold admits every clear whose title appears anywhere in given, so such a clear takes the register lock and reads the process table even though a clear can no longer be read as inheriting; extra work, no wrong answer.
