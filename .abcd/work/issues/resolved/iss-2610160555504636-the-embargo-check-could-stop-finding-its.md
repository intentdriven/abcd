---
schema_version: 1
id: "iss-2610160555504636"
slug: "the-embargo-check-could-stop-finding-its"
severity: "minor"
category: "security"
source: "security-advisory"
found_during: "embargo intake 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/embargo"
remedy: "Look for the store under every root commit of the default branch, use the one that holds it, refuse when more than one does, and never create a second store"
resolution: "The embargo store is found by reading every store the repository can reach under the home, whatever its root commits, and a second store is never created (#1052). Shipped in v0.15.1."
impact: fix
shipped_in: v0.15.1
resolved_by:
  commit: "f55943168"
---

The embargo check could stop finding its store after a merge brought in an unrelated root commit.

abcd keeps the private advisories it holds under embargo in a store keyed on a root commit of the repository: the first root commit `git rev-list --max-parents=0` lists. A merge that brings an unrelated root commit into the default branch, such as a `git subtree add`, a merge with `--allow-unrelated-histories`, or an empty-diff merge of an orphan commit, can change which root commit is listed first.

After such a merge every checkout looked for the store under a new key, did not find it, and `abcd embargo check` reported nothing under embargo, so outbound text naming an unpublished advisory was no longer refused. The next write to the store created a new, empty store under the new key.

Reaching this needs a merge to the default branch, and someone must still write an embargoed advisory's identifier or title into public text for anything to be disclosed.

Advisory: [GHSA-2c9m-5jgr-5qj5](https://github.com/intentdriven/abcd/security/advisories/GHSA-2c9m-5jgr-5qj5), published 2026-10-17 with v0.15.1 as the patched version. The record was held out of the public tree until then.
