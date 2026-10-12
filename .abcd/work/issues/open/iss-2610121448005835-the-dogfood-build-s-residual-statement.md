---
schema_version: 1
id: "iss-2610121448005835"
slug: "the-dogfood-build-s-residual-statement"
severity: "nitpick"
category: "documentation"
source: "impl-review"
found_during: "security review round 3 of fix/hooks-run-head-in-dogfood-roots, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/dogfood/dogfood.go"
remedy: "state the residual as a rule (the builder and the hooks inherit the account's whole environment), with the toolchain and git variables as examples"
---

The dogfood build's residual statement (AGENTS.md, internal/core/dogfood/dogfood.go's package doc, the brief's dogfood store section) names the account's environment by example: the go toolchain, the module cache, GOENV, PATH and ~/.ssh/config. Git's own inherited environment is the same residual and is not named: GIT_EXEC_PATH, GIT_SSL_CAINFO and SSL_CERT_FILE, https_proxy, and ~/.ssh/known_hosts. A reader can take the list as complete.
