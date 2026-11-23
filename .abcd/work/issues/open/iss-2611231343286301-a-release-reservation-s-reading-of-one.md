---
schema_version: 1
id: "iss-2611231343286301"
slug: "a-release-reservation-s-reading-of-one"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/withdrawchain.go"
remedy: "find the holder's newest push without reading every commit above it (for example by a ref or trailer the holder's own pushes carry, or by a bounded search that skips unreadable or foreign commits by count without refusing), so the number of commits others push never refuses the holder; test with a line of 600 foreign commits above the holder's push"
---

A release reservation's reading of one holder's record stops at maxChainRead (512) commits of its release branch's first-parent line: with 513 or more commits pushed above the holder's newest push, every act of that holder on the release is refused, and only a hand recovery the refusal names gets past it. The page bound keeps a read within a few MiB, but the commit count of a line another writer can lengthen decides whether the holder can act at all.
