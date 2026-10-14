---
schema_version: 1
id: "iss-2610121542258778"
slug: "a-capture-disposition-s-title-an-issue"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "the build loop's stored-text sweep on fix/sec-build-next-series, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/eligible.go"
remedy: "Pass a disposition's title and every reader error a capture surface echoes through termsafe.Sanitize where the text is composed, with tests planting U+202E and U+009B in a record's first line and a stored value."
resolution: "A drain disposition's title, and every capture surface's skipped record (its path and the reader's refusal), are masked with termsafe.Sanitize where they are composed, and the reading families' schema_version check masks the value it echoes."
impact: fix
resolved_by:
  commit: "42aecad7777ba889aa26f50905ad9c8ad6f9bac8"
---

A capture disposition's title (an issue record's first body line) reaches abcd drain --json raw, and capture surfaces other than the drain still show the ledger reader's unreadable-record error raw (only the drain's copy is masked, since iss-2610121516552317), so a repository whose issue record body or a stored value carries a control or bidi rune puts it on those streams.

Security scope (abcd-fixes-what-it-adds): the route does not exist without abcd. The text is echoed by abcd's own --json surface, so abcd added the route and fixes it at the boundary the text enters by; a fix changes nothing about how the repository works.
