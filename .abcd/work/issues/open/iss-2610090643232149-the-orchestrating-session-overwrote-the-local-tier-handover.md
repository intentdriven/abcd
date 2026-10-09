---
schema_version: 1
id: "iss-2610090643232149"
slug: "the-orchestrating-session-overwrote-the-local-tier-handover"
severity: "minor"
category: "lapse"
source: "agent-observation"
found_during: "2026-10-07/08 autonomous drain and v0.13.3 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "the local-tier handover file NEXT.md"
lapsed_at: "2026-10-07T15:55:00Z"
remedy: "A handover writer appends to NEXT.md or writes a new dated handover file, never replaces it unread; add the rule to the LIFEBOAT or CONCURRENCY rules the loader injects on handover prompts."
---

The orchestrating session overwrote the local-tier handover file .abcd/.work.local/NEXT.md with its handover note without reading what it held first; the file is gitignored, so the prior handover could not be recovered from git. The runbook's rule is never to overwrite NEXT.md unread.
