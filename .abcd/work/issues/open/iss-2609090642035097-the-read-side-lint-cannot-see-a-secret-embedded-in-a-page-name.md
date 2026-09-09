---
schema_version: 1
id: "iss-2609090642035097"
slug: "the-read-side-lint-cannot-see-a-secret-embedded-in-a-page-name"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "bughunt-triage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory"
related_issues: ["iss-2609020321100138"]
---

The write boundary now refuses a memory page whose filename carries a hard-fail secret, but a store written before that change is not repaired and the read side cannot report it. The MR001 lint scans stored text and the registry bytes, and a token embedded in a page name such as topic_auth_ghp_ is hidden from it by the same missing word boundary the write-side fix had to work around: underscore is a word character, so an anchored token pattern finds no boundary after the type and domain prefix and the span never matches. A legacy dirty filename therefore sits in the committed tree, in index.md, in log.md and in the registry back-link, and no instrument reports it. The write-side fix deliberately left this alone to keep its commit to one record. Fix: give the read-side scan the same component-wise treatment the write side uses, splitting a page name into its type, domain and slug parts before matching rather than relying on a boundary the naming grammar cannot provide. Detector: a store seeded with a page whose name embeds a token must be reported by the read-side lint, and an ordinary store must stay clean. Surfaced by the fix for iss-2609020321100138.
