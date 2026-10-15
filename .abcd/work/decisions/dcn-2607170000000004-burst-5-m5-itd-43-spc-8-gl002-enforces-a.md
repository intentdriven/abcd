---
id: dcn-2607170000000004
slug: burst-5-m5-itd-43-spc-8-gl002-enforces-a
date: 2026-07-17
---
2026-07-17 — Burst 5 (M5, itd-43/spc-8): GL002 enforces a deliberate subset
(['epic']) of the glossary's forbidden_synonyms — the others are common
English words whose false-positive rate would sink the gate; each becomes
opt-in via .abcd/record-lint.json as the corpus is readied. Two sweep hits
inside internal self-quotes (itd-48 quoting a working-log line and spc-12's
overview) were swept, not exempted: abcd's own records carry the canonical
word even when quoting themselves. spc-8 stays OPEN and itd-43 planned on
AC3 (spec-review token), blocked by itd-28's maintainer-gated dependency.
