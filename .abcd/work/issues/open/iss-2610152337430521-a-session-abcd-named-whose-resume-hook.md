---
schema_version: 1
id: "iss-2610152337430521"
slug: "a-session-abcd-named-whose-resume-hook"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "confirmation review of itd-2610111707085098, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname/register.go"
remedy: "On a clear with no entry on this host, consult the old id's given record when the host can name the session's previous id, or keep a per-host note of the last abcd name set; test the failed-resume-hook clear followed by a restart and a resume."
---

A session abcd named whose resume hook never ran (so the register holds no live entry for it) and which the person then clears is recorded as carrying the person's own name; after naming restarts and a new session is given that name, a resume of the cleared id keeps it, so two running sessions show the same name (C5, C6). It needs an earlier hook failure. Found by the confirmation review of 88f431b21, 2026-10-16, recorded as not blocking.
