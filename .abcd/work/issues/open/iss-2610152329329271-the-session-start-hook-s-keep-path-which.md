---
schema_version: 1
id: "iss-2610152329329271"
slug: "the-session-start-hook-s-keep-path-which"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "final review of itd-2610111707085098, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sessionname/register.go"
remedy: "Name a keep failure on stderr as the other naming failures are, once per session; test that a held lock at keep writes one note."
---

The session-start hook's keep path, which records a person's own title as an empty given entry, swallows a lock or write failure silently; the lost record later lets a fork or resume read that title as an inherited abcd name.
