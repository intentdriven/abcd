---
schema_version: 1
id: "iss-2610121542263820"
slug: "the-run-log-s-outage-spans-reach-abcd"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "the build loop's stored-text sweep on fix/sec-build-next-series, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement"
remedy: "Shape-check or mask the outage span text where implement record reads the run log, with a test forging a span."
---

The run log's outage spans reach abcd implement record --json raw: their text comes from the run log the session appends to, so a hand-edited or repository-shipped log carrying a control or bidi rune puts it on the stream.

Security scope (abcd-fixes-what-it-adds): the route does not exist without abcd. The text is echoed by abcd's own --json surface, so abcd added the route and fixes it at the boundary the text enters by; a fix changes nothing about how the repository works.
