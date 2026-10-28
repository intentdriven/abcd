---
schema_version: 1
id: "iss-2610262035409725"
slug: "the-banlist-chapter-has-write-paths-add"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 589 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/20-banlist.md:12"
remedy: "none (filed automatically)"
finding_ref: cfk-4b4e766a7df83aa3
---

The banlist chapter has write paths (add, remove, migrate); universal patterns lists `banlist` among the read-only renders.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/20-banlist.md:12` — "read-only**; adding and removing are the write paths, and each names its layer"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:90` — "`lint` and its targets, `capture`, `intent`, `guard`, `--version`, `rules`, `spec`, `identity` and `banlist` are read-only renders, and `abcd lint` in particular is documented as performing zero writes"

20-banlist says only bare invocation and listing are read-only, and that adding and removing are the write paths. Its migrate sub-verb also rewrites the private store. 04-universal-patterns lists `banlist` with the verbs it calls read-only renders.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 589.
