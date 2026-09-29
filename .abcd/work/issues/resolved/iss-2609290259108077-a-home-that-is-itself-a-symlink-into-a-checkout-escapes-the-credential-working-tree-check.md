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
resolution: "Fixed by 16b7e58da: workingTreeAbove judges the place by its lexical path and again with the home replaced by where it resolves, so a home that is itself a symlink into a checkout is seen as lying inside it; the abcd home is refused there with nothing written, and a pointer under that home is not read. The home is never refused for being a link."
impact: fix
resolved_by:
  commit: "16b7e58da"
---

The credential store's working-tree check misses a home directory that is itself a symlink into a git checkout. workingTreeAbove (internal/core/credential/store.go) climbs the lexical path of ~/.abcd, so when HOME is a link such as ~ -> <repo>/home, the target's parents (<repo>/.git) are never examined; Set with the abcd home then writes credentials.json, a value, inside the checkout, where a commit could carry it. The dotfiles layout AGENTS.md names (a symlinked ~/.abcd) is refused already; this is the account's own home layout, same uid. Judging the tree on the home's resolved path (filepath.EvalSymlinks) as well as its lexical path closes it without refusing a home for being a link. Surfaced by the security review of integ/land-14 (review-integ14 LOW (a)).

## Grounds

- pursued: a home linked into a checkout is refused for the abcd home with nothing written in the checkout and the keychain untouched; a Set that writes credentials.json under such a home, or a refusal of a home linked outside any checkout, would show it wrong
