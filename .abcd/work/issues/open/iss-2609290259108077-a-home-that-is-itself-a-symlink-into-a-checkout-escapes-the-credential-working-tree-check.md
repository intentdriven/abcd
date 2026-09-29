---
schema_version: 1
id: "iss-2609290259108077"
slug: "a-home-that-is-itself-a-symlink-into-a-checkout-escapes-the-credential-working-tree-check"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/credential/store.go"
---

The credential store's working-tree check misses a home directory that is itself a symlink into a git checkout. workingTreeAbove (internal/core/credential/store.go) climbs the lexical path of ~/.abcd, so when HOME is a link such as ~ -> <repo>/home, the target's parents (<repo>/.git) are never examined; Set with the abcd home then writes credentials.json, a value, inside the checkout, where a commit could carry it. The dotfiles layout AGENTS.md names (a symlinked ~/.abcd) is refused already; this is the account's own home layout, same uid. Judging the tree on the home's resolved path (filepath.EvalSymlinks) as well as its lexical path closes it without refusing a home for being a link. Surfaced by the security review of integ/land-14 (review-integ14 LOW (a)).
