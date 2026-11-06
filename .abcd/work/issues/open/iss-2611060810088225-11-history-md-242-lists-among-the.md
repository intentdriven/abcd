---
schema_version: 1
id: "iss-2611060810088225"
slug: "11-history-md-242-lists-among-the"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md"
remedy: "Rewrite the list from the hook's notices, replacing the setup-version item with the after-update line."
---

`11-history.md:242` lists among the session-start notices one for a repo last set up under a different version. No such notice is in the session-start body of `internal/surface/cli/cli.go`, where `meta.setup_version` appears only in the comment at line 2352. The nearest behaviour is the constant after-update line (`internal/surface/cli/update_news.go:140`), printed only when something is pending. The list also omits notices that do exist: git giving no root commit, hooks git does not run, the harness settings notice and the unseen-update notice.
