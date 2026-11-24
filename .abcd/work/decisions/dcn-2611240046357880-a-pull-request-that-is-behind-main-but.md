---
id: dcn-2611240046357880
slug: a-pull-request-that-is-behind-main-but
date: 2026-11-24
---
2026-11-24 — A pull request that is behind main but armed through main's merge queue is left behind rather than synced with main: the queue tests the merge result with main, and syncing would restart the pull request's hour of CI for nothing. #1780, #1782 and #1783 each merged from behind on 23 November. Reversed by syncing such a pull request when the queue refuses it for being out of date.
