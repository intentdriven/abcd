---
id: dcn-2608300000000013
slug: itd-179-round-3-the-double-quoted
date: 2026-08-30
---
2026-08-30 — itd-179 round-3: the double-quoted frontmatter scalar decoder's canonical home is `internal/core/frontmatter` (`Unquote`), joining `IsNull` there, rather than exporting `capture.unquote`. Rejected: lint importing capture, which is an import cycle — capture's own tests import lint.
